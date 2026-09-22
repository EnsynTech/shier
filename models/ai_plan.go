package models

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const launchTokenTTL = 15 * time.Minute

var (
	// ErrLaunchTokenInvalid is returned when confirm_token is missing/expired/used.
	ErrLaunchTokenInvalid = errors.New("launch_confirmation_required")
	// ErrAIPlanNotFound indicates unknown plan id for this user.
	ErrAIPlanNotFound = errors.New("ai plan not found")
	// ErrAIPlanValidation is a generic validation failure for AI plans.
	ErrAIPlanValidation = errors.New("ai plan validation failed")
)

// AIPlan persists an orchestration resource graph created via /api/ai/plans.
type AIPlan struct {
	Id              int64     `json:"id"`
	UserId          int64     `json:"-"`
	IdempotencyKey  string    `json:"idempotency_key"`
	ClientLabel     string    `json:"client_label"`
	ProjectLabel    string    `json:"project_label"`
	Status          string    `json:"status"`
	GroupId         int64     `json:"group_id"`
	TemplateId      int64     `json:"template_id"`
	PageId          int64     `json:"page_id"`
	CampaignId      int64     `json:"campaign_id"`
	SMTPId          int64     `json:"smtp_id,omitempty"`
	SendingChannelId int64    `json:"sending_channel_id,omitempty"`
	ScenarioPackId  string    `json:"scenario_pack_id,omitempty"`
	Warnings        string    `json:"warnings"`
	CreatedAt       time.Time `json:"created_at"`
}

// AIPlanTarget is one recipient in an AI plan request.
type AIPlanTarget struct {
	Email      string `json:"email"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Position   string `json:"position"`
	Phone      string `json:"phone"`
	Department string `json:"department"`
}

// AIPlanChannel selects SMTP or sending channel by id.
type AIPlanChannel struct {
	SMTPId           int64 `json:"smtp_id"`
	SendingChannelId int64 `json:"sending_channel_id"`
}

// AIPlanSchedule optional launch window.
type AIPlanSchedule struct {
	LaunchDate time.Time `json:"launch_date"`
	SendByDate time.Time `json:"send_by_date"`
}

// AIPlanRequest is the intent payload for dry-run / create.
type AIPlanRequest struct {
	ClientLabel    string         `json:"client_label"`
	ProjectLabel   string         `json:"project_label"`
	Name           string         `json:"name"`
	URL            string         `json:"url"`
	ScenarioPackId string         `json:"scenario_pack_id"`
	TemplateId     int64          `json:"template_id"`
	PageId         int64          `json:"page_id"`
	GroupId        int64          `json:"group_id"`
	Targets        []AIPlanTarget `json:"targets"`
	Channel        AIPlanChannel  `json:"channel"`
	Schedule       AIPlanSchedule `json:"schedule"`
}

// AIPlanResponse is returned to API clients.
type AIPlanResponse struct {
	Id               int64    `json:"id"`
	Status           string   `json:"status"`
	ClientLabel      string   `json:"client_label"`
	ProjectLabel     string   `json:"project_label"`
	GroupId          int64    `json:"group_id"`
	TemplateId       int64    `json:"template_id"`
	PageId           int64    `json:"page_id"`
	CampaignId       int64    `json:"campaign_id"`
	SMTPId           int64    `json:"smtp_id,omitempty"`
	SendingChannelId int64    `json:"sending_channel_id,omitempty"`
	ScenarioPackId   string   `json:"scenario_pack_id,omitempty"`
	Warnings         []string `json:"warnings"`
	CreatedAt        time.Time `json:"created_at"`
}

type launchTokenEntry struct {
	CampaignId int64
	UserId     int64
	Expires    time.Time
}

var (
	launchTokens   = map[string]launchTokenEntry{}
	launchTokensMu sync.Mutex
)

// IssueLaunchConfirmToken creates a one-time token bound to campaign+user.
func IssueLaunchConfirmToken(campaignId, userId int64) (string, time.Time, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", time.Time{}, err
	}
	token := hex.EncodeToString(b)
	exp := time.Now().UTC().Add(launchTokenTTL)
	launchTokensMu.Lock()
	defer launchTokensMu.Unlock()
	// Drop expired entries opportunistically.
	now := time.Now().UTC()
	for k, v := range launchTokens {
		if v.Expires.Before(now) {
			delete(launchTokens, k)
		}
	}
	launchTokens[token] = launchTokenEntry{
		CampaignId: campaignId,
		UserId:     userId,
		Expires:    exp,
	}
	return token, exp, nil
}

// ConsumeLaunchConfirmToken validates and single-uses a token.
func ConsumeLaunchConfirmToken(token string, campaignId, userId int64) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrLaunchTokenInvalid
	}
	launchTokensMu.Lock()
	defer launchTokensMu.Unlock()
	ent, ok := launchTokens[token]
	if !ok || ent.Expires.Before(time.Now().UTC()) {
		delete(launchTokens, token)
		return ErrLaunchTokenInvalid
	}
	if ent.CampaignId != campaignId || ent.UserId != userId {
		return ErrLaunchTokenInvalid
	}
	delete(launchTokens, token)
	return nil
}

// GetAIPlan returns a plan owned by uid.
func GetAIPlan(id, uid int64) (*AIPlan, error) {
	p := &AIPlan{}
	err := db.Where("id=? AND user_id=?", id, uid).First(p).Error
	if err != nil {
		return nil, ErrAIPlanNotFound
	}
	return p, nil
}

// GetAIPlanByIdempotency returns an existing plan for the key, if any.
func GetAIPlanByIdempotency(uid int64, key string) (*AIPlan, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil
	}
	p := &AIPlan{}
	err := db.Where("user_id=? AND idempotency_key=?", uid, key).First(p).Error
	if err != nil {
		return nil, nil
	}
	return p, nil
}

func (p *AIPlan) toResponse() AIPlanResponse {
	var warnings []string
	if strings.TrimSpace(p.Warnings) != "" {
		warnings = strings.Split(p.Warnings, "\n")
	}
	if warnings == nil {
		warnings = []string{}
	}
	return AIPlanResponse{
		Id:               p.Id,
		Status:           p.Status,
		ClientLabel:      p.ClientLabel,
		ProjectLabel:     p.ProjectLabel,
		GroupId:          p.GroupId,
		TemplateId:       p.TemplateId,
		PageId:           p.PageId,
		CampaignId:       p.CampaignId,
		SMTPId:           p.SMTPId,
		SendingChannelId: p.SendingChannelId,
		ScenarioPackId:   p.ScenarioPackId,
		Warnings:         warnings,
		CreatedAt:        p.CreatedAt,
	}
}

// ValidateAIPlanRequest checks intent without writing.
func ValidateAIPlanRequest(uid int64, req *AIPlanRequest) []string {
	var errs []string
	if strings.TrimSpace(req.URL) == "" {
		errs = append(errs, "url is required")
	}
	if req.Channel.SMTPId == 0 && req.Channel.SendingChannelId == 0 {
		errs = append(errs, "channel.smtp_id or channel.sending_channel_id is required")
	}
	if req.Channel.SMTPId != 0 && req.Channel.SendingChannelId != 0 {
		errs = append(errs, "specify only one of channel.smtp_id or channel.sending_channel_id")
	}
	hasTargets := len(req.Targets) > 0
	if req.GroupId == 0 && !hasTargets {
		errs = append(errs, "targets or group_id is required")
	}
	if req.ScenarioPackId == "" && (req.TemplateId == 0 || req.PageId == 0) {
		errs = append(errs, "scenario_pack_id or both template_id and page_id are required")
	}
	for i, t := range req.Targets {
		if strings.TrimSpace(t.Email) == "" {
			errs = append(errs, fmt.Sprintf("targets[%d].email is required", i))
		}
	}
	if req.Channel.SMTPId != 0 {
		if db != nil {
			if _, err := GetSMTP(req.Channel.SMTPId, uid); err != nil {
				errs = append(errs, "smtp profile not found")
			}
		}
	}
	if req.Channel.SendingChannelId != 0 {
		if db != nil {
			if _, err := GetSendingChannel(req.Channel.SendingChannelId, uid); err != nil {
				errs = append(errs, "sending channel not found")
			}
		}
	}
	if req.GroupId != 0 {
		if db != nil {
			if _, err := GetGroup(req.GroupId, uid); err != nil {
				errs = append(errs, "group not found")
			}
		}
	}
	if req.TemplateId != 0 {
		if db != nil {
			if _, err := GetTemplate(req.TemplateId, uid); err != nil {
				errs = append(errs, "template not found")
			}
		}
	}
	if req.PageId != 0 {
		if db != nil {
			if _, err := GetPage(req.PageId, uid); err != nil {
				errs = append(errs, "page not found")
			}
		}
	}
	if req.ScenarioPackId != "" {
		packs, err := ListScenarioPacks()
		if err != nil {
			errs = append(errs, "cannot list scenario packs")
		} else {
			found := false
			for _, p := range packs {
				if p.ID == req.ScenarioPackId {
					found = true
					break
				}
			}
			if !found {
				errs = append(errs, "scenario_pack_id not found")
			}
		}
	}
	return errs
}

// CreateAIPlan builds group/template/page/draft campaign and stores the plan.
func CreateAIPlan(uid int64, req *AIPlanRequest, idempotencyKey string) (*AIPlanResponse, error) {
	if existing, _ := GetAIPlanByIdempotency(uid, idempotencyKey); existing != nil {
		resp := existing.toResponse()
		return &resp, nil
	}
	if errs := ValidateAIPlanRequest(uid, req); len(errs) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrAIPlanValidation, strings.Join(errs, "; "))
	}

	var warnings []string
	templateID := req.TemplateId
	pageID := req.PageId
	packID := strings.TrimSpace(req.ScenarioPackId)

	if packID != "" {
		imp, err := ImportScenarioPack(uid, packID)
		if err != nil {
			return nil, err
		}
		templateID = imp.TemplateID
		pageID = imp.PageID
		warnings = append(warnings, fmt.Sprintf("imported scenario pack %s as template=%d page=%d", packID, templateID, pageID))
	}

	tmpl, err := GetTemplate(templateID, uid)
	if err != nil {
		return nil, err
	}
	page, err := GetPage(pageID, uid)
	if err != nil {
		return nil, err
	}

	groupID := req.GroupId
	if groupID == 0 {
		gName := fmt.Sprintf("ai-%s-%d", sanitizeLabel(req.ClientLabel), time.Now().UnixNano())
		if strings.TrimSpace(req.ProjectLabel) != "" {
			gName = fmt.Sprintf("ai-%s-%s-%d", sanitizeLabel(req.ClientLabel), sanitizeLabel(req.ProjectLabel), time.Now().UnixNano())
		}
		g := Group{
			Name:         gName,
			UserId:       uid,
			ModifiedDate: time.Now().UTC(),
			Targets:      make([]Target, 0, len(req.Targets)),
		}
		for _, t := range req.Targets {
			dept := t.Department
			if dept == "" {
				dept = t.Position
			}
			g.Targets = append(g.Targets, Target{BaseRecipient: BaseRecipient{
				Email:      strings.TrimSpace(t.Email),
				FirstName:  t.FirstName,
				LastName:   t.LastName,
				Position:   t.Position,
				Phone:      t.Phone,
				Department: dept,
			}})
		}
		if err := PostGroup(&g); err != nil {
			return nil, err
		}
		groupID = g.Id
	}
	group, err := GetGroup(groupID, uid)
	if err != nil {
		return nil, err
	}

	campName := strings.TrimSpace(req.Name)
	if campName == "" {
		campName = fmt.Sprintf("%s %s", strings.TrimSpace(req.ClientLabel), time.Now().Format("2006-01-02 15:04"))
		campName = strings.TrimSpace(campName)
		if campName == "" {
			campName = fmt.Sprintf("AI Plan %d", time.Now().Unix())
		}
	}

	c := Campaign{
		Name:       campName,
		URL:        strings.TrimSpace(req.URL),
		LaunchDate: req.Schedule.LaunchDate,
		SendByDate: req.Schedule.SendByDate,
		Groups:     []Group{{Name: group.Name}},
		Template:   Template{Name: tmpl.Name},
		Page:       Page{Name: page.Name},
	}
	if req.Channel.SendingChannelId != 0 {
		ch, err := GetSendingChannel(req.Channel.SendingChannelId, uid)
		if err != nil {
			return nil, err
		}
		c.Channel = ch.Channel
		c.SendingChannel = SendingChannel{Name: ch.Name}
		c.SendingChannelId = ch.Id
	} else {
		smtp, err := GetSMTP(req.Channel.SMTPId, uid)
		if err != nil {
			return nil, err
		}
		c.Channel = ChannelEmail
		c.SMTP = SMTP{Name: smtp.Name}
		c.SMTPId = smtp.Id
	}

	if err := PostCampaignDraft(&c, uid); err != nil {
		return nil, err
	}

	plan := &AIPlan{
		UserId:           uid,
		IdempotencyKey:   strings.TrimSpace(idempotencyKey),
		ClientLabel:      strings.TrimSpace(req.ClientLabel),
		ProjectLabel:     strings.TrimSpace(req.ProjectLabel),
		Status:           "draft",
		GroupId:          groupID,
		TemplateId:       templateID,
		PageId:           pageID,
		CampaignId:       c.Id,
		SMTPId:           c.SMTPId,
		SendingChannelId: c.SendingChannelId,
		ScenarioPackId:   packID,
		Warnings:         strings.Join(warnings, "\n"),
		CreatedAt:        time.Now().UTC(),
	}
	if err := db.Save(plan).Error; err != nil {
		return nil, err
	}
	resp := plan.toResponse()
	return &resp, nil
}

// MarkAIPlansLaunched updates plan rows pointing at the campaign.
func MarkAIPlansLaunched(uid, campaignId int64) error {
	return db.Model(&AIPlan{}).
		Where("user_id=? AND campaign_id=? AND status=?", uid, campaignId, "draft").
		UpdateColumn("status", "launched").Error
}

func sanitizeLabel(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "client"
	}
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else if r == ' ' || r == '_' {
			b.WriteByte('-')
		}
	}
	out := b.String()
	if out == "" {
		return "client"
	}
	if len(out) > 32 {
		out = out[:32]
	}
	return out
}
