package models

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/gophish/gomail"
	"github.com/jung-kurt/gofpdf"
	qrcode "github.com/skip2/go-qrcode"
)

// QRCodeCIDFilename is the Content-ID / embed filename used for the
// per-recipient phishing URL QR code image.
const QRCodeCIDFilename = "qrcode.png"

// QRCodePDFFilename is the placeholder attachment name replaced at send time
// with a generated PDF containing the recipient QR code.
const QRCodePDFFilename = "qrcode.pdf"

// QRCodeSize is the pixel size of the generated QR code PNG.
const QRCodeSize = 160

// qrTemplateToken is the Go template variable that inserts the CID QR image.
const qrTemplateToken = "{{.QR}}"

// generateQRCode returns PNG bytes encoding content as a QR code.
func generateQRCode(content string) ([]byte, error) {
	return qrcode.Encode(content, qrcode.Medium, QRCodeSize)
}

// qrDataURI returns a data: URI for the PNG bytes (usable in landing pages).
func qrDataURI(png []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}

// qrImgHTML returns the HTML <img> tag that references the CID-embedded QR PNG.
func qrImgHTML() string {
	return fmt.Sprintf(
		`<img src="cid:%s" alt="" width="%d" height="%d"/>`,
		QRCodeCIDFilename, QRCodeSize, QRCodeSize,
	)
}

// templateUsesQR reports whether the email template references {{.QR}}.
func templateUsesQR(html, text string) bool {
	return strings.Contains(html, qrTemplateToken) || strings.Contains(text, qrTemplateToken)
}

// embedQRCode attaches the QR PNG as an inline CID part when the template uses {{.QR}}.
func embedQRCode(msg *gomail.Message, html, text string, ptx PhishingTemplateContext) {
	if !templateUsesQR(html, text) || len(ptx.QRPng) == 0 {
		return
	}
	png := ptx.QRPng
	msg.Embed(QRCodeCIDFilename, gomail.SetCopyFunc(func(w io.Writer) error {
		_, err := w.Write(png)
		return err
	}))
}

// isQRPlaceholderAttachment reports whether the attachment is a send-time QR placeholder.
func isQRPlaceholderAttachment(name string) bool {
	base := strings.ToLower(filepath.Base(name))
	return base == QRCodeCIDFilename || base == QRCodePDFFilename
}

// generateQRPdf builds a one-page PDF embedding the QR PNG (Latin labels only).
func generateQRPdf(png []byte) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)
	pdf.Cell(40, 10, "Scan to verify")
	pdf.Ln(12)
	pdf.SetFont("Helvetica", "", 11)
	pdf.MultiCell(0, 6, "Use your phone camera to scan the QR code below.", "", "", false)
	pdf.Ln(6)

	opt := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
	pdf.RegisterImageOptionsReader("qr", opt, bytes.NewReader(png))
	if pdf.Error() != nil {
		return nil, pdf.Error()
	}
	pdf.ImageOptions("qr", 20, 40, 50, 50, false, opt, 0, "")
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// resolveQRAttachmentContent returns bytes for placeholder QR attachments at send time.
func resolveQRAttachmentContent(name string, ptx PhishingTemplateContext) ([]byte, error) {
	base := strings.ToLower(filepath.Base(name))
	switch base {
	case QRCodeCIDFilename:
		if len(ptx.QRPng) == 0 {
			return nil, fmt.Errorf("missing QR PNG for attachment %s", name)
		}
		return ptx.QRPng, nil
	case QRCodePDFFilename:
		if len(ptx.QRPng) == 0 {
			return nil, fmt.Errorf("missing QR PNG for attachment %s", name)
		}
		return generateQRPdf(ptx.QRPng)
	default:
		return nil, fmt.Errorf("not a QR placeholder: %s", name)
	}
}
