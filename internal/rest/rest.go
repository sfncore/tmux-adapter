// Package rest provides HTTP REST endpoints for agent management.
package rest

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gastownhall/tmux-adapter/internal/agents"
	"github.com/gastownhall/tmux-adapter/internal/auth"
	"github.com/gastownhall/tmux-adapter/internal/nudge"
	"github.com/gastownhall/tmux-adapter/internal/tmux"
)

// Handler provides REST API endpoints for agent management.
type Handler struct {
	registry  *agents.Registry
	ctrl      *tmux.ControlMode
	authToken string
}

// New creates a new REST Handler.
func New(registry *agents.Registry, ctrl *tmux.ControlMode, authToken string) *Handler {
	return &Handler{
		registry:  registry,
		ctrl:      ctrl,
		authToken: authToken,
	}
}

// Register mounts all REST endpoints on the provided mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/agents", h.handleAgents)
	mux.HandleFunc("/api/agents/", h.handleAgentByName)
}

// handleAgents handles GET /api/agents — list all agents.
func (h *Handler) handleAgents(w http.ResponseWriter, r *http.Request) {
	if !auth.IsAuthorizedRequest(h.authToken, r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	all := h.registry.GetAgents()
	writeJSON(w, http.StatusOK, map[string]any{"agents": all})
}

// handleAgentByName routes /api/agents/{name} and /api/agents/{name}/... sub-paths.
func (h *Handler) handleAgentByName(w http.ResponseWriter, r *http.Request) {
	if !auth.IsAuthorizedRequest(h.authToken, r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
		return
	}

	// Strip "/api/agents/" prefix
	rest := strings.TrimPrefix(r.URL.Path, "/api/agents/")
	rest = strings.Trim(rest, "/")

	parts := strings.SplitN(rest, "/", 2)
	name := parts[0]
	var sub string
	if len(parts) == 2 {
		sub = parts[1]
	}

	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "agent name required"})
		return
	}

	switch {
	case sub == "" && r.Method == http.MethodGet:
		h.getAgent(w, r, name)
	case sub == "" && r.Method == http.MethodDelete:
		h.killAgent(w, r, name)
	case sub == "prompt" && r.Method == http.MethodPost:
		h.sendPrompt(w, r, name)
	case sub == "screen" && r.Method == http.MethodGet:
		h.captureScreen(w, r, name)
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
	}
}

// getAgent handles GET /api/agents/{name}.
func (h *Handler) getAgent(w http.ResponseWriter, _ *http.Request, name string) {
	agent, ok := h.registry.GetAgent(name)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "agent not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"agent": agent})
}

// sendPrompt handles POST /api/agents/{name}/prompt.
func (h *Handler) sendPrompt(w http.ResponseWriter, r *http.Request, name string) {
	agent, ok := h.registry.GetAgent(name)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "agent not found"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "failed to read body"})
		return
	}

	var payload struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Prompt == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "prompt field required"})
		return
	}

	mu := nudge.GetLock(name)
	mu.Lock()
	defer mu.Unlock()

	if err := nudge.Session(h.ctrl, agent, payload.Prompt); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// captureScreen handles GET /api/agents/{name}/screen.
func (h *Handler) captureScreen(w http.ResponseWriter, _ *http.Request, name string) {
	if _, ok := h.registry.GetAgent(name); !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "agent not found"})
		return
	}

	content, err := h.ctrl.CapturePaneVisible(name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"screen": content})
}

// killAgent handles DELETE /api/agents/{name}.
func (h *Handler) killAgent(w http.ResponseWriter, _ *http.Request, name string) {
	if _, ok := h.registry.GetAgent(name); !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "agent not found"})
		return
	}

	if err := h.ctrl.KillSession(name); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func writeJSON(w http.ResponseWriter, status int, payload map[string]any) {
	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		log.Printf("write json response: %v", err)
	}
}
