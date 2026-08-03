package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"aster/internal/report"
	"aster/internal/server/auth"
)

// handleExportReport builds a session report in the requested format and streams
// it to the client as a downloadable file. Only the session owner may export.
//
//	GET /api/v1/sessions/{id}/report?format=html|pdf|docx|xlsx
func (s *Server) handleExportReport(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if sessionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session id required"})
		return
	}

	user := auth.GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	// Ownership check.
	sess, err := s.sessions.Get(sessionID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	if sess.UserID != user.ID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	// Resolve format.
	format := report.Format(strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format"))))
	if format == "" {
		format = report.FormatHTML
	}
	if !format.Valid() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("unsupported format %q (use html, pdf, docx, or xlsx)", format)})
		return
	}

	// Load messages.
	rows, err := s.db.ListMessagesBySession(sessionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	data := &report.Data{
		Session: report.Session{
			ID:        sess.ID,
			Title:     sess.Title,
			AgentName: sess.AgentName,
			CreatedAt: sess.CreatedAt,
			UpdatedAt: sess.UpdatedAt,
			Username:  user.Username,
		},
		GeneratedBy: "Aster Web",
		GeneratedAt: time.Now(),
	}
	for _, m := range rows {
		data.Messages = append(data.Messages, report.Message{
			Role:      m.Role,
			Content:   m.Content,
			CreatedAt: m.CreatedAt,
		})
	}

	renderer := report.New(format)
	if renderer == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "no renderer for " + string(format)})
		return
	}
	payload, err := renderer.Render(context.Background(), data)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "render failed: " + err.Error()})
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"report-%s.%s\"", sessionID[:min(8, len(sessionID))], format))
	w.Header().Set("Content-Type", mimeFor(format))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(payload)))
	w.Write(payload)
}

func mimeFor(f report.Format) string {
	switch f {
	case report.FormatHTML:
		return "text/html; charset=utf-8"
	case report.FormatPDF:
		return "application/pdf"
	case report.FormatDOCX:
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case report.FormatXLSX:
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	return "application/octet-stream"
}
