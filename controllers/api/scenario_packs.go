package api

import (
	"net/http"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// ScenarioPacks handles GET /api/scenario_packs/
func (as *Server) ScenarioPacks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusBadRequest)
		return
	}
	packs, err := models.ListScenarioPacks()
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: "Error listing scenario packs"}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, packs, http.StatusOK)
}

// ImportScenarioPack handles POST /api/scenario_packs/{id}/import
func (as *Server) ImportScenarioPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusBadRequest)
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]
	uid := ctx.Get(r, "user_id").(int64)
	result, err := models.ImportScenarioPack(uid, id)
	if err == models.ErrScenarioPackNotFound {
		JSONResponse(w, models.Response{Success: false, Message: "Scenario pack not found"}, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	JSONResponse(w, result, http.StatusCreated)
}
