package models

import (
	"strings"
	"time"

	"github.com/gophish/gomail"
	check "gopkg.in/check.v1"
)

func (s *ModelsSuite) TestPostCampaignMultiStepQueuesOnlyStepZero(c *check.C) {
	campaign := s.createCampaignDependencies(c)
	t2 := Template{Name: "Step Two Template", Subject: "Follow up", HTML: "<html>step2 {{.URL}}</html>", Text: "step2", UserId: 1}
	c.Assert(PostTemplate(&t2), check.IsNil)

	campaign.Name = "Multi step campaign"
	campaign.Steps = []CampaignStep{
		{Template: Template{Name: campaign.Template.Name}, DelayMinutes: 0, Trigger: StepTriggerAfterSent},
		{Template: Template{Name: t2.Name}, DelayMinutes: 30, Trigger: StepTriggerAfterSent},
		{Template: Template{Name: t2.Name}, DelayMinutes: 10, Trigger: StepTriggerOnReply},
	}
	c.Assert(PostCampaign(&campaign, campaign.UserId), check.IsNil)

	steps, err := GetCampaignSteps(campaign.Id)
	c.Assert(err, check.IsNil)
	c.Assert(len(steps), check.Equals, 3)
	c.Assert(steps[1].DelayMinutes, check.Equals, 30)
	c.Assert(steps[2].Trigger, check.Equals, StepTriggerOnReply)

	ms, err := GetMailLogsByCampaign(campaign.Id)
	c.Assert(err, check.IsNil)
	c.Assert(len(ms), check.Equals, len(campaign.Results))
	for _, m := range ms {
		c.Assert(m.StepIndex, check.Equals, 0)
		c.Assert(m.Sent, check.Equals, false)
	}
}

func (s *ModelsSuite) TestMailLogSuccessEnqueuesNextAfterSentStep(c *check.C) {
	campaign := s.createCampaignDependencies(c)
	t2 := Template{Name: "Followup Template", Subject: "Re: notice", HTML: "<html>click {{.URL}}</html>", UserId: 1}
	c.Assert(PostTemplate(&t2), check.IsNil)
	campaign.Name = "Chain send"
	campaign.Steps = []CampaignStep{
		{Template: Template{Name: campaign.Template.Name}, Trigger: StepTriggerAfterSent},
		{Template: Template{Name: t2.Name}, DelayMinutes: 0, Trigger: StepTriggerAfterSent},
	}
	c.Assert(PostCampaign(&campaign, campaign.UserId), check.IsNil)

	result := campaign.Results[0]
	m := &MailLog{}
	c.Assert(db.Where("r_id = ?", result.RId).First(m).Error, check.IsNil)
	c.Assert(m.Success(), check.IsNil)

	ms, err := GetMailLogsByCampaign(campaign.Id)
	c.Assert(err, check.IsNil)
	var step1 *MailLog
	for _, ml := range ms {
		if ml.StepIndex == 1 && ml.RId == result.RId {
			step1 = ml
		}
	}
	c.Assert(step1, check.NotNil)
	c.Assert(step1.Sent, check.Equals, false)
	c.Assert(step1.SendDate.Before(time.Now().UTC().Add(time.Minute)), check.Equals, true)
}

func (s *ModelsSuite) TestMailLogGeneratePersistsMessageIDAndThreading(c *check.C) {
	campaign := s.createCampaignDependencies(c)
	t2 := Template{Name: "Threaded Template", Subject: "Re: hello", HTML: "<html>{{.URL}}</html>", UserId: 1}
	c.Assert(PostTemplate(&t2), check.IsNil)
	campaign.Name = "Thread campaign"
	campaign.Steps = []CampaignStep{
		{Template: Template{Name: campaign.Template.Name}, Trigger: StepTriggerAfterSent},
		{Template: Template{Name: t2.Name}, DelayMinutes: 0, Trigger: StepTriggerAfterSent},
	}
	c.Assert(PostCampaign(&campaign, campaign.UserId), check.IsNil)

	result := campaign.Results[0]
	m0 := &MailLog{}
	c.Assert(db.Where("r_id = ? AND step_index = 0", result.RId).First(m0).Error, check.IsNil)
	msg0 := gomail.NewMessage()
	c.Assert(m0.Generate(msg0), check.IsNil)
	c.Assert(m0.MessageID, check.Not(check.Equals), "")
	c.Assert(m0.Success(), check.IsNil)

	m1 := &MailLog{}
	c.Assert(db.Where("r_id = ? AND step_index = 1", result.RId).First(m1).Error, check.IsNil)
	msg1 := gomail.NewMessage()
	c.Assert(m1.Generate(msg1), check.IsNil)
	c.Assert(msg1.GetHeader("In-Reply-To")[0], check.Equals, m0.MessageID)
	c.Assert(strings.Contains(strings.Join(msg1.GetHeader("References"), " "), m0.MessageID), check.Equals, true)
	subj := msg1.GetHeader("Subject")[0]
	c.Assert(strings.HasPrefix(strings.ToLower(subj), "re:"), check.Equals, true)
}

func (s *ModelsSuite) TestHandleEmailReplyByMessageID(c *check.C) {
	campaign := s.createCampaign(c)
	result := campaign.Results[0]
	m := &MailLog{}
	c.Assert(db.Where("r_id = ?", result.RId).First(m).Error, check.IsNil)
	msg := gomail.NewMessage()
	c.Assert(m.Generate(msg), check.IsNil)
	c.Assert(m.Success(), check.IsNil)

	found, err := GetResultByMessageID(m.MessageID)
	c.Assert(err, check.IsNil)
	c.Assert(found.RId, check.Equals, result.RId)
	c.Assert(found.HandleEmailReply(EventDetails{}), check.IsNil)
	updated, err := GetResult(result.RId)
	c.Assert(err, check.IsNil)
	c.Assert(updated.Replied, check.Equals, true)
	c.Assert(updated.Status, check.Equals, EventSent)
}

func (s *ModelsSuite) TestHandleEmailReplyEnqueuesOnReplyStep(c *check.C) {
	campaign := s.createCampaignDependencies(c)
	t2 := Template{Name: "On Reply Template", Subject: "请确认", HTML: "<html>{{.URL}}</html>", UserId: 1}
	c.Assert(PostTemplate(&t2), check.IsNil)
	campaign.Name = "Reply followup"
	campaign.Steps = []CampaignStep{
		{Template: Template{Name: campaign.Template.Name}, Trigger: StepTriggerAfterSent},
		{Template: Template{Name: t2.Name}, DelayMinutes: 0, Trigger: StepTriggerOnReply},
	}
	c.Assert(PostCampaign(&campaign, campaign.UserId), check.IsNil)
	result := campaign.Results[0]
	m0 := &MailLog{}
	c.Assert(db.Where("r_id = ? AND step_index = 0", result.RId).First(m0).Error, check.IsNil)
	c.Assert(m0.Success(), check.IsNil)

	result, err := GetResult(result.RId)
	c.Assert(err, check.IsNil)
	c.Assert(result.HandleEmailReply(EventDetails{}), check.IsNil)

	m1 := &MailLog{}
	c.Assert(db.Where("r_id = ? AND step_index = 1", result.RId).First(m1).Error, check.IsNil)
	c.Assert(m1.Sent, check.Equals, false)
	// Second reply must not duplicate
	c.Assert(result.HandleEmailReply(EventDetails{}), check.IsNil)
	var count int64
	c.Assert(db.Model(&MailLog{}).Where("r_id = ? AND step_index = 1", result.RId).Count(&count).Error, check.IsNil)
	c.Assert(count, check.Equals, int64(1))
}

func (s *ModelsSuite) TestEnsureReplySubjectAndLastEmailStep(c *check.C) {
	c.Assert(ensureReplySubject("Hello"), check.Equals, "Re: Hello")
	c.Assert(ensureReplySubject("Re: Hello"), check.Equals, "Re: Hello")
	c.Assert(ensureReplySubject("回复: 你好"), check.Equals, "回复: 你好")

	campaign := s.createCampaign(c)
	result := campaign.Results[0]
	m := &MailLog{}
	c.Assert(db.Where("r_id = ?", result.RId).First(m).Error, check.IsNil)
	c.Assert(m.Success(), check.IsNil)
	updated, err := GetResult(result.RId)
	c.Assert(err, check.IsNil)
	c.Assert(updated.LastEmailStep, check.Equals, 0)
}

func (s *ModelsSuite) TestResolvePageForResultUsesStepPage(c *check.C) {
	campaign := s.createCampaignDependencies(c)
	p2 := Page{Name: "Step Two Page", HTML: "<html>step2</html>", UserId: 1}
	c.Assert(PostPage(&p2), check.IsNil)
	t2 := Template{Name: "Step Two Tmpl", Subject: "step2", HTML: "<html>{{.URL}}</html>", UserId: 1}
	c.Assert(PostTemplate(&t2), check.IsNil)
	campaign.Name = "Per step page"
	campaign.Steps = []CampaignStep{
		{Template: Template{Name: campaign.Template.Name}, Page: Page{Name: campaign.Page.Name}, Trigger: StepTriggerAfterSent},
		{Template: Template{Name: t2.Name}, Page: Page{Name: p2.Name}, DelayMinutes: 0, Trigger: StepTriggerAfterSent},
	}
	c.Assert(PostCampaign(&campaign, campaign.UserId), check.IsNil)
	steps, err := GetCampaignSteps(campaign.Id)
	c.Assert(err, check.IsNil)
	c.Assert(steps[1].PageId, check.Equals, p2.Id)

	result := campaign.Results[0]
	m0 := &MailLog{}
	c.Assert(db.Where("r_id = ? AND step_index = 0", result.RId).First(m0).Error, check.IsNil)
	c.Assert(m0.Success(), check.IsNil)
	m1 := &MailLog{}
	c.Assert(db.Where("r_id = ? AND step_index = 1", result.RId).First(m1).Error, check.IsNil)
	c.Assert(m1.Success(), check.IsNil)
	result, err = GetResult(result.RId)
	c.Assert(err, check.IsNil)
	c.Assert(result.LastEmailStep, check.Equals, 1)

	camp, err := GetCampaign(campaign.Id, 1)
	c.Assert(err, check.IsNil)
	page, err := ResolvePageForResult(&camp, &result)
	c.Assert(err, check.IsNil)
	c.Assert(page.Id, check.Equals, p2.Id)
}

func (s *ModelsSuite) TestImportConversationScenarioPacks(c *check.C) {
	ms, err := ImportScenarioPack(1, "multistep_it_notice")
	c.Assert(err, check.IsNil)
	c.Assert(ms.TemplateID, check.Not(check.Equals), int64(0))
	// Second template should exist
	_, err = GetTemplateByName("[多步骤] IT系统升级-确认", 1)
	c.Assert(err, check.IsNil)

	reply, err := ImportScenarioPack(1, "reply_meeting_confirm")
	c.Assert(err, check.IsNil)
	c.Assert(strings.Contains(reply.TemplateName, "会议"), check.Equals, true)
}
