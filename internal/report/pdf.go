package report

import (
	"bytes"
	"context"
	"errors"

	"github.com/go-pdf/fpdf"
)

// wqyZenheiTTF is the embedded WenQuanYi Zen Hei TrueType font used for CJK
// rendering in PDF reports. It is injected at build time (see font_embed.go).
//
// The font lives in the report package's fonts/ directory so that any build
// machine can produce Chinese PDFs without a system-level CJK font.
var wqyZenheiTTF []byte

// PDFRenderer renders a report to a multi-page PDF using go-pdf/fpdf.
type PDFRenderer struct{}

func (r *PDFRenderer) Render(ctx context.Context, data *Data) ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	if len(wqyZenheiTTF) == 0 {
		return nil, errors.New("CJK font not embedded; rebuild with internal/report/fonts/wqy-zenhei-regular.ttf")
	}

	var buf bytes.Buffer
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(data.Session.Title, true)
	pdf.SetAuthor(data.GeneratedBy, true)

	marginX := 15.0
	marginY := 18.0
	pdf.SetMargins(marginX, marginY, marginX)
	pdf.SetAutoPageBreak(true, 20)

	pdf.AddUTF8FontFromBytes("wqy", "", wqyZenheiTTF)
	pdf.SetFont("wqy", "", 12)

	pdf.AddPage()

	// Title
	pdf.SetFontSize(20)
	pdf.SetY(marginY)
	pdf.Cell(0, 10, truncate(data.Session.Title, 40))
	pdf.Ln(12)

	// Metadata
	pdf.SetFontSize(9)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(0, 5, "会话ID: "+data.Session.ID)
	pdf.Ln(5)
	if data.Session.Username != "" {
		pdf.Cell(0, 5, "分析人: "+data.Session.Username)
		pdf.Ln(5)
	}
	if !data.Session.CreatedAt.IsZero() {
		pdf.Cell(0, 5, "创建时间: "+data.Session.CreatedAt.Format("2006-01-02 15:04"))
		pdf.Ln(5)
	}
	pdf.Cell(0, 5, "生成时间: "+data.GeneratedAt.Format("2006-01-02 15:04"))
	pdf.Ln(8)
	pdf.SetTextColor(0, 0, 0)

	// Conversation body
	pdf.SetFont("wqy", "", 11)
	if len(data.Messages) == 0 {
		pdf.Cell(0, 6, "该会话暂无对话内容。")
		pdf.Ln(6)
	}
	for _, m := range data.Messages {
		label := "[用户]"
		if m.Role != "user" {
			label = "[Agent]"
		}
		pdf.SetFont("wqy", "", 10)
		pdf.SetTextColor(0, 90, 180)
		pdf.Cell(0, 6, label)
		pdf.Ln(6)
		pdf.SetTextColor(40, 40, 40)
		pdf.Write(5, m.Content)
		pdf.Ln(5)
	}

	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}
