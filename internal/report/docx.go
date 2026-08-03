package report

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"strings"
)

// DOCXRenderer renders a report to a minimal OOXML .docx file using only the
// standard library (archive/zip + encoding/xml). Generated documents open in
// MS Word, LibreOffice, and WPS. Chinese text renders correctly as DOCX uses
// Unicode internally.
type DOCXRenderer struct{}

func (r *DOCXRenderer) Render(_ context.Context, data *Data) ([]byte, error) {
	if r == nil {
		return nil, nil
	}

	documentXML := buildDocumentXML(data)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	files := map[string][]byte{
		"[Content_Types].xml": []byte(contentTypesXML),
		"_rels/.rels":         []byte(relsXML),
		"word/document.xml":   []byte(documentXML),
	}
	// Add empty numbering/styles to keep Word happy when opening.
	files["word/styles.xml"] = []byte(stylesXML)

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(content); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildDocumentXML(data *Data) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<w:document xmlns:wpc="http://schemas.microsoft.com/office/word/2010/wordprocessingCanvas" xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006" xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math" xmlns:v="urn:schemas-microsoft-com:vml" xmlns:wp14="http://schemas.microsoft.com/office/word/2010/wordprocessingDrawing" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:w10="urn:schemas-microsoft-com:office:word" xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:w14="http://schemas.microsoft.com/office/word/2010/wordml" xmlns:wpg="http://schemas.microsoft.com/office/word/2010/wordprocessingGroup" xmlns:wpi="http://schemas.microsoft.com/office/word/2010/wordprocessingInk" xmlns:wne="http://schemas.microsoft.com/office/word/2006/wordml" xmlns:wps="http://schemas.microsoft.com/office/word/2010/wordprocessingShape"><w:body>`)

	// Title
	b.WriteString(`<w:p><w:pPr><w:jc w:val="center"/><w:spacing w:after="240"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="36"/></w:rPr><w:t>` + xmlEscape(data.Session.Title) + ` — 安全评估报告</w:t></w:r></w:p>`)

	// Metadata
	addParagraph(&b, "会话ID: "+data.Session.ID, false)
	if data.Session.Username != "" {
		addParagraph(&b, "分析人: "+data.Session.Username, false)
	}
	if !data.Session.CreatedAt.IsZero() {
		addParagraph(&b, "创建时间: "+data.Session.CreatedAt.Format("2006-01-02 15:04"), false)
	}
	addParagraph(&b, "生成时间: "+data.GeneratedAt.Format("2006-01-02 15:04"), false)
	addParagraph(&b, "", false)

	// Body
	b.WriteString(`<w:p><w:pPr><w:spacing w:before="240"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="28"/></w:rPr><w:t>对话记录</w:t></w:r></w:p>`)
	if len(data.Messages) == 0 {
		addParagraph(&b, "该会话暂无对话内容。", false)
	}
	for _, m := range data.Messages {
		label := "Agent"
		if m.Role == "user" {
			label = "用户"
		}
		b.WriteString(`<w:p><w:pPr><w:spacing w:before="160"/></w:pPr><w:r><w:rPr><w:b/><w:color w:val="2E74B5"/></w:rPr><w:t>` + xmlEscape(label) + `</w:t></w:r></w:p>`)
		addParagraph(&b, m.Content, false)
	}

	b.WriteString(`</w:body></w:document>`)
	return b.String()
}

func addParagraph(b *strings.Builder, text string, bold bool) {
	b.WriteString(`<w:p><w:r>`)
	if bold {
		b.WriteString(`<w:rPr><w:b/></w:rPr>`)
	}
	b.WriteString(`<w:t xml:space="preserve">` + xmlEscape(text) + `</w:t></w:r></w:p>`)
}

func xmlEscape(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>`

const relsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="word/styles.xml"/>
</Relationships>`

const stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Calibri" w:eastAsia="Microsoft YaHei" w:hAnsi="Calibri"/><w:sz w:val="22"/></w:rPr></w:rPrDefault></w:docDefaults>
<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>
</w:styles>`
