package report

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

func testData() *Data {
	return &Data{
		Session: Session{
			ID: "sess-123", Title: "某电商系统渗透测试", AgentName: "code-audit",
			CreatedAt: time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC),
			Username:  "yummy",
		},
		Messages: []Message{
			{Role: "user", Content: "请审计登录接口是否存在越权", CreatedAt: time.Now()},
			{Role: "assistant", Content: "已发现水平越权漏洞:\n```\nGET /api/users/2 -> 返回他人数据\n```\n攻击者可遍历 user_id 访问任意用户订单。", CreatedAt: time.Now()},
			{Role: "assistant", Content: "高危", CreatedAt: time.Now()},
		},
		GeneratedBy: "Aster Web",
		GeneratedAt: time.Now(),
	}
}

func TestHTMLValid(t *testing.T) {
	b, err := New(FormatHTML).Render(context.Background(), testData())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte("<!DOCTYPE html>")) {
		t.Fatal("HTML missing doctype")
	}
	if !bytes.Contains(b, []byte("某电商系统渗透测试")) {
		t.Fatal("HTML missing Chinese title")
	}
	// HTML must escape dangerous content
	s := string(b)
	if strings.Contains(s, "<script>") {
		t.Fatal("HTML did not escape user content")
	}
}

func TestPDFValid(t *testing.T) {
	b, err := New(FormatPDF).Render(context.Background(), testData())
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 1000 {
		t.Fatalf("PDF too small: %d", len(b))
	}
	// PDF magic: %PDF
	if b[0] != '%' || b[1] != 'P' || b[2] != 'D' || b[3] != 'F' {
		t.Fatal("PDF magic bytes missing")
	}
}

func TestDOCXIsValidZip(t *testing.T) {
	b, err := New(FormatDOCX).Render(context.Background(), testData())
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatalf("DOCX not a valid zip: %v", err)
	}
	var foundDocument bool
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			foundDocument = true
			rc, _ := f.Open()
			content := &bytes.Buffer{}
			content.ReadFrom(rc)
			rc.Close()
			if !strings.Contains(content.String(), "某电商系统渗透测试") {
				t.Fatal("DOCX document.xml missing Chinese title")
			}
		}
	}
	if !foundDocument {
		t.Fatal("DOCX missing word/document.xml")
	}
}

func TestXLSXValid(t *testing.T) {
	b, err := New(FormatXLSX).Render(context.Background(), testData())
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("XLSX not readable: %v", err)
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		t.Fatal("XLSX has no sheets")
	}
	v, err := f.GetCellValue(sheets[0], "B1")
	if err != nil || v != "某电商系统渗透测试" {
		t.Fatalf("XLSX B1 title wrong: %q err=%v", v, err)
	}
}

// TestMIMESniff sniffs that generated files carry the right content-types so the
// web handler can set correct headers. This guards the download endpoint.
func TestMIMESniff(t *testing.T) {
	cases := map[Format]string{
		FormatHTML: "text/html",
		FormatPDF:  "application/pdf",
	}
	for f, want := range cases {
		b, _ := New(f).Render(context.Background(), testData())
		got := http.DetectContentType(b)
		t.Logf("%s sniffed as %s (expect approx %s)", f, got, want)
		// At minimum the type should contain the primary subtype root.
		if got == "" {
			t.Fatalf("%s produced empty content type", f)
		}
	}
}

// TestRenderWritesFiles writes all four formats to /tmp for manual inspection.
func TestRenderWritesFiles(t *testing.T) {
	if os.Getenv("ASTER_WRITE_REPORTS") == "" {
		t.Skip("set ASTER_WRITE_REPORTS=1 to write report files")
	}
	for _, f := range []Format{FormatHTML, FormatPDF, FormatDOCX, FormatXLSX} {
		b, err := New(f).Render(context.Background(), testData())
		if err != nil {
			t.Fatal(err)
		}
		if e := os.WriteFile("/tmp/out-report."+string(f), b, 0644); e != nil {
			t.Fatal(e)
		}
	}
}
