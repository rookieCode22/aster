package report

import (
	"bytes"
	"context"
	"html"
	"strings"
	"time"
)

// HTMLRenderer renders a report to a standalone HTML document.
type HTMLRenderer struct{}

func (r *HTMLRenderer) Render(_ context.Context, data *Data) ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	var b bytes.Buffer
	b.WriteString("<!DOCTYPE html>\n<html lang=\"zh\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>" + html.EscapeString(data.Session.Title) + " — 安全评估报告</title>\n")
	b.WriteString("<style>\n")
	b.WriteString("body{font-family:'-apple-system','Segoe UI','Microsoft YaHei',sans-serif;max-width:900px;margin:32px auto;padding:0 24px;color:#1a1a1a;line-height:1.7;}\n")
	b.WriteString("h1{font-size:24px;border-bottom:2px solid #4a90d9;padding-bottom:8px;}\n")
	b.WriteString("h2{font-size:18px;color:#2c3e50;margin-top:28px;}\n")
	b.WriteString(".meta{color:#666;font-size:13px;margin:12px 0 24px;}\n")
	b.WriteString(".msg{border:1px solid #e3e3e3;border-radius:6px;padding:12px 16px;margin:12px 0;}\n")
	b.WriteString(".msg .role{font-weight:600;font-size:12px;text-transform:uppercase;color:#4a90d9;margin-bottom:6px;}\n")
	b.WriteString(".msg.user .role{color:#27ae60;}\n")
	b.WriteString(".msg .body{white-space:pre-wrap;word-break:break-word;}\n")
	b.WriteString("code{background:#f4f4f4;padding:2px 4px;border-radius:3px;font-size:90%;}\n")
	b.WriteString("pre{background:#f6f8fa;padding:12px;border-radius:6px;overflow-x:auto;font-size:13px;}\n")
	b.WriteString("table{border-collapse:collapse;width:100%;margin:12px 0;}\n")
	b.WriteString("th,td{border:1px solid #ddd;padding:8px;text-align:left;font-size:14px;}\n")
	b.WriteString("th{background:#f2f2f2;}\n")
	b.WriteString("</style>\n</head>\n<body>\n")

	b.WriteString("<h1>" + html.EscapeString(data.Session.Title) + "</h1>\n")
	b.WriteString("<div class=\"meta\">\n")
	b.WriteString("  会话ID: " + html.EscapeString(data.Session.ID) + "<br>\n")
	if data.Session.Username != "" {
		b.WriteString("  分析人: " + html.EscapeString(data.Session.Username) + "<br>\n")
	}
	if !data.Session.CreatedAt.IsZero() {
		b.WriteString("  创建时间: " + data.Session.CreatedAt.Format(time.RFC3339) + "<br>\n")
	}
	b.WriteString("  生成时间: " + data.GeneratedAt.Format(time.RFC3339) + "<br>\n")
	b.WriteString("  生成工具: " + html.EscapeString(data.GeneratedBy) + "\n")
	b.WriteString("</div>\n")

	b.WriteString("<h2>对话记录</h2>\n")
	if len(data.Messages) == 0 {
		b.WriteString("<p>该会话暂无对话内容。</p>\n")
	}
	for _, m := range data.Messages {
		cls := "msg"
		if m.Role == "user" {
			cls += " user"
		}
		b.WriteString("<div class=\"" + cls + "\">\n")
		b.WriteString("<div class=\"role\">" + html.EscapeString(roleLabel(m.Role)) + "</div>\n")
		b.WriteString("<div class=\"body\">" + renderMarkdownish(m.Content) + "</div>\n")
		b.WriteString("</div>\n")
	}

	b.WriteString("</body>\n</html>\n")
	return b.Bytes(), nil
}

func roleLabel(role string) string {
	switch role {
	case "user":
		return "用户"
	case "assistant":
		return "Agent"
	case "system":
		return "系统"
	}
	return role
}

// renderMarkdownish does lightweight formatting: escapes HTML, wraps code
// blocks in <pre>, and renders inline backticks/bold as <code>/<b>. It is a
// pragmatic approximation, not a full Markdown parser — good enough for
// security report readability and safe against HTML injection.
func renderMarkdownish(content string) string {
	escaped := html.EscapeString(content)

	// Fenced code blocks: ```lang\n...\n```
	builder := &strings.Builder{}
	rest := escaped
	for {
		idx := strings.Index(rest, "```")
		if idx < 0 {
			builder.WriteString(renderInline(rest))
			break
		}
		builder.WriteString(renderInline(rest[:idx]))
		rest = rest[idx+3:]
		closeIdx := strings.Index(rest, "```")
		if closeIdx < 0 {
			builder.WriteString("<pre>" + rest + "</pre>")
			break
		}
		code := rest[:closeIdx]
		// strip optional language tag on first line
		if nl := strings.IndexByte(code, '\n'); nl >= 0 && len(code[:nl]) <= 20 && !strings.ContainsAny(code[:nl], " ") {
			code = code[nl+1:]
		}
		builder.WriteString("<pre><code>" + code + "</code></pre>")
		rest = rest[closeIdx+3:]
	}
	return builder.String()
}

func renderInline(s string) string {
	// inline code `x` -> <code>x</code>
	var b strings.Builder
	for {
		i := strings.Index(s, "`")
		if i < 0 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:i])
		s = s[i+1:]
		j := strings.Index(s, "`")
		if j < 0 {
			b.WriteString("<code>" + s + "</code>")
			break
		}
		b.WriteString("<code>" + s[:j] + "</code>")
		s = s[j+1:]
	}
	return b.String()
}
