package models

import (
	"strconv"

	check "gopkg.in/check.v1"
)

func (s *ModelsSuite) TestSendingTrustValidation(c *check.C) {
	invalid := Campaign{
		Name:         "Trust",
		SendingTrust: "not-a-tier",
		Groups:       []Group{{Name: "g"}},
		Template:     Template{Name: "t"},
		Page:         Page{Name: "p"},
		SMTP:         SMTP{Name: "s"},
	}
	c.Assert(invalid.Validate(), check.Equals, ErrInvalidSendingTrust)

	campaign := s.createCampaignDependencies(c)
	campaign.SendingTrust = "  " + SendingTrustCustomerDomain + "  "
	c.Assert(PostCampaign(&campaign, campaign.UserId), check.IsNil)
	got, err := GetCampaign(campaign.Id, campaign.UserId)
	c.Assert(err, check.IsNil)
	c.Assert(got.SendingTrust, check.Equals, SendingTrustCustomerDomain)

	summaries, err := GetCampaignSummaries(campaign.UserId)
	c.Assert(err, check.IsNil)
	c.Assert(len(summaries.Campaigns) > 0, check.Equals, true)
	found := false
	for _, summary := range summaries.Campaigns {
		if summary.Id == campaign.Id {
			found = true
			c.Assert(summary.SendingTrust, check.Equals, SendingTrustCustomerDomain)
		}
	}
	c.Assert(found, check.Equals, true)
}

func (s *ModelsSuite) postReportCampaign(c *check.C, name, trust, pack, status string, clicked int) Campaign {
	group := Group{Name: name + " group", UserId: 1}
	targets := make([]Target, 0, 6)
	for i := 0; i < 5; i++ {
		targets = append(targets, Target{BaseRecipient: BaseRecipient{
			Email:      name + "-fin" + string(rune('a'+i)) + "@example.com",
			FirstName:  "A",
			Department: "财务",
		}})
	}
	targets = append(targets, Target{BaseRecipient: BaseRecipient{
		Email:      name + "-it@example.com",
		FirstName:  "B",
		Department: "IT",
	}})
	group.Targets = targets
	c.Assert(PostGroup(&group), check.IsNil)

	tmpl := Template{Name: name + " template", UserId: 1, Subject: "s", Text: "t", HTML: "h", ScenarioPackId: pack}
	c.Assert(PostTemplate(&tmpl), check.IsNil)
	page := Page{Name: name + " page", UserId: 1, HTML: "<html></html>"}
	c.Assert(PostPage(&page), check.IsNil)
	smtp := SMTP{Name: name + " smtp", UserId: 1, Host: "example.com", FromAddress: "test@test.com"}
	c.Assert(PostSMTP(&smtp), check.IsNil)

	camp := Campaign{
		Name:         name,
		Template:     tmpl,
		Page:         page,
		SMTP:         smtp,
		Groups:       []Group{group},
		URL:          "http://localhost",
		SendingTrust: trust,
	}
	c.Assert(PostCampaign(&camp, 1), check.IsNil)
	c.Assert(db.Model(&camp).UpdateColumns(map[string]interface{}{
		"status":           status,
		"scenario_pack_id": pack,
		"sending_trust":    trust,
	}).Error, check.IsNil)

	var results []Result
	c.Assert(db.Where("campaign_id = ?", camp.Id).Order("email").Find(&results).Error, check.IsNil)
	marked := 0
	for i := range results {
		results[i].Status = EventSent
		if results[i].Department == "财务" && marked < clicked {
			results[i].Status = EventClicked
			marked++
		}
		c.Assert(db.Save(&results[i]).Error, check.IsNil)
	}
	camp.Status = status
	camp.ScenarioPackId = pack
	camp.SendingTrust = trust
	return camp
}

func (s *ModelsSuite) TestCampaignBriefAndCompare(c *check.C) {
	first := s.postReportCampaign(c, "alpha", SendingTrustCustomerDomain, "pack-a", CampaignComplete, 4)
	second := s.postReportCampaign(c, "beta", SendingTrustCustomerDomain, "pack-a", CampaignComplete, 1)
	third := s.postReportCampaign(c, "gamma", SendingTrustLookalike, "pack-a", CampaignInProgress, 2)

	_, err := GetCampaignBrief(1, "")
	c.Assert(err, check.Equals, ErrReportIDsRequired)

	_, err = GetCampaignBrief(1, "999999")
	c.Assert(err, check.Equals, ErrCampaignNotFound)

	ids := strconv.FormatInt(first.Id, 10) + "," + strconv.FormatInt(second.Id, 10)
	brief, err := GetCampaignBrief(1, ids)
	c.Assert(err, check.IsNil)
	c.Assert(brief.Coverage.CampaignCount, check.Equals, 2)
	c.Assert(brief.Coverage.Delivered, check.Equals, int64(12))
	c.Assert(brief.Overall.Clicked, check.Equals, int64(5))
	c.Assert(brief.Overall.PhishProne, check.Equals, float64(5)/float64(12))
	c.Assert(len(brief.WeakestDepartments) >= 1, check.Equals, true)
	c.Assert(brief.WeakestDepartments[0].Department, check.Equals, "财务")
	c.Assert(brief.WeakestDepartments[0].SampleSmall, check.Equals, false)
	c.Assert(brief.Advice.WeakestDepartment, check.Equals, "财务")
	c.Assert(brief.Advice.NoReports, check.Equals, true)
	c.Assert(len(brief.BySendingTrust), check.Equals, 1)
	c.Assert(brief.BySendingTrust[0].SendingTrust, check.Equals, SendingTrustCustomerDomain)

	_, err = GetCampaignBrief(1, ids+","+strconv.FormatInt(third.Id, 10))
	c.Assert(err, check.Equals, ErrBriefNotCompleted)

	cmp, err := GetCampaignCompare(1, ids)
	c.Assert(err, check.IsNil)
	c.Assert(cmp.Narrative, check.Equals, CompareNarrativeRetest)
	c.Assert(len(cmp.SharedDepartments), check.Equals, 2)

	mixed, err := GetCampaignCompare(1, ids+","+strconv.FormatInt(third.Id, 10))
	c.Assert(err, check.IsNil)
	c.Assert(mixed.Narrative, check.Equals, CompareNarrativeExposure)

	_, err = GetCampaignCompare(1, strconv.FormatInt(first.Id, 10))
	c.Assert(err, check.Equals, ErrCompareCount)

	blank := s.postReportCampaign(c, "delta", "", "pack-a", CampaignComplete, 1)
	samePack, err := GetCampaignCompare(1, strconv.FormatInt(first.Id, 10)+","+strconv.FormatInt(blank.Id, 10))
	c.Assert(err, check.IsNil)
	c.Assert(samePack.Narrative, check.Equals, CompareNarrativeExposure)
}
