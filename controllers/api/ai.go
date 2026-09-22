package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// AISchema returns a machine-readable capability catalog for external agents.
func (as *Server) AISchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusBadRequest)
		return
	}
	packs, err := models.ListScenarioPacks()
	if err != nil {
		log.Error(err)
		packs = []models.ScenarioPackSummary{}
	}
	schema := map[string]interface{}{
		"product": "shier",
		"version": "ai-orchestration-v1",
		"auth":    []string{"Authorization: Bearer <api_key>", "query api_key"},
		"policy": map[string]interface{}{
			"configure_allowed": true,
			"launch_requires":   []string{"ui_confirm", "confirm_token"},
			"forbidden":         []string{"/api/users/", "/api/reset", "/api/webhooks/", "/api/campaigns/{id}/purge"},
		},
		"endpoints": []map[string]string{
			{"method": "GET", "path": "/api/ai/schema", "purpose": "capability catalog"},
			{"method": "POST", "path": "/api/ai/plans/dry_run", "purpose": "validate intent without writes"},
			{"method": "POST", "path": "/api/ai/plans", "purpose": "create draft resource graph"},
			{"method": "GET", "path": "/api/ai/plans/{id}", "purpose": "fetch plan"},
			{"method": "POST", "path": "/api/ai/campaigns/{id}/prepare_launch", "purpose": "issue confirm_token"},
			{"method": "POST", "path": "/api/ai/campaigns/{id}/launch", "purpose": "launch draft with confirm_token"},
		},
		"plan_request_fields": map[string]interface{}{
			"required": []string{"url", "channel.smtp_id|channel.sending_channel_id", "targets|group_id", "scenario_pack_id|template_id+page_id"},
			"optional": []string{"client_label", "project_label", "name", "schedule.launch_date", "schedule.send_by_date"},
			"headers":  []string{"Idempotency-Key"},
		},
		"error_codes": []string{
			"launch_confirmation_required",
			"purge_confirmation_required",
			"ai plan validation failed",
			"campaign is not in draft status",
		},
		"campaign_statuses": []string{
			models.CampaignDraft,
			models.CampaignQueued,
			models.CampaignInProgress,
			models.CampaignComplete,
		},
		"scenario_packs": packs,
	}
	JSONResponse(w, schema, http.StatusOK)
}

// AIPlansDryRun validates an AI plan intent without persisting.
func (as *Server) AIPlansDryRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusBadRequest)
		return
	}
	uid := ctx.Get(r, "user_id").(int64)
	req := models.AIPlanRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid JSON structure"}, http.StatusBadRequest)
		return
	}
	errs := models.ValidateAIPlanRequest(uid, &req)
	if len(errs) > 0 {
		JSONResponse(w, models.Response{
			Success: false,
			Message: "validation_failed",
			Data:    errs,
		}, http.StatusBadRequest)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "ok", Data: map[string]string{"status": "valid"}}, http.StatusOK)
}

// AIPlans creates a draft plan or returns an existing idempotent result.
func (as *Server) AIPlans(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		uid := ctx.Get(r, "user_id").(int64)
		req := models.AIPlanRequest{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid JSON structure"}, http.StatusBadRequest)
			return
		}
		idem := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		plan, err := models.CreateAIPlan(uid, &req, idem)
		if err != nil {
			if errors.Is(err, models.ErrAIPlanValidation) {
				JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
				return
			}
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		JSONResponse(w, plan, http.StatusCreated)
	default:
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusBadRequest)
	}
}

// AIPlan returns a single plan by id.
func (as *Server) AIPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusBadRequest)
		return
	}
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)
	plan, err := models.GetAIPlan(id, uid)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
		return
	}
	resp := map[string]interface{}{
		"id":                 plan.Id,
		"status":             plan.Status,
		"client_label":       plan.ClientLabel,
		"project_label":      plan.ProjectLabel,
		"group_id":           plan.GroupId,
		"template_id":        plan.TemplateId,
		"page_id":            plan.PageId,
		"campaign_id":        plan.CampaignId,
		"smtp_id":            plan.SMTPId,
		"sending_channel_id": plan.SendingChannelId,
		"scenario_pack_id":   plan.ScenarioPackId,
		"warnings":           strings.Split(plan.Warnings, "\n"),
		"created_at":         plan.CreatedAt,
	}
	if plan.Warnings == "" {
		resp["warnings"] = []string{}
	}
	JSONResponse(w, resp, http.StatusOK)
}

type launchConfirmBody struct {
	ConfirmToken string `json:"confirm_token"`
}

// AIPrepareLaunch issues a one-time confirm_token for a draft campaign.
func (as *Server) AIPrepareLaunch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusBadRequest)
		return
	}
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)
	c, err := models.GetCampaign(id, uid)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "campaign not found"}, http.StatusNotFound)
		return
	}
	if c.Status != models.CampaignDraft {
		JSONResponse(w, models.Response{Success: false, Message: models.ErrCampaignNotDraft.Error()}, http.StatusConflict)
		return
	}
	token, exp, err := models.IssueLaunchConfirmToken(id, uid)
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, map[string]interface{}{
		"campaign_id":   id,
		"confirm_token": token,
		"expires_at":    exp,
	}, http.StatusOK)
}

// AILaunch launches a draft campaign when a valid confirm_token is provided.
func (as *Server) AILaunch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusBadRequest)
		return
	}
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)
	body := launchConfirmBody{}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := models.ConsumeLaunchConfirmToken(body.ConfirmToken, id, uid); err != nil {
		JSONResponse(w, models.Response{
			Success: false,
			Message: "launch_confirmation_required",
		}, http.StatusForbidden)
		return
	}
	c, err := models.LaunchDraftCampaign(id, uid)
	if err != nil {
		if errors.Is(err, models.ErrCampaignNotDraft) {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusConflict)
			return
		}
		if errors.Is(err, models.ErrCampaignNotFound) {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusNotFound)
			return
		}
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	if c.Status == models.CampaignInProgress {
		go as.worker.LaunchCampaign(*c)
	}
	// Mark related AI plans launched when present.
	_ = models.MarkAIPlansLaunched(uid, id)
	JSONResponse(w, c, http.StatusOK)
}
