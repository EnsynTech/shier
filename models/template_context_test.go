package models

import (
	"bytes"
	"fmt"
	"strings"

	check "gopkg.in/check.v1"
)

type mockTemplateContext struct {
	URL         string
	FromAddress string
}

func (m mockTemplateContext) getFromAddress() string {
	return m.FromAddress
}

func (m mockTemplateContext) getBaseURL() string {
	return m.URL
}

func (s *ModelsSuite) TestNewTemplateContext(c *check.C) {
	r := Result{
		BaseRecipient: BaseRecipient{
			FirstName: "Foo",
			LastName:  "Bar",
			Email:     "foo@bar.com",
		},
		RId: "1234567",
	}
	ctx := mockTemplateContext{
		URL:         "http://example.com",
		FromAddress: "From Address <from@example.com>",
	}
	expectedURL := fmt.Sprintf("%s?rid=%s", ctx.URL, r.RId)
	expected := PhishingTemplateContext{
		URL:           expectedURL,
		BaseURL:       ctx.URL,
		BaseRecipient: r.BaseRecipient,
		TrackingURL:   fmt.Sprintf("%s/track?rid=%s", ctx.URL, r.RId),
		From:          "From Address",
		RId:           r.RId,
		QR:            qrImgHTML(),
	}
	expected.Tracker = "<img alt='' style='display: none' src='" + expected.TrackingURL + "'/>"
	got, err := NewPhishingTemplateContext(ctx, r.BaseRecipient, r.RId)
	c.Assert(err, check.Equals, nil)
	c.Assert(got.QRPng, check.NotNil)
	c.Assert(bytes.HasPrefix(got.QRPng, []byte{0x89, 0x50, 0x4e, 0x47}), check.Equals, true)
	c.Assert(strings.HasPrefix(got.QRDataURI, "data:image/png;base64,"), check.Equals, true)
	expected.QRDataURI = got.QRDataURI
	got.QRPng = nil
	c.Assert(got, check.DeepEquals, expected)

	rendered, err := ExecuteTemplate(qrTemplateToken, got)
	c.Assert(err, check.Equals, nil)
	c.Assert(rendered, check.Equals, qrImgHTML())
	c.Assert(rendered, check.Matches, ".*cid:qrcode\\.png.*")

	pageImg, err := ExecuteTemplate(`<img src="{{.QRDataURI}}"/>`, got)
	c.Assert(err, check.Equals, nil)
	c.Assert(strings.Contains(pageImg, "data:image/png;base64,"), check.Equals, true)
}

func (s *ModelsSuite) TestTemplateUsesQR(c *check.C) {
	c.Assert(templateUsesQR("Hello {{.QR}}", ""), check.Equals, true)
	c.Assert(templateUsesQR("", "Scan {{.QR}}"), check.Equals, true)
	c.Assert(templateUsesQR("{{.URL}}", "{{.FirstName}}"), check.Equals, false)
}
