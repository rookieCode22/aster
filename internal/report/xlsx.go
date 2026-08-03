package report

import (
	"bytes"
	"context"
	"strconv"

	"github.com/xuri/excelize/v2"
)

// XLSXRenderer renders a report to a structured .xlsx workbook. One sheet holds
// the session metadata header; each subsequent row is a chat message with role
// and content. This format is convenient for downstream analysis or import into
// audit dashboards.
type XLSXRenderer struct{}

func (r *XLSXRenderer) Render(_ context.Context, data *Data) ([]byte, error) {
	if r == nil {
		return nil, nil
	}

	f := excelize.NewFile()
	defer f.Close()

	sheet := "报告"
	idx, err := f.NewSheet(sheet)
	if err != nil {
		return nil, err
	}
	f.SetActiveSheet(idx)
	// Remove the default sheet excelize creates.
	if err := f.DeleteSheet("Sheet1"); err != nil {
		return nil, err
	}

	// Metadata block
	f.SetCellValue(sheet, "A1", "会话标题")
	f.SetCellValue(sheet, "B1", data.Session.Title)
	f.SetCellValue(sheet, "A2", "会话ID")
	f.SetCellValue(sheet, "B2", data.Session.ID)
	if data.Session.Username != "" {
		f.SetCellValue(sheet, "A3", "分析人")
		f.SetCellValue(sheet, "B3", data.Session.Username)
	}
	if !data.Session.CreatedAt.IsZero() {
		f.SetCellValue(sheet, "A4", "创建时间")
		f.SetCellValue(sheet, "B4", data.Session.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	f.SetCellValue(sheet, "A5", "生成时间")
	f.SetCellValue(sheet, "B5", data.GeneratedAt.Format("2006-01-02 15:04:05"))
	f.SetCellValue(sheet, "A6", "生成工具")
	f.SetCellValue(sheet, "B6", data.GeneratedBy)

	// Column headers for message table
	hdrRow := 8
	f.SetCellValue(sheet, "A"+strconv.Itoa(hdrRow), "序号")
	f.SetCellValue(sheet, "B"+strconv.Itoa(hdrRow), "角色")
	f.SetCellValue(sheet, "C"+strconv.Itoa(hdrRow), "时间")
	f.SetCellValue(sheet, "D"+strconv.Itoa(hdrRow), "内容")

	row := hdrRow + 1
	for i, m := range data.Messages {
		f.SetCellValue(sheet, "A"+strconv.Itoa(row), i+1)
		f.SetCellValue(sheet, "B"+strconv.Itoa(row), m.Role)
		ts := ""
		if !m.CreatedAt.IsZero() {
			ts = m.CreatedAt.Format("2006-01-02 15:04:05")
		}
		f.SetCellValue(sheet, "C"+strconv.Itoa(row), ts)
		f.SetCellValue(sheet, "D"+strconv.Itoa(row), m.Content)
		row++
	}

	// Widen columns for readability.
	widths := map[string]float64{"A": 8, "B": 12, "C": 22, "D": 80}
	for col, w := range widths {
		f.SetColWidth(sheet, col, col, w)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
