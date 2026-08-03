// Package report renders a security assessment session into multiple
// deliverable formats: HTML, PDF, DOCX, and XLSX.
package report

import (
	"context"
	"time"
)

// Message is a chat message within a session, used as report input.
type Message struct {
	Role      string    `json:"role"` // user | assistant | system
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Session is the metadata header for a report.
type Session struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	AgentName string    `json:"agent_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Username  string    `json:"username"`
}

// Data bundles everything a report renderer needs.
type Data struct {
	Session  Session   `json:"session"`
	Messages []Message `json:"messages"`
	// GeneratedBy identifies the tool that produced the report.
	GeneratedBy string `json:"generated_by"`
	GeneratedAt time.Time
}

// Format identifies an output format.
type Format string

const (
	FormatHTML Format = "html"
	FormatPDF  Format = "pdf"
	FormatDOCX Format = "docx"
	FormatXLSX Format = "xlsx"
)

// Valid reports whether f is a supported output format.
func (f Format) Valid() bool {
	switch f {
	case FormatHTML, FormatPDF, FormatDOCX, FormatXLSX:
		return true
	}
	return false
}

// Renderer renders report data to a specific format as bytes.
type Renderer interface {
	Render(ctx context.Context, data *Data) ([]byte, error)
}

// New returns a renderer for the given format.
func New(f Format) Renderer {
	switch f {
	case FormatHTML:
		return &HTMLRenderer{}
	case FormatPDF:
		return &PDFRenderer{}
	case FormatDOCX:
		return &DOCXRenderer{}
	case FormatXLSX:
		return &XLSXRenderer{}
	}
	return nil
}
