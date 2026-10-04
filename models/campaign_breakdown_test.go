package models

import (
	check "gopkg.in/check.v1"
)

func (s *ModelsSuite) TestGetCampaignBreakdown(c *check.C) {
	pack, err := ImportScenarioPack(1, "qr_account_verify")
	c.Assert(err, check.IsNil)
	tmpl, err := GetTemplate(pack.TemplateID, 1)
	c.Assert(err, check.IsNil)
	page, err := GetPage(pack.PageID, 1)
	c.Assert(err, check.IsNil)

	group := Group{Name: "Breakdown Group", UserId: 1}
	group.Targets = []Target{
		{BaseRecipient: BaseRecipient{Email: "fin1@example.com", FirstName: "A", Department: "财务"}},
		{BaseRecipient: BaseRecipient{Email: "fin2@example.com", FirstName: "B", Department: "财务"}},
		{BaseRecipient: BaseRecipient{Email: "it1@example.com", FirstName: "C", Department: "IT"}},
		{BaseRecipient: BaseRecipient{Email: "none@example.com", FirstName: "D"}},
	}
	c.Assert(PostGroup(&group), check.IsNil)

	smtp := SMTP{Name: "Breakdown SMTP", UserId: 1, Host: "example.com", FromAddress: "test@test.com"}
	c.Assert(PostSMTP(&smtp), check.IsNil)

	camp := Campaign{
		Name:     "Breakdown Campaign",
		Template: tmpl,
		Page:     page,
		SMTP:     smtp,
		Groups:   []Group{group},
		URL:      "http://localhost",
	}
	c.Assert(PostCampaign(&camp, 1), check.IsNil)
	c.Assert(camp.ScenarioPackId, check.Equals, "qr_account_verify")
	c.Assert(camp.Difficulty > 0, check.Equals, true)
	c.Assert(len(camp.Techniques) > 0, check.Equals, true)

	var results []Result
	c.Assert(db.Where("campaign_id = ?", camp.Id).Find(&results).Error, check.IsNil)
	c.Assert(len(results), check.Equals, 4)
	deptByEmail := map[string]string{}
	for _, r := range results {
		deptByEmail[r.Email] = r.Department
	}
	c.Assert(deptByEmail["fin1@example.com"], check.Equals, "财务")
	c.Assert(deptByEmail["none@example.com"], check.Equals, "")

	for i := range results {
		switch results[i].Email {
		case "fin1@example.com":
			results[i].Status = EventClicked
			c.Assert(db.Save(&results[i]).Error, check.IsNil)
		case "fin2@example.com":
			results[i].Status = EventDataSubmit
			c.Assert(db.Save(&results[i]).Error, check.IsNil)
		case "it1@example.com":
			results[i].Status = EventOpened
			c.Assert(db.Save(&results[i]).Error, check.IsNil)
		}
	}

	bd, err := GetCampaignBreakdown(1)
	c.Assert(err, check.IsNil)
	c.Assert(bd, check.NotNil)

	byDept := map[string]BreakdownBucket{}
	for _, b := range bd.ByDepartment {
		byDept[b.Department] = b
	}
	c.Assert(byDept["财务"].Total, check.Equals, int64(2))
	c.Assert(byDept["财务"].Delivered, check.Equals, int64(2))
	c.Assert(byDept["财务"].Clicked, check.Equals, int64(2))
	c.Assert(byDept["财务"].Submitted, check.Equals, int64(1))
	c.Assert(byDept["财务"].SampleSmall, check.Equals, true)
	c.Assert(byDept["IT"].Total, check.Equals, int64(1))
	c.Assert(byDept["IT"].Clicked, check.Equals, int64(0))
	_, hasEmpty := byDept[BreakdownEmptyDepartment]
	c.Assert(hasEmpty, check.Equals, false)
	c.Assert(bd.ByDepartment[0].Department, check.Equals, "财务")

	foundQR := false
	for _, b := range bd.ByTechnique {
		if b.Technique == TechniqueQR {
			foundQR = true
			c.Assert(b.Total, check.Equals, int64(3))
			c.Assert(b.Clicked, check.Equals, int64(2))
		}
	}
	c.Assert(foundQR, check.Equals, true)

	foundCross := false
	for _, b := range bd.DepartmentXTechnique {
		if b.Department == "财务" && b.Technique == TechniqueQR {
			foundCross = true
			c.Assert(b.Total, check.Equals, int64(2))
			c.Assert(b.Clicked, check.Equals, int64(2))
			c.Assert(b.ClickRate, check.Equals, 1.0)
		}
	}
	c.Assert(foundCross, check.Equals, true)
}
