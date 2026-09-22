package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
)

// SendingChannels handles requests for /api/sending_channels/
func (as *Server) SendingChannels(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		cs, err := models.GetSendingChannels(ctx.Get(r, "user_id").(int64))
		if err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		for i := range cs {
			cs[i].MaskSecret()
		}
		JSONResponse(w, cs, http.StatusOK)
	case r.Method == "POST":
		s := models.SendingChannel{}
		err := json.NewDecoder(r.Body).Decode(&s)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
			return
		}
		_, err = models.GetSendingChannelByName(s.Name, ctx.Get(r, "user_id").(int64))
		if err != gorm.ErrRecordNotFound && err != models.ErrSendingChannelNotFound {
			JSONResponse(w, models.Response{Success: false, Message: "Sending channel name already in use"}, http.StatusConflict)
			return
		}
		s.ModifiedDate = time.Now().UTC()
		s.UserId = ctx.Get(r, "user_id").(int64)
		err = models.PostSendingChannel(&s)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		s.MaskSecret()
		JSONResponse(w, s, http.StatusCreated)
	}
}

// SendingChannel handles requests for /api/sending_channels/:id
func (as *Server) SendingChannel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	uid := ctx.Get(r, "user_id").(int64)
	s, err := models.GetSendingChannel(id, uid)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Sending channel not found"}, http.StatusNotFound)
		return
	}
	switch {
	case r.Method == "GET":
		s.MaskSecret()
		JSONResponse(w, s, http.StatusOK)
	case r.Method == "DELETE":
		err = models.DeleteSendingChannel(id, uid)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Sending channel deleted successfully!"}, http.StatusOK)
	case r.Method == "PUT":
		s = models.SendingChannel{}
		err = json.NewDecoder(r.Body).Decode(&s)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
			return
		}
		if s.Id != id {
			JSONResponse(w, models.Response{Success: false, Message: "Error: /:id and channel id mismatch"}, http.StatusBadRequest)
			return
		}
		s.UserId = uid
		err = models.PutSendingChannel(&s)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		s.MaskSecret()
		JSONResponse(w, s, http.StatusOK)
	}
}

type sendingChannelTestPayload struct {
	models.SendingChannel
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Message   string `json:"message"`
	Subject   string `json:"subject"`
	URL       string `json:"url"`
}

// TestSendingChannel handles POST /api/sending_channels/test and /api/sending_channels/:id/test
func (as *Server) TestSendingChannel(w http.ResponseWriter, r *http.Request) {
	uid := ctx.Get(r, "user_id").(int64)
	payload := sendingChannelTestPayload{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
		return
	}
	s := payload.SendingChannel
	vars := mux.Vars(r)
	if idStr, ok := vars["id"]; ok && idStr != "" {
		id, _ := strconv.ParseInt(idStr, 0, 64)
		existing, err := models.GetSendingChannel(id, uid)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Sending channel not found"}, http.StatusNotFound)
			return
		}
		if s.Name == "" {
			s = existing
		} else {
			s.Id = existing.Id
			s.UserId = uid
			if s.AuthHeaderValue == "" {
				s.AuthHeaderValue = existing.AuthHeaderValue
			}
		}
	}
	s.UserId = uid
	req := models.SendingChannelTestRequest{
		Phone:     payload.Phone,
		Email:     payload.Email,
		FirstName: payload.FirstName,
		LastName:  payload.LastName,
		Message:   payload.Message,
		Subject:   payload.Subject,
		URL:       payload.URL,
	}
	if err := s.TestDeliver(req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
		return
	}
	JSONResponse(w, models.Response{Success: true, Message: "Test message sent successfully!"}, http.StatusOK)
}
