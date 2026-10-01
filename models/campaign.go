package models

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/webhook"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Campaign is a struct representing a created campaign
type Campaign struct {
	Id               int64          `json:"id"`
	UserId           int64          `json:"-"`
	Name             string         `json:"name" sql:"not null"`
	CreatedDate      time.Time      `json:"created_date"`
	LaunchDate       time.Time      `json:"launch_date"`
	SendByDate       time.Time      `json:"send_by_date"`
	CompletedDate    time.Time      `json:"completed_date"`
	TemplateId       int64          `json:"-"`
	Template         Template       `json:"template"`
	PageId           int64          `json:"-"`
	Page             Page           `json:"page"`
	Status           string         `json:"status"`
	Results          []Result       `json:"results,omitempty"`
	Groups           []Group        `json:"groups,omitempty"`
	Events           []Event        `json:"timeline,omitempty"`
	Steps            []CampaignStep `json:"steps,omitempty"`
	SMTPId           int64          `json:"-"`
	SMTP             SMTP           `json:"smtp"`
	Channel          string         `json:"channel"`
	SendingChannelId int64          `json:"sending_channel_id"`
	SendingChannel   SendingChannel `json:"sending_channel"`
	URL              string         `json:"url"`
	ScenarioPackId   string         `json:"scenario_pack_id"`
	Difficulty       int            `json:"difficulty"`
	Industries       StringList     `json:"industries" gorm:"type:text"`
	Techniques       StringList     `json:"techniques" gorm:"type:text"`
	DeletedAt        *time.Time     `json:"deleted_at,omitempty" sql:"index"`
}

// CampaignResults is a struct representing the results from a campaign
type CampaignResults struct {
	Id             int64      `json:"id"`
	Name           string     `json:"name"`
	Status         string     `json:"status"`
	Results        []Result   `json:"results,omitempty"`
	Events         []Event    `json:"timeline,omitempty"`
	ScenarioPackId string     `json:"scenario_pack_id"`
	Difficulty     int        `json:"difficulty"`
	Industries     StringList `json:"industries"`
	Techniques     StringList `json:"techniques"`
}

// CampaignSummaries is a struct representing the overview of campaigns
type CampaignSummaries struct {
	Total     int64             `json:"total"`
	Campaigns []CampaignSummary `json:"campaigns"`
}

// CampaignSummary is a struct representing the overview of a single camaign
type CampaignSummary struct {
	Id            int64         `json:"id"`
	CreatedDate   time.Time     `json:"created_date"`
	LaunchDate    time.Time     `json:"launch_date"`
	SendByDate    time.Time     `json:"send_by_date"`
	CompletedDate time.Time     `json:"completed_date"`
	Status        string        `json:"status"`
	Name          string        `json:"name"`
	Stats         CampaignStats `json:"stats"`
	DeletedAt     *time.Time    `json:"deleted_at,omitempty"`
}

// CampaignStats is a struct representing the statistics for a single campaign
type CampaignStats struct {
	Total         int64 `json:"total"`
	EmailsSent    int64 `json:"sent"`
	OpenedEmail   int64 `json:"opened"`
	ClickedLink   int64 `json:"clicked"`
	SubmittedData int64 `json:"submitted_data"`
	EmailReported int64 `json:"email_reported"`
	EmailReplied  int64 `json:"email_replied"`
	Error         int64 `json:"error"`
}

// Event contains the fields for an event
// that occurs during the campaign
type Event struct {
	Id         int64     `json:"-"`
	CampaignId int64     `json:"campaign_id"`
	Email      string    `json:"email"`
	Time       time.Time `json:"time"`
	Message    string    `json:"message"`
	Details    string    `json:"details"`
}

// EventDetails is a struct that wraps common attributes we want to store
// in an event
type EventDetails struct {
	Payload   url.Values        `json:"payload"`
	Browser   map[string]string `json:"browser"`
	From      string            `json:"from,omitempty"`
	Subject   string            `json:"subject,omitempty"`
	MessageID string            `json:"message_id,omitempty"`
	Source    string            `json:"source,omitempty"`
}

// EventError is a struct that wraps an error that occurs when sending an
// email to a recipient
type EventError struct {
	Error string `json:"error"`
}

// ErrCampaignNameNotSpecified indicates there was no template given by the user
var ErrCampaignNameNotSpecified = errors.New("Campaign name not specified")

// ErrGroupNotSpecified indicates there was no template given by the user
var ErrGroupNotSpecified = errors.New("No groups specified")

// ErrTemplateNotSpecified indicates there was no template given by the user
var ErrTemplateNotSpecified = errors.New("No email template specified")

// ErrPageNotSpecified indicates a landing page was not provided for the campaign
var ErrPageNotSpecified = errors.New("No landing page specified")

// ErrSMTPNotSpecified indicates a sending profile was not provided for the campaign
var ErrSMTPNotSpecified = errors.New("No sending profile specified")

// ErrTemplateNotFound indicates the template specified does not exist in the database
var ErrTemplateNotFound = errors.New("Template not found")

// ErrGroupNotFound indicates a group specified by the user does not exist in the database
var ErrGroupNotFound = errors.New("Group not found")

// ErrPageNotFound indicates a page specified by the user does not exist in the database
var ErrPageNotFound = errors.New("Page not found")

// ErrSMTPNotFound indicates a sending profile specified by the user does not exist in the database
var ErrSMTPNotFound = errors.New("Sending profile not found")

// ErrInvalidSendByDate indicates that the user specified a send by date that occurs before the
// launch date
var ErrInvalidSendByDate = errors.New("The launch date must be before the \"send emails by\" date")

// ErrCampaignNotDraft indicates launch was requested for a non-draft campaign.
var ErrCampaignNotDraft = errors.New("campaign is not in draft status")

// ErrCampaignNotFound indicates the campaign does not exist for this user.
var ErrCampaignNotFound = errors.New("campaign not found")

// ErrPurgeConfirmationRequired indicates a purge request lacked confirm=purge.
var ErrPurgeConfirmationRequired = errors.New("purge_confirmation_required")

// PurgeConfirmValue is the request body value required to permanently delete a campaign.
const PurgeConfirmValue = "purge"

// RecipientParameter is the URL parameter that points to the result ID for a recipient.
const RecipientParameter = "rid"

// NormalizeChannel returns a supported campaign channel, defaulting to email.
func NormalizeChannel(ch string) string {
	switch strings.ToLower(strings.TrimSpace(ch)) {
	case ChannelSMS:
		return ChannelSMS
	case ChannelIM:
		return ChannelIM
	default:
		return ChannelEmail
	}
}

// IsHTTPChannel reports whether the campaign uses SMS or IM delivery.
func (c *Campaign) IsHTTPChannel() bool {
	ch := NormalizeChannel(c.Channel)
	return ch == ChannelSMS || ch == ChannelIM
}

// Validate checks to make sure there are no invalid fields in a submitted campaign
func (c *Campaign) Validate() error {
	c.Channel = NormalizeChannel(c.Channel)
	if c.Name == "" {
		return ErrCampaignNameNotSpecified
	}
	if len(c.Groups) == 0 {
		return ErrGroupNotSpecified
	}
	if len(c.Steps) > 0 {
		for i := range c.Steps {
			if c.Steps[i].Template.Name == "" {
				return ErrCampaignStepInvalid
			}
			trig, err := normalizeTrigger(c.Steps[i].Trigger)
			if err != nil {
				return err
			}
			c.Steps[i].Trigger = trig
			c.Steps[i].StepIndex = i
			if i == 0 {
				c.Steps[i].DelayMinutes = 0
			}
		}
		// Align top-level template with step 0 for backward-compatible clients.
		c.Template.Name = c.Steps[0].Template.Name
	} else if c.Template.Name == "" {
		return ErrTemplateNotSpecified
	}
	if c.Page.Name == "" {
		return ErrPageNotSpecified
	}
	if c.IsHTTPChannel() {
		if c.SendingChannel.Name == "" && c.SendingChannelId == 0 {
			return ErrSendingChannelNotSpecified
		}
	} else if c.SMTP.Name == "" {
		return ErrSMTPNotSpecified
	}
	if !c.SendByDate.IsZero() && !c.LaunchDate.IsZero() && c.SendByDate.Before(c.LaunchDate) {
		return ErrInvalidSendByDate
	}
	return nil
}

// UpdateStatus changes the campaign status appropriately
func (c *Campaign) UpdateStatus(s string) error {
	// This could be made simpler, but I think there's a bug in gorm
	return db.Table("campaigns").Where("id=?", c.Id).Update("status", s).Error
}

// AddEvent creates a new campaign event in the database
func AddEvent(e *Event, campaignID int64) error {
	e.CampaignId = campaignID
	e.Time = time.Now().UTC()

	whs, err := GetActiveWebhooks()
	if err == nil {
		whEndPoints := []webhook.EndPoint{}
		for _, wh := range whs {
			whEndPoints = append(whEndPoints, webhook.EndPoint{
				URL:    wh.URL,
				Secret: wh.Secret,
			})
		}
		webhook.SendAll(whEndPoints, e)
	} else {
		log.Errorf("error getting active webhooks: %v", err)
	}

	return db.Save(e).Error
}

// getDetails retrieves the related attributes of the campaign
// from the database. If the Events and the Results are not available,
// an error is returned. Otherwise, the attribute name is set to [Deleted],
// indicating the user deleted the attribute (template, smtp, etc.)
func (c *Campaign) getDetails() error {
	err := db.Model(c).Related(&c.Results).Error
	if err != nil {
		log.Warnf("%s: results not found for campaign", err)
		return err
	}
	err = db.Model(c).Related(&c.Events).Error
	if err != nil {
		log.Warnf("%s: events not found for campaign", err)
		return err
	}
	err = db.Table("templates").Where("id=?", c.TemplateId).Find(&c.Template).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		c.Template = Template{Name: "[Deleted]"}
		log.Warnf("%s: template not found for campaign", err)
	}
	err = db.Where("template_id=?", c.Template.Id).Find(&c.Template.Attachments).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Warn(err)
		return err
	}
	err = db.Table("pages").Where("id=?", c.PageId).Find(&c.Page).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		c.Page = Page{Name: "[Deleted]"}
		log.Warnf("%s: page not found for campaign", err)
	}
	if c.SMTPId != 0 {
		err = db.Table("smtp").Where("id=?", c.SMTPId).Find(&c.SMTP).Error
		if err != nil {
			// Check if the SMTP was deleted
			if err != gorm.ErrRecordNotFound {
				return err
			}
			c.SMTP = SMTP{Name: "[Deleted]"}
			log.Warnf("%s: sending profile not found for campaign", err)
		}
		err = db.Where("smtp_id=?", c.SMTP.Id).Find(&c.SMTP.Headers).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			log.Warn(err)
			return err
		}
	}
	if c.SendingChannelId != 0 {
		ch, chErr := GetSendingChannel(c.SendingChannelId, c.UserId)
		if chErr != nil {
			if chErr != ErrSendingChannelNotFound && chErr != gorm.ErrRecordNotFound {
				return chErr
			}
			c.SendingChannel = SendingChannel{Name: "[Deleted]"}
			log.Warnf("%s: sending channel not found for campaign", chErr)
		} else {
			ch.MaskSecret()
			c.SendingChannel = ch
		}
	}
	c.Channel = NormalizeChannel(c.Channel)
	steps, err := GetCampaignSteps(c.Id)
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Warn(err)
		return err
	}
	c.Steps = steps
	return nil
}

// getBaseURL returns the Campaign's configured URL.
// This is used to implement the TemplateContext interface.
func (c *Campaign) getBaseURL() string {
	return c.URL
}

// getFromAddress returns the Campaign's configured SMTP "From" address.
// This is used to implement the TemplateContext interface.
func (c *Campaign) getFromAddress() string {
	if c.SMTP.FromAddress != "" {
		return c.SMTP.FromAddress
	}
	// SMS/IM campaigns may have no SMTP profile; provide a parseable placeholder.
	return "noreply@localhost"
}

// generateSendDate creates a sendDate
func (c *Campaign) generateSendDate(idx int, totalRecipients int) time.Time {
	// If no send date is specified, just return the launch date
	if c.SendByDate.IsZero() || c.SendByDate.Equal(c.LaunchDate) {
		return c.LaunchDate
	}
	// Otherwise, we can calculate the range of minutes to send emails
	// (since we only poll once per minute)
	totalMinutes := c.SendByDate.Sub(c.LaunchDate).Minutes()

	// Next, we can determine how many minutes should elapse between emails
	minutesPerEmail := totalMinutes / float64(totalRecipients)

	// Then, we can calculate the offset for this particular email
	offset := int(minutesPerEmail * float64(idx))

	// Finally, we can just add this offset to the launch date to determine
	// when the email should be sent
	return c.LaunchDate.Add(time.Duration(offset) * time.Minute)
}

// getCampaignStats returns a CampaignStats object for the campaign with the given campaign ID.
// It also backfills numbers as appropriate with a running total, so that the values are aggregated.
func getCampaignStats(cid int64) (CampaignStats, error) {
	s := CampaignStats{}
	query := db.Table("results").Where("campaign_id = ?", cid)
	err := query.Count(&s.Total).Error
	if err != nil {
		return s, err
	}
	query.Where("status=?", EventDataSubmit).Count(&s.SubmittedData)
	if err != nil {
		return s, err
	}
	query.Where("status=?", EventClicked).Count(&s.ClickedLink)
	if err != nil {
		return s, err
	}
	query.Where("reported=?", true).Count(&s.EmailReported)
	if err != nil {
		return s, err
	}
	query = db.Table("results").Where("campaign_id = ?", cid)
	query.Where("replied=?", true).Count(&s.EmailReplied)
	if err != nil {
		return s, err
	}
	// Every submitted data event implies they clicked the link
	s.ClickedLink += s.SubmittedData
	err = query.Where("status=?", EventOpened).Count(&s.OpenedEmail).Error
	if err != nil {
		return s, err
	}
	// Every clicked link event implies they opened the email
	s.OpenedEmail += s.ClickedLink
	err = query.Where("status=?", EventSent).Count(&s.EmailsSent).Error
	if err != nil {
		return s, err
	}
	// Every opened email event implies the email was sent
	s.EmailsSent += s.OpenedEmail
	err = query.Where("status=?", Error).Count(&s.Error).Error
	return s, err
}

func whereCampaignNotDeleted(query *gorm.DB) *gorm.DB {
	return query.Where("deleted_at IS NULL")
}

func whereJoinedCampaignNotDeleted(query *gorm.DB) *gorm.DB {
	return query.Where("campaigns.deleted_at IS NULL")
}

// GetCampaigns returns the campaigns owned by the given user.
func GetCampaigns(uid int64) ([]Campaign, error) {
	cs := []Campaign{}
	err := db.Where("user_id = ?", uid).Find(&cs).Error
	if err != nil {
		log.Error(err)
	}
	for i := range cs {
		err = cs[i].getDetails()
		if err != nil {
			log.Error(err)
		}
	}
	return cs, err
}

// GetCampaignSummaries gets the summary objects for all the campaigns
// owned by the current user
func GetCampaignSummaries(uid int64) (CampaignSummaries, error) {
	overview := CampaignSummaries{}
	cs := []CampaignSummary{}
	// Get the basic campaign information
	query := whereCampaignNotDeleted(db.Table("campaigns").Where("user_id = ?", uid))
	query = query.Select("id, name, created_date, launch_date, send_by_date, completed_date, status")
	err := query.Scan(&cs).Error
	if err != nil {
		log.Error(err)
		return overview, err
	}
	for i := range cs {
		s, err := getCampaignStats(cs[i].Id)
		if err != nil {
			log.Error(err)
			return overview, err
		}
		cs[i].Stats = s
	}
	overview.Total = int64(len(cs))
	overview.Campaigns = cs
	return overview, nil
}

// GetCampaignSummary gets the summary object for a campaign specified by the campaign ID
func GetCampaignSummary(id int64, uid int64) (CampaignSummary, error) {
	cs := CampaignSummary{}
	query := whereCampaignNotDeleted(db.Table("campaigns").Where("user_id = ? AND id = ?", uid, id))
	query = query.Select("id, name, created_date, launch_date, send_by_date, completed_date, status")
	err := query.Scan(&cs).Error
	if err != nil {
		log.Error(err)
		return cs, err
	}
	if cs.Id == 0 {
		return cs, gorm.ErrRecordNotFound
	}
	s, err := getCampaignStats(cs.Id)
	if err != nil {
		log.Error(err)
		return cs, err
	}
	cs.Stats = s
	return cs, nil
}

// GetCampaignMailContext returns a campaign object with just the relevant
// data needed to generate and send emails. This includes the top-level
// metadata, the template, and the sending profile.
//
// This should only ever be used if you specifically want this lightweight
// context, since it returns a non-standard campaign object.
// ref: #1726
func GetCampaignMailContext(id int64, uid int64) (Campaign, error) {
	c := Campaign{}
	err := db.Where("id = ?", id).Where("user_id = ?", uid).Find(&c).Error
	if err != nil {
		return c, err
	}
	err = db.Table("smtp").Where("id=?", c.SMTPId).Find(&c.SMTP).Error
	if err != nil {
		return c, err
	}
	err = db.Where("smtp_id=?", c.SMTP.Id).Find(&c.SMTP.Headers).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return c, err
	}
	err = db.Table("templates").Where("id=?", c.TemplateId).Find(&c.Template).Error
	if err != nil {
		return c, err
	}
	err = db.Where("template_id=?", c.Template.Id).Find(&c.Template.Attachments).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return c, err
	}
	return c, nil
}

// GetCampaign returns the campaign, if it exists, specified by the given id and user_id.
func GetCampaign(id int64, uid int64) (Campaign, error) {
	c := Campaign{}
	err := db.Where("id = ?", id).Where("user_id = ?", uid).Find(&c).Error
	if err != nil {
		log.Errorf("%s: campaign not found", err)
		return c, err
	}
	err = c.getDetails()
	return c, err
}

// GetCampaignResults returns just the campaign results for the given campaign
func GetCampaignResults(id int64, uid int64) (CampaignResults, error) {
	cr := CampaignResults{}
	err := whereCampaignNotDeleted(db.Table("campaigns").Where("id=? and user_id=?", id, uid)).Find(&cr).Error
	if err != nil {
		log.WithFields(logrus.Fields{
			"campaign_id": id,
			"error":       err,
		}).Error(err)
		return cr, err
	}
	err = db.Table("results").Where("campaign_id=? and user_id=?", cr.Id, uid).Find(&cr.Results).Error
	if err != nil {
		log.Errorf("%s: results not found for campaign", err)
		return cr, err
	}
	err = db.Table("events").Where("campaign_id=?", cr.Id).Find(&cr.Events).Error
	if err != nil {
		log.Errorf("%s: events not found for campaign", err)
		return cr, err
	}
	return cr, err
}

// GetQueuedCampaigns returns the campaigns that are queued up for this given minute
func GetQueuedCampaigns(t time.Time) ([]Campaign, error) {
	cs := []Campaign{}
	err := db.Where("launch_date <= ?", t).
		Where("status = ?", CampaignQueued).
		Where("deleted_at IS NULL").Find(&cs).Error
	if err != nil {
		log.Error(err)
	}
	log.Infof("Found %d Campaigns to run\n", len(cs))
	for i := range cs {
		err = cs[i].getDetails()
		if err != nil {
			log.Error(err)
		}
	}
	return cs, err
}

// PostCampaign inserts a campaign and all associated records into the database.
func PostCampaign(c *Campaign, uid int64) error {
	return postCampaign(c, uid, false)
}

// PostCampaignDraft inserts a campaign that will not be sent until LaunchDraftCampaign.
func PostCampaignDraft(c *Campaign, uid int64) error {
	return postCampaign(c, uid, true)
}

// postCampaign inserts a campaign. When draft is true, status stays Draft and
// no mail/delivery entries are marked processing.
func postCampaign(c *Campaign, uid int64, draft bool) error {
	err := c.Validate()
	if err != nil {
		return err
	}
	// Fill in the details
	c.UserId = uid
	c.CreatedDate = time.Now().UTC()
	c.CompletedDate = time.Time{}
	c.Status = CampaignQueued
	if c.LaunchDate.IsZero() {
		c.LaunchDate = c.CreatedDate
	} else {
		c.LaunchDate = c.LaunchDate.UTC()
	}
	if !c.SendByDate.IsZero() {
		c.SendByDate = c.SendByDate.UTC()
	}
	if draft {
		c.Status = CampaignDraft
	} else if c.LaunchDate.Before(c.CreatedDate) || c.LaunchDate.Equal(c.CreatedDate) {
		c.Status = CampaignInProgress
	}
	// Check to make sure all the groups already exist
	// Also, later we'll need to know the total number of recipients (counting
	// duplicates is ok for now), so we'll do that here to save a loop.
	totalRecipients := 0
	for i, g := range c.Groups {
		c.Groups[i], err = GetGroupByName(g.Name, uid)
		if err == gorm.ErrRecordNotFound {
			log.WithFields(logrus.Fields{
				"group": g.Name,
			}).Error("Group does not exist")
			return ErrGroupNotFound
		} else if err != nil {
			log.Error(err)
			return err
		}
		totalRecipients += len(c.Groups[i].Targets)
	}
	// Check to make sure the template exists
	t, err := GetTemplateByName(c.Template.Name, uid)
	if err == gorm.ErrRecordNotFound {
		log.WithFields(logrus.Fields{
			"template": c.Template.Name,
		}).Error("Template does not exist")
		return ErrTemplateNotFound
	} else if err != nil {
		log.Error(err)
		return err
	}
	c.Template = t
	c.TemplateId = t.Id
	c.ScenarioPackId = t.ScenarioPackId
	c.Difficulty = t.Difficulty
	c.Industries = t.Industries
	c.Techniques = t.Techniques
	// Check to make sure the page exists
	p, err := GetPageByName(c.Page.Name, uid)
	if err == gorm.ErrRecordNotFound {
		log.WithFields(logrus.Fields{
			"page": c.Page.Name,
		}).Error("Page does not exist")
		return ErrPageNotFound
	} else if err != nil {
		log.Error(err)
		return err
	}
	c.Page = p
	c.PageId = p.Id
	c.Channel = NormalizeChannel(c.Channel)
	if c.IsHTTPChannel() {
		var ch SendingChannel
		if c.SendingChannel.Name != "" {
			ch, err = GetSendingChannelByName(c.SendingChannel.Name, uid)
		} else if c.SendingChannelId != 0 {
			ch, err = GetSendingChannel(c.SendingChannelId, uid)
		} else {
			return ErrSendingChannelNotSpecified
		}
		if err == ErrSendingChannelNotFound || err == gorm.ErrRecordNotFound {
			log.WithFields(logrus.Fields{
				"channel": c.SendingChannel.Name,
			}).Error("Sending channel does not exist")
			return ErrSendingChannelNotFound
		} else if err != nil {
			log.Error(err)
			return err
		}
		if ch.Channel != c.Channel {
			return fmt.Errorf("sending channel type %s does not match campaign channel %s", ch.Channel, c.Channel)
		}
		c.SendingChannel = ch
		c.SendingChannelId = ch.Id
		c.SMTPId = 0
		if strings.TrimSpace(c.Template.Text) == "" && strings.TrimSpace(c.Template.HTML) == "" {
			return ErrTemplateTextRequired
		}
	} else {
		// Check to make sure the sending profile exists
		s, err := GetSMTPByName(c.SMTP.Name, uid)
		if err == gorm.ErrRecordNotFound {
			log.WithFields(logrus.Fields{
				"smtp": c.SMTP.Name,
			}).Error("Sending profile does not exist")
			return ErrSMTPNotFound
		} else if err != nil {
			log.Error(err)
			return err
		}
		c.SMTP = s
		c.SMTPId = s.Id
		c.SendingChannelId = 0
	}
	// Persist steps separately after the campaign row (avoid GORM association double-insert).
	pendingSteps := c.Steps
	c.Steps = nil
	// Insert into the DB
	err = db.Save(c).Error
	if err != nil {
		log.Error(err)
		return err
	}

	// Persist multi-step definitions (or a single implicit step from template).
	stepsToSave := pendingSteps
	if len(stepsToSave) == 0 {
		stepsToSave = []CampaignStep{{
			StepIndex:    0,
			TemplateId:   c.TemplateId,
			Template:     c.Template,
			PageId:       c.PageId,
			Page:         c.Page,
			DelayMinutes: 0,
			Trigger:      StepTriggerAfterSent,
		}}
	} else {
		for i := range stepsToSave {
			st, err := GetTemplateByName(stepsToSave[i].Template.Name, uid)
			if err == gorm.ErrRecordNotFound {
				return ErrTemplateNotFound
			} else if err != nil {
				return err
			}
			stepsToSave[i].Template = st
			stepsToSave[i].TemplateId = st.Id
			stepsToSave[i].CampaignId = c.Id
			stepsToSave[i].StepIndex = i
			trig, err := normalizeTrigger(stepsToSave[i].Trigger)
			if err != nil {
				return err
			}
			stepsToSave[i].Trigger = trig
			if stepsToSave[i].Page.Name != "" {
				pg, err := GetPageByName(stepsToSave[i].Page.Name, uid)
				if err == gorm.ErrRecordNotFound {
					return ErrPageNotFound
				} else if err != nil {
					return err
				}
				stepsToSave[i].Page = pg
				stepsToSave[i].PageId = pg.Id
			} else if i == 0 {
				stepsToSave[i].PageId = c.PageId
				stepsToSave[i].Page = c.Page
			}
			if i == 0 {
				stepsToSave[i].DelayMinutes = 0
				c.Template = st
				c.TemplateId = st.Id
				c.ScenarioPackId = st.ScenarioPackId
				c.Difficulty = st.Difficulty
				c.Industries = st.Industries
				c.Techniques = st.Techniques
			}
		}
		// Ensure campaign template_id and tag snapshot match step 0 after resolving steps.
		err = db.Model(c).UpdateColumns(map[string]interface{}{
			"template_id":      c.TemplateId,
			"scenario_pack_id": c.ScenarioPackId,
			"difficulty":       c.Difficulty,
			"industries":       c.Industries,
			"techniques":       c.Techniques,
		}).Error
		if err != nil {
			return err
		}
	}
	for i := range stepsToSave {
		stepsToSave[i].CampaignId = c.Id
		stepsToSave[i].Id = 0
		err = db.Save(&stepsToSave[i]).Error
		if err != nil {
			log.Error(err)
			return err
		}
	}
	c.Steps = stepsToSave

	err = AddEvent(&Event{Message: "Campaign Created"}, c.Id)
	if err != nil {
		log.Error(err)
	}
	if draft {
		err = AddEvent(&Event{Message: "Campaign Saved as Draft"}, c.Id)
		if err != nil {
			log.Error(err)
		}
	}
	// Insert all the results
	resultMap := make(map[string]bool)
	recipientIndex := 0
	tx := db.Begin()
	for _, g := range c.Groups {
		// Insert a result for each target in the group
		for _, t := range g.Targets {
			// Remove duplicate results - we should only
			// send emails to unique email addresses.
			if _, ok := resultMap[t.Email]; ok {
				continue
			}
			resultMap[t.Email] = true
			sendDate := c.generateSendDate(recipientIndex, totalRecipients)
			r := &Result{
				BaseRecipient: BaseRecipient{
					Email:      t.Email,
					Position:   t.Position,
					FirstName:  t.FirstName,
					LastName:   t.LastName,
					Phone:      t.Phone,
					Department: t.Department,
				},
				Status:       StatusScheduled,
				CampaignId:   c.Id,
				UserId:       c.UserId,
				SendDate:     sendDate,
				Reported:     false,
				Replied:      false,
				ModifiedDate: c.CreatedDate,
			}
			if c.Channel == ChannelSMS && strings.TrimSpace(t.Phone) == "" {
				tx.Rollback()
				return ErrPhoneNotSpecified
			}
			err = r.GenerateId(tx)
			if err != nil {
				log.Error(err)
				tx.Rollback()
				return err
			}
			processing := false
			if !draft && (r.SendDate.Before(c.CreatedDate) || r.SendDate.Equal(c.CreatedDate)) {
				r.Status = StatusSending
				processing = true
			}
			err = tx.Save(r).Error
			if err != nil {
				log.WithFields(logrus.Fields{
					"email": t.Email,
				}).Errorf("error creating result: %v", err)
				tx.Rollback()
				return err
			}
			c.Results = append(c.Results, *r)
			log.WithFields(logrus.Fields{
				"email":     r.Email,
				"send_date": sendDate,
			}).Debug("creating delivery queue entry")
			// Only enqueue step 0 when its trigger is after_sent (default).
			if stepsToSave[0].Trigger == StepTriggerAfterSent {
				if c.IsHTTPChannel() {
					d := &DeliveryLog{
						UserId:     c.UserId,
						CampaignId: c.Id,
						RId:        r.RId,
						SendDate:   sendDate,
						Processing: processing,
						StepIndex:  0,
						Sent:       false,
						Channel:    c.Channel,
					}
					err = tx.Save(d).Error
					if err != nil {
						log.WithFields(logrus.Fields{
							"email": t.Email,
						}).Errorf("error creating delivery_log entry: %v", err)
						tx.Rollback()
						return err
					}
				} else {
					m := &MailLog{
						UserId:     c.UserId,
						CampaignId: c.Id,
						RId:        r.RId,
						SendDate:   sendDate,
						Processing: processing,
						StepIndex:  0,
						Sent:       false,
					}
					err = tx.Save(m).Error
					if err != nil {
						log.WithFields(logrus.Fields{
							"email": t.Email,
						}).Errorf("error creating maillog entry: %v", err)
						tx.Rollback()
						return err
					}
				}
			}
			recipientIndex++
		}
	}
	return tx.Commit().Error
}

// LaunchDraftCampaign promotes a draft campaign to Queued or In progress and
// marks due mail/delivery logs for processing. Caller must invoke the worker
// when the returned campaign status is CampaignInProgress.
func LaunchDraftCampaign(id int64, uid int64) (*Campaign, error) {
	c, err := GetCampaign(id, uid)
	if err == gorm.ErrRecordNotFound {
		return nil, ErrCampaignNotFound
	} else if err != nil {
		return nil, err
	}
	if c.Status != CampaignDraft {
		return nil, ErrCampaignNotDraft
	}
	now := time.Now().UTC()
	// Immediate launch if unset or not in the future relative to now.
	if c.LaunchDate.IsZero() || !c.LaunchDate.After(now) {
		c.LaunchDate = now
		c.Status = CampaignInProgress
	} else {
		c.Status = CampaignQueued
	}
	err = db.Model(&Campaign{}).Where("id=? AND user_id=?", id, uid).
		UpdateColumns(map[string]interface{}{
			"status":      c.Status,
			"launch_date": c.LaunchDate,
		}).Error
	if err != nil {
		return nil, err
	}
	// Activate results / queue entries that are due now.
	var results []Result
	err = db.Where("campaign_id=?", id).Find(&results).Error
	if err != nil {
		return nil, err
	}
	for _, r := range results {
		if c.Status == CampaignInProgress && (r.SendDate.Before(now) || r.SendDate.Equal(now) || r.SendDate.Before(c.LaunchDate) || r.SendDate.Equal(c.LaunchDate)) {
			err = db.Model(&Result{}).Where("id=?", r.Id).UpdateColumn("status", StatusSending).Error
			if err != nil {
				return nil, err
			}
			err = db.Model(&MailLog{}).Where("r_id=?", r.RId).UpdateColumn("processing", true).Error
			if err != nil {
				return nil, err
			}
			err = db.Model(&DeliveryLog{}).Where("r_id=?", r.RId).UpdateColumn("processing", true).Error
			if err != nil {
				return nil, err
			}
		}
	}
	_ = AddEvent(&Event{Message: "Campaign Launched"}, c.Id)
	c, err = GetCampaign(id, uid)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// DeleteCampaign soft-deletes the campaign. Sending is stopped, the campaign is
// marked complete, and deleted_at is set so it only appears in the trash.
func DeleteCampaign(id int64) error {
	log.WithFields(logrus.Fields{
		"campaign_id": id,
	}).Info("Soft-deleting campaign")
	c := Campaign{}
	err := db.Where("id = ?", id).Find(&c).Error
	if err != nil {
		log.Error(err)
		return err
	}
	err = db.Where("campaign_id=?", id).Delete(&MailLog{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	err = db.Where("campaign_id=?", id).Delete(&DeliveryLog{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	if c.Status != CampaignComplete {
		c.CompletedDate = time.Now().UTC()
		c.Status = CampaignComplete
		err = db.Model(&Campaign{}).Where("id=?", id).
			Select([]string{"completed_date", "status"}).UpdateColumns(&c).Error
		if err != nil {
			log.Error(err)
			return err
		}
	}
	err = db.Delete(&Campaign{Id: id}).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// GetDeletedCampaignSummaries returns trash summaries for campaigns owned by uid.
func GetDeletedCampaignSummaries(uid int64) (CampaignSummaries, error) {
	overview := CampaignSummaries{}
	cs := []CampaignSummary{}
	query := db.Unscoped().Table("campaigns").Where("user_id = ? AND deleted_at IS NOT NULL", uid)
	query = query.Select("id, name, created_date, launch_date, send_by_date, completed_date, status, deleted_at")
	err := query.Scan(&cs).Error
	if err != nil {
		log.Error(err)
		return overview, err
	}
	for i := range cs {
		s, err := getCampaignStats(cs[i].Id)
		if err != nil {
			log.Error(err)
			return overview, err
		}
		cs[i].Stats = s
	}
	overview.Total = int64(len(cs))
	overview.Campaigns = cs
	return overview, nil
}

// RestoreCampaign clears deleted_at. Sending is not resumed.
func RestoreCampaign(id int64, uid int64) error {
	c := Campaign{}
	err := db.Unscoped().Where("id = ? AND user_id = ?", id, uid).Find(&c).Error
	if err != nil {
		return err
	}
	if c.DeletedAt == nil {
		return ErrCampaignNotFound
	}
	log.WithFields(logrus.Fields{
		"campaign_id": id,
	}).Info("Restoring campaign from trash")
	return db.Unscoped().Model(&Campaign{}).Where("id = ? AND user_id = ?", id, uid).
		UpdateColumn("deleted_at", gorm.Expr("NULL")).Error
}

// PurgeCampaign permanently deletes a campaign that is already in the trash.
func PurgeCampaign(id int64, uid int64) error {
	c := Campaign{}
	err := db.Unscoped().Where("id = ? AND user_id = ?", id, uid).Find(&c).Error
	if err != nil {
		return err
	}
	if c.DeletedAt == nil {
		return ErrCampaignNotFound
	}
	return purgeCampaignRows(id)
}

// DeleteAllCampaignsForUser permanently deletes every campaign owned by uid,
// including those already in the trash. Used when deleting a user account.
func DeleteAllCampaignsForUser(uid int64) error {
	ids := []int64{}
	err := db.Unscoped().Model(&Campaign{}).Where("user_id = ?", uid).Pluck("id", &ids).Error
	if err != nil {
		return err
	}
	for _, id := range ids {
		err = purgeCampaignRows(id)
		if err != nil {
			return err
		}
	}
	return nil
}

func purgeCampaignRows(id int64) error {
	log.WithFields(logrus.Fields{
		"campaign_id": id,
	}).Info("Purging campaign")
	err := db.Where("campaign_id=?", id).Delete(&Result{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	err = db.Where("campaign_id=?", id).Delete(&Event{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	err = db.Where("campaign_id=?", id).Delete(&MailLog{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	err = db.Where("campaign_id=?", id).Delete(&DeliveryLog{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	err = db.Where("campaign_id=?", id).Delete(&CampaignStep{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	err = db.Unscoped().Delete(&Campaign{Id: id}).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// CompleteCampaign effectively "ends" a campaign.
// Any future emails clicked will return a simple "404" page.
func CompleteCampaign(id int64, uid int64) error {
	log.WithFields(logrus.Fields{
		"campaign_id": id,
	}).Info("Marking campaign as complete")
	c, err := GetCampaign(id, uid)
	if err != nil {
		return err
	}
	// Delete any maillogs still set to be sent out, preventing future emails
	err = db.Where("campaign_id=?", id).Delete(&MailLog{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	err = db.Where("campaign_id=?", id).Delete(&DeliveryLog{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	// Don't overwrite original completed time
	if c.Status == CampaignComplete {
		return nil
	}
	// Mark the campaign as complete
	c.CompletedDate = time.Now().UTC()
	c.Status = CampaignComplete
	err = db.Model(&Campaign{}).Where("id=? and user_id=?", id, uid).
		Select([]string{"completed_date", "status"}).UpdateColumns(&c).Error
	if err != nil {
		log.Error(err)
	}
	return err
}
