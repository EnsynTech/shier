package models

import (
	"errors"
	"math"
	"strings"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/sirupsen/logrus"
)

// DeliveryLog is the outbound queue entry for SMS/IM HTTP deliveries.
type DeliveryLog struct {
	Id          int64     `json:"-"`
	UserId      int64     `json:"-"`
	CampaignId  int64     `json:"campaign_id"`
	RId         string    `json:"id"`
	SendDate    time.Time `json:"send_date"`
	SendAttempt int       `json:"send_attempt"`
	Processing  bool      `json:"-"`
	Sent        bool      `json:"sent"`
	Channel     string    `json:"channel"`
	StepIndex   int       `json:"step_index"`

	cachedCampaign *Campaign
}

// CacheCampaign stores the campaign metadata for this delivery.
func (d *DeliveryLog) CacheCampaign(c *Campaign) error {
	if c == nil {
		return errors.New("campaign cannot be nil")
	}
	if c.Id != d.CampaignId {
		return errors.New("campaign id mismatch")
	}
	d.cachedCampaign = c
	return nil
}

// Unlock clears the processing lock.
func (d *DeliveryLog) Unlock() error {
	d.Processing = false
	return db.Save(d).Error
}

// Lock sets the processing lock.
func (d *DeliveryLog) Lock() error {
	d.Processing = true
	return db.Save(d).Error
}

// Backoff schedules a retry with exponential delay.
func (d *DeliveryLog) Backoff(reason error) error {
	r, err := GetResult(d.RId)
	if err != nil {
		return err
	}
	if d.SendAttempt == MaxSendAttempts {
		r.HandleEmailError(ErrMaxSendAttempts)
		return ErrMaxSendAttempts
	}
	d.SendAttempt++
	backoffDuration := math.Pow(2, float64(d.SendAttempt))
	d.SendDate = d.SendDate.Add(time.Minute * time.Duration(backoffDuration))
	err = db.Save(d).Error
	if err != nil {
		return err
	}
	err = r.HandleEmailBackoff(reason, d.SendDate)
	if err != nil {
		return err
	}
	return d.Unlock()
}

// Error permanently fails the delivery and deletes the queue row.
func (d *DeliveryLog) Error(e error) error {
	r, err := GetResult(d.RId)
	if err != nil {
		log.Warn(err)
		return err
	}
	err = r.HandleEmailError(e)
	if err != nil {
		log.Warn(err)
		return err
	}
	return db.Delete(d).Error
}

// Success marks the delivery as sent and updates the result funnel.
func (d *DeliveryLog) Success() error {
	r, err := GetResult(d.RId)
	if err != nil {
		return err
	}
	err = r.HandleMessageSent(d.Channel)
	if err != nil {
		return err
	}
	d.Sent = true
	d.Processing = false
	return db.Save(d).Error
}

// GetQueuedDeliveryLogs returns unsent, unlocked delivery logs due before t.
func GetQueuedDeliveryLogs(t time.Time) ([]*DeliveryLog, error) {
	ds := []*DeliveryLog{}
	err := db.Where("send_date <= ? AND processing = ? AND sent = ?", t, false, false).
		Find(&ds).Error
	if err != nil {
		log.Error(err)
	}
	return ds, err
}

// GetDeliveryLogsByCampaign returns all delivery logs for a campaign.
func GetDeliveryLogsByCampaign(cid int64) ([]*DeliveryLog, error) {
	ds := []*DeliveryLog{}
	err := db.Where("campaign_id = ?", cid).Find(&ds).Error
	return ds, err
}

// LockDeliveryLogs bulk-updates the processing flag.
func LockDeliveryLogs(ds []*DeliveryLog, lock bool) error {
	tx := db.Begin()
	for i := range ds {
		ds[i].Processing = lock
		err := tx.Save(ds[i]).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// UnlockAllDeliveryLogs clears processing locks after an unclean shutdown.
func UnlockAllDeliveryLogs() error {
	return db.Model(&DeliveryLog{}).Where("processing = ?", true).
		Update("processing", false).Error
}

// GetCampaignDeliveryContext loads campaign fields needed for HTTP delivery.
func GetCampaignDeliveryContext(id int64, uid int64) (Campaign, error) {
	c := Campaign{}
	err := db.Where("id = ? AND user_id = ?", id, uid).Find(&c).Error
	if err != nil {
		return c, err
	}
	if c.SendingChannelId != 0 {
		ch, err := GetSendingChannel(c.SendingChannelId, uid)
		if err != nil {
			return c, err
		}
		c.SendingChannel = ch
	}
	err = db.Table("templates").Where("id=?", c.TemplateId).Find(&c.Template).Error
	if err != nil {
		return c, err
	}
	return c, nil
}

// Process sends this delivery log via the configured HTTP channel.
func (d *DeliveryLog) Process() error {
	c := d.cachedCampaign
	var err error
	if c == nil {
		campaign, err := GetCampaignDeliveryContext(d.CampaignId, d.UserId)
		if err != nil {
			return d.Error(err)
		}
		c = &campaign
		d.cachedCampaign = c
	}
	r, err := GetResult(d.RId)
	if err != nil {
		return d.Error(err)
	}
	ch := c.SendingChannel
	if ch.Id == 0 {
		ch, err = GetSendingChannel(c.SendingChannelId, d.UserId)
		if err != nil {
			return d.Error(err)
		}
	}
	ptx, err := NewPhishingTemplateContext(c, r.BaseRecipient, r.RId)
	if err != nil {
		return d.Error(err)
	}
	messageText := c.Template.Text
	if messageText == "" && c.Template.HTML != "" {
		messageText = stripHTMLTags(c.Template.HTML)
	}
	message, err := ExecuteTemplate(messageText, ptx)
	if err != nil {
		log.Warn(err)
	}
	subject, err := ExecuteTemplate(c.Template.Subject, ptx)
	if err != nil {
		log.Warn(err)
	}
	phone := r.Phone
	if phone == "" {
		phone = r.BaseRecipient.Phone
	}
	ctx := DeliveryBodyContext{
		Phone:     phone,
		Email:     r.Email,
		IMUser:    r.Email,
		FirstName: r.FirstName,
		LastName:  r.LastName,
		Position:  r.Position,
		URL:       ptx.URL,
		RId:       r.RId,
		Message:   message,
		Subject:   subject,
	}
	if ch.Channel == ChannelSMS && ctx.Phone == "" {
		return d.Error(ErrPhoneNotSpecified)
	}
	err = ch.Deliver(ctx)
	if err != nil {
		log.WithFields(logrus.Fields{
			"rid":     d.RId,
			"channel": ch.Channel,
		}).Warn(err)
		// Treat most HTTP failures as temporary unless clearly permanent.
		if isPermanentDeliveryError(err) {
			return d.Error(err)
		}
		return d.Backoff(err)
	}
	return d.Success()
}

func isPermanentDeliveryError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "not valid JSON") ||
		strings.Contains(msg, "Phone number not specified") ||
		strings.Contains(msg, "invalid body template")
}

func stripHTMLTags(html string) string {
	var b strings.Builder
	inTag := false
	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
