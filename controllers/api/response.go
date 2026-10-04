package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// JSONResponse attempts to set the status code, c, and marshal the given interface, d, into a response that
// is written to the given ResponseWriter.
func JSONResponse(w http.ResponseWriter, d interface{}, c int) {
	dj, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		http.Error(w, "Error creating JSON response", http.StatusInternalServerError)
		log.Error(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	fmt.Fprintf(w, "%s", dj)
}

// sendFailureResponse returns the raw error and a stable reason code.
func sendFailureResponse(w http.ResponseWriter, err error, status int) {
	JSONResponse(w, models.Response{
		Success: false,
		Message: err.Error(),
		Data:    map[string]string{"reason": models.ClassifySendError(err.Error())},
	}, status)
}
