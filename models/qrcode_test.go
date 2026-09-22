package models

import (
	"bytes"
	"strings"

	check "gopkg.in/check.v1"
)

func (s *ModelsSuite) TestResolveQRAttachmentPNG(c *check.C) {
	ptx, err := NewPhishingTemplateContext(mockTemplateContext{
		URL:         "http://example.com",
		FromAddress: "From <from@example.com>",
	}, BaseRecipient{Email: "a@b.com"}, "rid123")
	c.Assert(err, check.IsNil)

	png, err := resolveQRAttachmentContent(QRCodeCIDFilename, ptx)
	c.Assert(err, check.IsNil)
	c.Assert(bytes.HasPrefix(png, []byte{0x89, 0x50, 0x4e, 0x47}), check.Equals, true)
	c.Assert(isQRPlaceholderAttachment("qrcode.png"), check.Equals, true)
	c.Assert(isQRPlaceholderAttachment("QRCODE.PDF"), check.Equals, true)
	c.Assert(isQRPlaceholderAttachment("other.pdf"), check.Equals, false)
}

func (s *ModelsSuite) TestResolveQRAttachmentPDF(c *check.C) {
	ptx, err := NewPhishingTemplateContext(mockTemplateContext{
		URL:         "http://example.com",
		FromAddress: "From <from@example.com>",
	}, BaseRecipient{Email: "a@b.com"}, "rid123")
	c.Assert(err, check.IsNil)

	pdf, err := resolveQRAttachmentContent(QRCodePDFFilename, ptx)
	c.Assert(err, check.IsNil)
	c.Assert(bytes.HasPrefix(pdf, []byte("%PDF")), check.Equals, true)
	c.Assert(len(pdf) > 100, check.Equals, true)
}

func (s *ModelsSuite) TestGenerateQRPdf(c *check.C) {
	png, err := generateQRCode("http://example.com/?rid=1")
	c.Assert(err, check.IsNil)
	pdf, err := generateQRPdf(png)
	c.Assert(err, check.IsNil)
	c.Assert(strings.HasPrefix(string(pdf), "%PDF"), check.Equals, true)
}
