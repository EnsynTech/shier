package models

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"strings"
	"time"

	check "gopkg.in/check.v1"
)

func (s *ModelsSuite) TestListScenarioPacks(c *check.C) {
	packs, err := ListScenarioPacks()
	c.Assert(err, check.IsNil)
	c.Assert(len(packs) >= 12, check.Equals, true)
	ids := map[string]bool{}
	validTechniques := map[string]bool{
		TechniqueLink: true, TechniqueAttachment: true, TechniqueQR: true,
		TechniqueSMS: true, TechniqueIM: true, TechniqueCalendar: true, TechniqueReply: true,
	}
	for _, p := range packs {
		ids[p.ID] = true
		c.Assert(p.Title, check.Not(check.Equals), "")
		c.Assert(p.Difficulty >= 1 && p.Difficulty <= 5, check.Equals, true)
		c.Assert(len(p.Industries) > 0, check.Equals, true)
		c.Assert(len(p.Techniques) > 0, check.Equals, true)
		for _, tech := range p.Techniques {
			c.Assert(validTechniques[tech], check.Equals, true)
		}
	}
	c.Assert(ids["calendar_video_meeting"], check.Equals, true)
	c.Assert(ids["office_remote_image"], check.Equals, true)
	c.Assert(ids["office_macro_openurl"], check.Equals, true)
	c.Assert(ids["qr_pdf_verify"], check.Equals, true)
	c.Assert(ids["qr_landing_rescan"], check.Equals, true)
	c.Assert(ids["qr_png_attachment"], check.Equals, true)
	c.Assert(ids["multistep_it_notice"], check.Equals, true)
	c.Assert(ids["reply_meeting_confirm"], check.Equals, true)
	c.Assert(ids["office_excel_remote"], check.Equals, true)
}

func (s *ModelsSuite) TestImportCalendarScenarioPack(c *check.C) {
	result, err := ImportScenarioPack(1, "calendar_video_meeting")
	c.Assert(err, check.IsNil)
	c.Assert(result.TemplateID, check.Not(check.Equals), int64(0))
	c.Assert(result.PageID, check.Not(check.Equals), int64(0))

	t, err := GetTemplate(result.TemplateID, 1)
	c.Assert(err, check.IsNil)
	c.Assert(strings.Contains(t.HTML, "{{.URL}}"), check.Equals, true)
	c.Assert(len(t.Attachments), check.Equals, 1)
	c.Assert(strings.HasSuffix(t.Attachments[0].Name, ".ics"), check.Equals, true)
	c.Assert(t.ScenarioPackId, check.Equals, "calendar_video_meeting")
	c.Assert(t.Difficulty, check.Equals, 2)
	c.Assert(len(t.Techniques) > 0, check.Equals, true)

	raw, err := base64.StdEncoding.DecodeString(t.Attachments[0].Content)
	c.Assert(err, check.IsNil)
	ics := string(raw)
	c.Assert(strings.Contains(ics, "{{.URL}}"), check.Equals, true)
	c.Assert(strings.Contains(ics, "BEGIN:VCALENDAR"), check.Equals, true)
	c.Assert(strings.Contains(ics, "Asia/Shanghai"), check.Equals, true)

	p, err := GetPage(result.PageID, 1)
	c.Assert(err, check.IsNil)
	c.Assert(p.CaptureCredentials, check.Equals, true)

	// Re-import should rename, not conflict
	result2, err := ImportScenarioPack(1, "calendar_video_meeting")
	c.Assert(err, check.IsNil)
	c.Assert(result2.TemplateName, check.Not(check.Equals), result.TemplateName)
	c.Assert(strings.Contains(result2.TemplateName, "导入"), check.Equals, true)
}

func (s *ModelsSuite) TestImportQRScenarioPack(c *check.C) {
	result, err := ImportScenarioPack(1, "qr_account_verify")
	c.Assert(err, check.IsNil)

	t, err := GetTemplate(result.TemplateID, 1)
	c.Assert(err, check.IsNil)
	c.Assert(strings.Contains(t.HTML, "{{.QR}}"), check.Equals, true)
	c.Assert(len(t.Attachments), check.Equals, 0)

	p, err := GetPage(result.PageID, 1)
	c.Assert(err, check.IsNil)
	c.Assert(p.CapturePasswords, check.Equals, true)
}

func (s *ModelsSuite) TestImportOfficeAndQRVariantPacks(c *check.C) {
	office, err := ImportScenarioPack(1, "office_remote_image")
	c.Assert(err, check.IsNil)
	ot, err := GetTemplate(office.TemplateID, 1)
	c.Assert(err, check.IsNil)
	c.Assert(len(ot.Attachments), check.Equals, 1)
	c.Assert(strings.HasSuffix(ot.Attachments[0].Name, ".docx"), check.Equals, true)

	excel, err := ImportScenarioPack(1, "office_excel_remote")
	c.Assert(err, check.IsNil)
	et, err := GetTemplate(excel.TemplateID, 1)
	c.Assert(err, check.IsNil)
	c.Assert(len(et.Attachments), check.Equals, 1)
	c.Assert(strings.HasSuffix(et.Attachments[0].Name, ".xlsx"), check.Equals, true)

	pdfPack, err := ImportScenarioPack(1, "qr_pdf_verify")
	c.Assert(err, check.IsNil)
	pt, err := GetTemplate(pdfPack.TemplateID, 1)
	c.Assert(err, check.IsNil)
	c.Assert(pt.Attachments[0].Name, check.Equals, "qrcode.pdf")

	landing, err := ImportScenarioPack(1, "qr_landing_rescan")
	c.Assert(err, check.IsNil)
	lp, err := GetPage(landing.PageID, 1)
	c.Assert(err, check.IsNil)
	c.Assert(strings.Contains(lp.HTML, "{{.QRDataURI}}"), check.Equals, true)

	pngPack, err := ImportScenarioPack(1, "qr_png_attachment")
	c.Assert(err, check.IsNil)
	pngT, err := GetTemplate(pngPack.TemplateID, 1)
	c.Assert(err, check.IsNil)
	c.Assert(len(pngT.Attachments), check.Equals, 2)
}

func (s *ModelsSuite) TestImportLinkScenarioPacks(c *check.C) {
	packs, err := ListScenarioPacks()
	c.Assert(err, check.IsNil)
	byID := map[string]ScenarioPackSummary{}
	for _, p := range packs {
		byID[p.ID] = p
	}
	passwordPacks := map[string]bool{
		"link_mailbox_verify":    true,
		"link_mailbox_filing":    true,
		"link_sso_login":         true,
		"link_vpn_reauth":        true,
		"link_it_password_reset": true,
		"link_hr_salary":         true,
		"link_invoice_check":     true,
	}
	ids := []string{
		"link_mailbox_verify",
		"link_mailbox_filing",
		"link_sso_login",
		"link_vpn_reauth",
		"link_it_password_reset",
		"link_hr_salary",
		"link_parcel_hold",
		"link_invoice_check",
		"link_tax_settlement",
		"link_festival_gift",
		"link_health_checkup",
		"link_etc_notice",
		"link_travel_draw",
	}
	for _, id := range ids {
		summary, ok := byID[id]
		c.Assert(ok, check.Equals, true)
		c.Assert(summary.Category, check.Equals, "link")
		c.Assert(summary.HasAttachment, check.Equals, false)
		c.Assert(summary.HasQR, check.Equals, false)
		c.Assert(summary.Techniques, check.DeepEquals, []string{"link"})

		result, err := ImportScenarioPack(1, id)
		c.Assert(err, check.IsNil)
		tmpl, err := GetTemplate(result.TemplateID, 1)
		c.Assert(err, check.IsNil)
		c.Assert(strings.Contains(tmpl.HTML, "{{.URL}}"), check.Equals, true)
		c.Assert(strings.Contains(tmpl.HTML, "{{.Tracker}}"), check.Equals, true)
		c.Assert(strings.Contains(tmpl.Text, "{{.URL}}"), check.Equals, true)
		page, err := GetPage(result.PageID, 1)
		c.Assert(err, check.IsNil)
		c.Assert(strings.Contains(strings.ToLower(page.HTML), "<form"), check.Equals, true)
		c.Assert(page.CaptureCredentials, check.Equals, true)
		c.Assert(page.CapturePasswords, check.Equals, passwordPacks[id])
	}
}

func (s *ModelsSuite) TestImportUnknownScenarioPack(c *check.C) {
	_, err := ImportScenarioPack(1, "does_not_exist")
	c.Assert(err, check.Equals, ErrScenarioPackNotFound)
}

func (s *ModelsSuite) TestRefreshICSNextWeekday(c *check.C) {
	// Saturday -> should land on Monday
	sat := time.Date(2026, 9, 19, 10, 0, 0, 0, time.FixedZone("CST", 8*3600))
	ics := refreshICS(`BEGIN:VCALENDAR
DTSTAMP:20200101T000000Z
DTSTART;TZID=Asia/Shanghai:20200101T140000
DTEND;TZID=Asia/Shanghai:20200101T150000
SUMMARY:旧标题
UID:old@shier.local
DESCRIPTION:{{.URL}}
END:VCALENDAR`, "新会议", sat)
	c.Assert(strings.Contains(ics, "SUMMARY:新会议"), check.Equals, true)
	c.Assert(strings.Contains(ics, "DTSTART;TZID=Asia/Shanghai:20260921T140000"), check.Equals, true)
	c.Assert(strings.Contains(ics, "DTEND;TZID=Asia/Shanghai:20260921T150000"), check.Equals, true)
	c.Assert(strings.Contains(ics, "UID:old@shier.local"), check.Equals, false)
}

func (s *ModelsSuite) TestOfficeRemoteImageTrackingURL(c *check.C) {
	result, err := ImportScenarioPack(1, "office_remote_image")
	c.Assert(err, check.IsNil)
	t, err := GetTemplate(result.TemplateID, 1)
	c.Assert(err, check.IsNil)
	c.Assert(len(t.Attachments), check.Equals, 1)

	ptx, err := NewPhishingTemplateContext(mockTemplateContext{
		URL:         "http://phish.example/?rid=abc",
		FromAddress: "From <from@example.com>",
	}, BaseRecipient{Email: "a@b.com", FirstName: "A"}, "abc")
	c.Assert(err, check.IsNil)

	reader, err := t.Attachments[0].ApplyTemplate(ptx)
	c.Assert(err, check.IsNil)
	raw, err := ioutil.ReadAll(reader)
	c.Assert(err, check.IsNil)
	// Unzip and check rels contain TrackingURL value
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	c.Assert(err, check.IsNil)
	found := false
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, ".rels") {
			rc, _ := f.Open()
			b, _ := ioutil.ReadAll(rc)
			rc.Close()
			if strings.Contains(string(b), ptx.TrackingURL) {
				found = true
			}
			c.Assert(strings.Contains(string(b), "{{.TrackingURL}}"), check.Equals, false)
		}
	}
	c.Assert(found, check.Equals, true)
}

func (s *ModelsSuite) TestOfficeMacroURLReplacement(c *check.C) {
	result, err := ImportScenarioPack(1, "office_macro_openurl")
	c.Assert(err, check.IsNil)
	t, err := GetTemplate(result.TemplateID, 1)
	c.Assert(err, check.IsNil)

	ptx, err := NewPhishingTemplateContext(mockTemplateContext{
		URL:         "http://phish.example/?rid=mac1",
		FromAddress: "From <from@example.com>",
	}, BaseRecipient{Email: "a@b.com"}, "mac1")
	c.Assert(err, check.IsNil)

	reader, err := t.Attachments[0].ApplyTemplate(ptx)
	c.Assert(err, check.IsNil)
	raw, err := ioutil.ReadAll(reader)
	c.Assert(err, check.IsNil)
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	c.Assert(err, check.IsNil)
	hasVBA := false
	hasURL := false
	for _, f := range zr.File {
		if f.Name == "word/vbaProject.bin" {
			hasVBA = true
		}
		if f.Name == "word/document.xml" || strings.HasSuffix(f.Name, ".rels") {
			rc, _ := f.Open()
			b, _ := ioutil.ReadAll(rc)
			rc.Close()
			if strings.Contains(string(b), ptx.URL) {
				hasURL = true
			}
			c.Assert(strings.Contains(string(b), "{{.URL}}"), check.Equals, false)
		}
	}
	c.Assert(hasVBA, check.Equals, true)
	c.Assert(hasURL, check.Equals, true)
}
