package models

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

//go:embed scenariopacks
var scenarioPackFS embed.FS

// Valid scenario technique tags for packs and campaign snapshots.
const (
	TechniqueLink       = "link"
	TechniqueAttachment = "attachment"
	TechniqueQR         = "qr"
	TechniqueSMS        = "sms"
	TechniqueIM         = "im"
	TechniqueCalendar   = "calendar"
	TechniqueReply      = "reply"
)

// ScenarioPackSummary is the list-item view of a built-in scenario pack.
type ScenarioPackSummary struct {
	ID            string   `json:"id"`
	Category      string   `json:"category"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	HasAttachment bool     `json:"has_attachment"`
	HasQR         bool     `json:"has_qr"`
	Difficulty    int      `json:"difficulty"`
	Industries    []string `json:"industries"`
	Techniques    []string `json:"techniques"`
}

// ScenarioPackImportResult is returned after a successful import.
type ScenarioPackImportResult struct {
	TemplateID   int64  `json:"template_id"`
	TemplateName string `json:"template_name"`
	PageID       int64  `json:"page_id"`
	PageName     string `json:"page_name"`
}

type scenarioPackAttachmentMeta struct {
	File string `json:"file"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type scenarioPackTemplateMeta struct {
	Subject      string `json:"subject"`
	TemplateName string `json:"template_name"`
	HTMLFile     string `json:"html"`
	TextFile     string `json:"text"`
}

type scenarioPackMeta struct {
	Subject            string                       `json:"subject"`
	TemplateName       string                       `json:"template_name"`
	PageName           string                       `json:"page_name"`
	CaptureCredentials bool                         `json:"capture_credentials"`
	CapturePasswords   bool                         `json:"capture_passwords"`
	AttachmentName     string                       `json:"attachment_name"`
	AttachmentFile     string                       `json:"attachment_file"`
	AttachmentType     string                       `json:"attachment_type"`
	ICSSummary         string                       `json:"ics_summary"`
	Attachments        []scenarioPackAttachmentMeta `json:"attachments"`
	Templates          []scenarioPackTemplateMeta   `json:"templates"`
}

// ErrScenarioPackNotFound is returned when the pack id is unknown.
var ErrScenarioPackNotFound = errors.New("scenario pack not found")

var (
	icsDTStampRe = regexp.MustCompile(`(?m)^DTSTAMP:.*$`)
	icsDTStartRe = regexp.MustCompile(`(?m)^DTSTART.*$`)
	icsDTEndRe   = regexp.MustCompile(`(?m)^DTEND.*$`)
	icsUIDRe     = regexp.MustCompile(`(?m)^UID:.*$`)
	icsSummaryRe = regexp.MustCompile(`(?m)^SUMMARY:.*$`)
)

// ListScenarioPacks returns metadata for all embedded scenario packs.
func ListScenarioPacks() ([]ScenarioPackSummary, error) {
	raw, err := scenarioPackFS.ReadFile("scenariopacks/manifest.json")
	if err != nil {
		return nil, err
	}
	var packs []ScenarioPackSummary
	if err := json.Unmarshal(raw, &packs); err != nil {
		return nil, err
	}
	return packs, nil
}

// ImportScenarioPack creates a Template and Page for the user from a pack.
func ImportScenarioPack(uid int64, packID string) (*ScenarioPackImportResult, error) {
	summary, err := findPackSummary(packID)
	if err != nil {
		return nil, err
	}
	base := path.Join("scenariopacks", packID)
	metaRaw, err := scenarioPackFS.ReadFile(path.Join(base, "meta.json"))
	if err != nil {
		return nil, err
	}
	var meta scenarioPackMeta
	if err := json.Unmarshal(metaRaw, &meta); err != nil {
		return nil, err
	}
	html, errHTML := scenarioPackFS.ReadFile(path.Join(base, "email.html"))
	text, errText := scenarioPackFS.ReadFile(path.Join(base, "email.txt"))
	if len(meta.Templates) == 0 {
		if errHTML != nil && errText != nil {
			return nil, fmt.Errorf("pack %s: need email.html and/or email.txt", packID)
		}
		if errHTML != nil {
			html = nil
		}
		if errText != nil {
			text = nil
		}
	} else if errHTML != nil {
		html = nil
	}
	pageHTML, err := scenarioPackFS.ReadFile(path.Join(base, "page.html"))
	if err != nil {
		return nil, err
	}

	pageName, err := uniquePageName(uid, meta.PageName)
	if err != nil {
		return nil, err
	}

	tmplSpecs := meta.Templates
	if len(tmplSpecs) == 0 {
		tmplSpecs = []scenarioPackTemplateMeta{{
			Subject:      meta.Subject,
			TemplateName: meta.TemplateName,
			HTMLFile:     "email.html",
			TextFile:     "email.txt",
		}}
	}

	var firstTemplate Template
	for i, spec := range tmplSpecs {
		htmlFile := spec.HTMLFile
		if htmlFile == "" {
			htmlFile = "email.html"
		}
		textFile := spec.TextFile
		if textFile == "" {
			textFile = "email.txt"
		}
		htmlBytes := html
		textBytes := text
		if i > 0 || len(meta.Templates) > 0 {
			htmlBytes, err = scenarioPackFS.ReadFile(path.Join(base, htmlFile))
			if err != nil {
				htmlBytes = nil
			}
			textBytes, err = scenarioPackFS.ReadFile(path.Join(base, textFile))
			if err != nil {
				return nil, err
			}
		}
		if len(htmlBytes) == 0 && len(textBytes) == 0 {
			return nil, fmt.Errorf("pack %s template %q has empty html and text", packID, spec.TemplateName)
		}
		templateName, err := uniqueTemplateName(uid, spec.TemplateName)
		if err != nil {
			return nil, err
		}
		t := Template{
			UserId:         uid,
			Name:           templateName,
			Subject:        spec.Subject,
			HTML:           string(htmlBytes),
			Text:           string(textBytes),
			ModifiedDate:   time.Now().UTC(),
			Attachments:    []Attachment{},
			ScenarioPackId: summary.ID,
			Difficulty:     summary.Difficulty,
			Industries:     StringList(summary.Industries),
			Techniques:     StringList(summary.Techniques),
		}
		// Attachments only on the first template for packs that carry files.
		if i == 0 {
			atts, err := loadPackAttachments(base, summary, meta)
			if err != nil {
				return nil, err
			}
			t.Attachments = atts
		}
		if err := PostTemplate(&t); err != nil {
			return nil, err
		}
		if i == 0 {
			firstTemplate = t
		}
	}

	p := Page{
		UserId:             uid,
		Name:               pageName,
		HTML:               string(pageHTML),
		CaptureCredentials: meta.CaptureCredentials,
		CapturePasswords:   meta.CapturePasswords,
		ModifiedDate:       time.Now().UTC(),
	}
	if err := PostPage(&p); err != nil {
		return nil, err
	}

	return &ScenarioPackImportResult{
		TemplateID:   firstTemplate.Id,
		TemplateName: firstTemplate.Name,
		PageID:       p.Id,
		PageName:     p.Name,
	}, nil
}

func loadPackAttachments(base string, summary *ScenarioPackSummary, meta scenarioPackMeta) ([]Attachment, error) {
	specs := meta.Attachments
	if len(specs) == 0 && summary.HasAttachment {
		file := meta.AttachmentFile
		if file == "" {
			file = "invite.ics"
		}
		name := meta.AttachmentName
		if name == "" {
			name = file
		}
		typ := meta.AttachmentType
		if typ == "" {
			typ = guessAttachmentType(name)
		}
		specs = []scenarioPackAttachmentMeta{{File: file, Name: name, Type: typ}}
	}

	out := make([]Attachment, 0, len(specs))
	for _, spec := range specs {
		raw, err := scenarioPackFS.ReadFile(path.Join(base, spec.File))
		if err != nil {
			return nil, err
		}
		content := raw
		name := spec.Name
		if name == "" {
			name = filepath.Base(spec.File)
		}
		if strings.EqualFold(filepath.Ext(name), ".ics") {
			content = []byte(refreshICS(string(raw), meta.ICSSummary, time.Now()))
		}
		typ := spec.Type
		if typ == "" {
			typ = guessAttachmentType(name)
		}
		out = append(out, Attachment{
			Content: base64.StdEncoding.EncodeToString(content),
			Type:    typ,
			Name:    name,
		})
	}
	return out, nil
}

func guessAttachmentType(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".ics":
		return "text/calendar"
	case ".pdf":
		return "application/pdf"
	case ".png":
		return "image/png"
	case ".txt":
		return "text/plain"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".docm":
		return "application/vnd.ms-word.document.macroEnabled.12"
	default:
		return "application/octet-stream"
	}
}

func findPackSummary(id string) (*ScenarioPackSummary, error) {
	packs, err := ListScenarioPacks()
	if err != nil {
		return nil, err
	}
	for i := range packs {
		if packs[i].ID == id {
			return &packs[i], nil
		}
	}
	return nil, ErrScenarioPackNotFound
}

func uniqueTemplateName(uid int64, base string) (string, error) {
	name := base
	for i := 0; i < 50; i++ {
		_, err := GetTemplateByName(name, uid)
		if err == gorm.ErrRecordNotFound {
			return name, nil
		}
		if err != nil {
			return "", err
		}
		if i == 0 {
			name = base + " (导入)"
		} else {
			name = fmt.Sprintf("%s (导入 %d)", base, i+1)
		}
	}
	return fmt.Sprintf("%s (导入 %d)", base, time.Now().Unix()), nil
}

func uniquePageName(uid int64, base string) (string, error) {
	name := base
	for i := 0; i < 50; i++ {
		_, err := GetPageByName(name, uid)
		if err == gorm.ErrRecordNotFound {
			return name, nil
		}
		if err != nil {
			return "", err
		}
		if i == 0 {
			name = base + " (导入)"
		} else {
			name = fmt.Sprintf("%s (导入 %d)", base, i+1)
		}
	}
	return fmt.Sprintf("%s (导入 %d)", base, time.Now().Unix()), nil
}

// refreshICS rewrites DTSTAMP/DTSTART/DTEND/UID for the next weekday 14:00–15:00 Asia/Shanghai.
func refreshICS(ics, summary string, now time.Time) string {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	start := nextWeekdayAt(now.In(loc), 14, 0)
	end := start.Add(time.Hour)
	stamp := now.UTC()

	out := ics
	out = icsDTStampRe.ReplaceAllString(out, "DTSTAMP:"+stamp.Format("20060102T150405Z"))
	out = icsDTStartRe.ReplaceAllString(out, "DTSTART;TZID=Asia/Shanghai:"+start.Format("20060102T150405"))
	out = icsDTEndRe.ReplaceAllString(out, "DTEND;TZID=Asia/Shanghai:"+end.Format("20060102T150405"))
	out = icsUIDRe.ReplaceAllString(out, fmt.Sprintf("UID:%d@shier.local", stamp.UnixNano()))
	if summary != "" {
		out = icsSummaryRe.ReplaceAllString(out, "SUMMARY:"+escapeICSText(summary))
	}
	return out
}

func nextWeekdayAt(now time.Time, hour, min int) time.Time {
	d := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, now.Location())
	if !d.After(now) {
		d = d.AddDate(0, 0, 1)
	}
	for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		d = d.AddDate(0, 0, 1)
	}
	return d
}

func escapeICSText(s string) string {
	r := strings.NewReplacer("\\", "\\\\", ";", "\\;", ",", "\\,", "\n", "\\n")
	return r.Replace(s)
}
