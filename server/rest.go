package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// validationError marks errors that should result in HTTP 400.
type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }

func newValidationError(msg string) error {
	return &validationError{msg: msg}
}

const maxRequestBodySize = 1 << 20 // 1MB

// RESTHandler provides plain JSON HTTP endpoints for web search and fetch.
type RESTHandler struct {
	srv *WebServer
}

// NewRESTHandler creates a new REST handler wrapping the given WebServer.
func NewRESTHandler(srv *WebServer) *RESTHandler {
	return &RESTHandler{srv: srv}
}

// HandleHealth handles GET /api/health
func (h *RESTHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSONResponse(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": "1.0.0",
	})
}

// HandleEngines handles GET /api/engines
func (h *RESTHandler) HandleEngines(w http.ResponseWriter, r *http.Request) {
	engines := h.srv.AvailableEngines()
	writeJSONResponse(w, http.StatusOK, map[string]any{
		"engines": engines,
		"default": h.srv.defaultEngine,
	})
}

// HandleSearch handles POST /api/search
func (h *RESTHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if !h.requireJSON(w, r) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var params WebSearchParams
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&params); err != nil {
		if err.Error() == "http: request body too large" {
			writeJSONError(w, http.StatusRequestEntityTooLarge, "Payload Too Large", "request body exceeds 1MB limit")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}

	resp, err := h.srv.HandleWebSearch(r.Context(), params)
	if err != nil {
		var ve *validationError
		if errors.As(err, &ve) {
			writeJSONError(w, http.StatusBadRequest, "Bad Request", err.Error())
		} else {
			writeJSONError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		}
		return
	}

	writeJSONResponse(w, http.StatusOK, resp)
}

// HandleFetch handles POST /api/fetch
func (h *RESTHandler) HandleFetch(w http.ResponseWriter, r *http.Request) {
	if !h.requireJSON(w, r) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var params WebFetchParams
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&params); err != nil {
		if err.Error() == "http: request body too large" {
			writeJSONError(w, http.StatusRequestEntityTooLarge, "Payload Too Large", "request body exceeds 1MB limit")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}

	resp, err := h.srv.HandleWebFetch(r.Context(), params)
	if err != nil {
		var ve *validationError
		if errors.As(err, &ve) {
			writeJSONError(w, http.StatusBadRequest, "Bad Request", err.Error())
		} else {
			writeJSONError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		}
		return
	}

	writeJSONResponse(w, http.StatusOK, resp)
}

func (h *RESTHandler) requireJSON(w http.ResponseWriter, r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	if ct != "" && !strings.HasPrefix(ct, "application/json") {
		writeJSONError(w, http.StatusUnsupportedMediaType, "Unsupported Media Type", "Content-Type must be application/json")
		return false
	}
	return true
}

