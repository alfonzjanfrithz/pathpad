package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"dontpad/internal/models"
	"dontpad/internal/storage"
)

// Handler holds dependencies for API handlers.
type Handler struct {
	Store          *storage.SQLiteStore
	Cache          *storage.Cache
	MaxContentSize int64
}

// extractPadPath extracts and normalizes the pad path from the URL.
// For routes like /api/pad/content/*, it strips the prefix.
func extractPadPath(r *http.Request, prefix string) string {
	path := strings.TrimPrefix(r.URL.Path, prefix)
	return models.NormalizePath(path)
}

// jsonResponse writes a JSON response with the given status code.
func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// jsonError writes a JSON error response.
func jsonError(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, map[string]string{"error": message})
}

// GetPad handles GET /api/pad/content/*
func (h *Handler) GetPad(w http.ResponseWriter, r *http.Request) {
	path := extractPadPath(r, "/api/pad/content/")
	// Also handle root path (no trailing path after prefix).
	if r.URL.Path == "/api/pad/content" || r.URL.Path == "/api/pad/content/" {
		path = ""
	}

	if err := models.ValidatePath(path); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Try cache first.
	if cached := h.Cache.Get(path); cached != nil {
		jsonResponse(w, http.StatusOK, cached)
		return
	}

	pad, err := h.Store.GetPad(path)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to get pad")
		return
	}

	// Cache the result.
	h.Cache.Set(path, pad)

	jsonResponse(w, http.StatusOK, pad)
}

// SavePad handles PUT /api/pad/content/*
func (h *Handler) SavePad(w http.ResponseWriter, r *http.Request) {
	path := extractPadPath(r, "/api/pad/content/")
	if r.URL.Path == "/api/pad/content" || r.URL.Path == "/api/pad/content/" {
		path = ""
	}

	if err := models.ValidatePath(path); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Read and parse request body.
	body, err := io.ReadAll(io.LimitReader(r.Body, h.MaxContentSize+1))
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to read request body")
		return
	}
	if int64(len(body)) > h.MaxContentSize {
		jsonError(w, http.StatusRequestEntityTooLarge, "content exceeds maximum size")
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	pad, err := h.Store.SavePad(path, req.Content)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to save pad")
		return
	}

	// Invalidate cache and set fresh entry.
	h.Cache.Invalidate(path)
	h.Cache.Set(path, pad)

	jsonResponse(w, http.StatusOK, pad)
}

// DeletePad handles DELETE /api/pad/content/*
func (h *Handler) DeletePad(w http.ResponseWriter, r *http.Request) {
	path := extractPadPath(r, "/api/pad/content/")
	if r.URL.Path == "/api/pad/content" || r.URL.Path == "/api/pad/content/" {
		path = ""
	}

	if err := models.ValidatePath(path); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	count, err := h.Store.DeletePad(path)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to delete pad")
		return
	}

	// Invalidate cache for the pad and descendants.
	if path == "" {
		h.Cache.InvalidatePrefix("")
	} else {
		h.Cache.InvalidatePrefix(path)
	}

	jsonResponse(w, http.StatusOK, map[string]int64{"deleted": count})
}

// GetChildren handles GET /api/pad/children/*
func (h *Handler) GetChildren(w http.ResponseWriter, r *http.Request) {
	path := extractPadPath(r, "/api/pad/children/")
	if r.URL.Path == "/api/pad/children" || r.URL.Path == "/api/pad/children/" {
		path = ""
	}

	if err := models.ValidatePath(path); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	children, err := h.Store.GetChildren(path)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to get children")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"children": children})
}

// Health handles GET /healthz
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	dbStatus := "ok"
	httpStatus := http.StatusOK

	if err := h.Store.Ping(); err != nil {
		dbStatus = "error"
		httpStatus = http.StatusServiceUnavailable
	}

	jsonResponse(w, httpStatus, map[string]string{
		"status": func() string {
			if httpStatus == http.StatusOK {
				return "ok"
			}
			return "error"
		}(),
		"db": dbStatus,
	})
}
