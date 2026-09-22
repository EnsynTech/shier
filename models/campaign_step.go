package models

import (
	"errors"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

// Step trigger values for campaign multi-step emails.
const (
	StepTriggerAfterSent = "after_sent"
	StepTriggerOnReply   = "on_reply"
)

// ErrCampaignStepInvalid indicates a campaign step is missing required fields.
var ErrCampaignStepInvalid = errors.New("each campaign step needs a template")

// ErrCampaignStepTriggerInvalid indicates an unknown step trigger.
var ErrCampaignStepTriggerInvalid = errors.New("campaign step trigger must be after_sent or on_reply")

// CampaignStep is one email in a multi-step campaign sequence.
type CampaignStep struct {
	Id           int64    `json:"-"`
	CampaignId   int64    `json:"-"`
	StepIndex    int      `json:"step_index"`
	TemplateId   int64    `json:"-"`
	Template     Template `json:"template"`
	PageId       int64    `json:"-"`
	Page         Page     `json:"page"`
	DelayMinutes int      `json:"delay_minutes"`
	Trigger      string   `json:"trigger"`
}

// normalizeTrigger returns a valid trigger, defaulting to after_sent.
func normalizeTrigger(t string) (string, error) {
	t = strings.TrimSpace(strings.ToLower(t))
	if t == "" {
		return StepTriggerAfterSent, nil
	}
	switch t {
	case StepTriggerAfterSent, StepTriggerOnReply:
		return t, nil
	default:
		return "", ErrCampaignStepTriggerInvalid
	}
}

// GetCampaignSteps returns ordered steps for a campaign.
func GetCampaignSteps(campaignID int64) ([]CampaignStep, error) {
	steps := []CampaignStep{}
	err := db.Where("campaign_id = ?", campaignID).Order("step_index asc").Find(&steps).Error
	if err != nil {
		return steps, err
	}
	for i := range steps {
		err = db.Table("templates").Where("id = ?", steps[i].TemplateId).Find(&steps[i].Template).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				steps[i].Template = Template{Name: "[Deleted]"}
				continue
			}
			return steps, err
		}
		_ = db.Where("template_id = ?", steps[i].Template.Id).Find(&steps[i].Template.Attachments).Error
		if steps[i].PageId != 0 {
			err = db.Table("pages").Where("id = ?", steps[i].PageId).Find(&steps[i].Page).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				return steps, err
			}
			if err == gorm.ErrRecordNotFound {
				steps[i].Page = Page{Name: "[Deleted]"}
			}
		}
	}
	return steps, nil
}

// GetCampaignStep returns the step at stepIndex for a campaign.
func GetCampaignStep(campaignID int64, stepIndex int) (*CampaignStep, error) {
	step := &CampaignStep{}
	err := db.Where("campaign_id = ? AND step_index = ?", campaignID, stepIndex).First(step).Error
	if err != nil {
		return nil, err
	}
	err = db.Table("templates").Where("id = ?", step.TemplateId).Find(&step.Template).Error
	if err != nil {
		return nil, err
	}
	_ = db.Where("template_id = ?", step.Template.Id).Find(&step.Template.Attachments).Error
	if step.PageId != 0 {
		_ = db.Table("pages").Where("id = ?", step.PageId).Find(&step.Page).Error
	}
	return step, nil
}

// loadTemplateForMailLog resolves the template for a maillog's campaign step.
func loadTemplateForMailLog(campaignID int64, stepIndex int, fallback Template) (Template, error) {
	step, err := GetCampaignStep(campaignID, stepIndex)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fallback, nil
		}
		return Template{}, err
	}
	return step.Template, nil
}

// ResolvePageForResult returns the landing page for a recipient click.
// Prefers the page bound to the last successfully sent campaign step.
func ResolvePageForResult(c *Campaign, r *Result) (Page, error) {
	if r != nil {
		step, err := GetCampaignStep(c.Id, r.LastEmailStep)
		if err == nil && step.PageId != 0 {
			if step.Page.Id != 0 {
				return step.Page, nil
			}
			p, err := GetPage(step.PageId, c.UserId)
			if err == nil {
				return p, nil
			}
		}
	}
	if c.Page.Id != 0 {
		return c.Page, nil
	}
	return GetPage(c.PageId, c.UserId)
}

// highestSentStepIndex returns the max step_index among sent maillogs for a recipient.
func highestSentStepIndex(rid string) (int, error) {
	var m MailLog
	err := db.Where("r_id = ? AND sent = ?", rid, true).Order("step_index desc").First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return -1, nil
	}
	if err != nil {
		return -1, err
	}
	return m.StepIndex, nil
}

// mailLogExistsForStep reports whether a maillog already exists for rid+step.
func mailLogExistsForStep(rid string, stepIndex int) (bool, error) {
	var count int64
	err := db.Model(&MailLog{}).Where("r_id = ? AND step_index = ?", rid, stepIndex).Count(&count).Error
	return count > 0, err
}

// enqueueOnReplySteps queues the next on_reply step after a recipient replies.
func enqueueOnReplySteps(r *Result) error {
	maxSent, err := highestSentStepIndex(r.RId)
	if err != nil {
		return err
	}
	steps, err := GetCampaignSteps(r.CampaignId)
	if err != nil {
		return err
	}
	for _, step := range steps {
		if step.StepIndex <= maxSent {
			continue
		}
		if step.Trigger != StepTriggerOnReply {
			// Only the next contiguous on_reply step after sent mail is queued.
			// If the next step is after_sent, leave it to the send-success path.
			return nil
		}
		exists, err := mailLogExistsForStep(r.RId, step.StepIndex)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}
		delay := time.Duration(step.DelayMinutes) * time.Minute
		next := &MailLog{
			UserId:     r.UserId,
			CampaignId: r.CampaignId,
			RId:        r.RId,
			SendDate:   time.Now().UTC().Add(delay),
			StepIndex:  step.StepIndex,
			Sent:       false,
			Processing: false,
		}
		return db.Save(next).Error
	}
	return nil
}
