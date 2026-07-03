package excel

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

type MultiStreamWorkbook struct {
	file       *excelize.File
	streams    map[string]*excelize.StreamWriter
	sheetNames []string
	rowCursors map[string]int
	flushed    map[string]bool
}

func NewMultiStreamWorkbook(sheetNames []string, headers []any) (*MultiStreamWorkbook, error) {
	file := excelize.NewFile()
	streams := make(map[string]*excelize.StreamWriter, len(sheetNames))
	rowCursors := make(map[string]int, len(sheetNames))
	flushed := make(map[string]bool, len(sheetNames))

	for i, sheet := range sheetNames {
		if i == 0 {
			if err := file.SetSheetName("Sheet1", sheet); err != nil {
				file.Close()
				return nil, err
			}
		} else if _, err := file.NewSheet(sheet); err != nil {
			file.Close()
			return nil, err
		}

		stream, err := file.NewStreamWriter(sheet)
		if err != nil {
			file.Close()
			return nil, err
		}
		if len(headers) > 0 {
			if err := stream.SetRow("A1", headers); err != nil {
				file.Close()
				return nil, err
			}
		}
		streams[sheet] = stream
		rowCursors[sheet] = 2
	}

	return &MultiStreamWorkbook{
		file:       file,
		streams:    streams,
		sheetNames: sheetNames,
		rowCursors: rowCursors,
		flushed:    flushed,
	}, nil
}

func (m *MultiStreamWorkbook) Stream(sheet string) *excelize.StreamWriter {
	return m.streams[sheet]
}

func (m *MultiStreamWorkbook) AppendInterfaceRows(sheet string, rows [][]interface{}) error {
	anyRows := make([][]any, len(rows))
	for i, row := range rows {
		anyRows[i] = make([]any, len(row))
		copy(anyRows[i], row)
	}
	return m.AppendRows(sheet, anyRows)
}

func (m *MultiStreamWorkbook) AppendRows(sheet string, rows [][]any) error {
	stream, ok := m.streams[sheet]
	if !ok {
		return fmt.Errorf("excel.MultiStreamWorkbook: unknown sheet %q", sheet)
	}
	rowNum := m.rowCursors[sheet]
	for _, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, rowNum)
		if err != nil {
			return err
		}
		if err := stream.SetRow(cell, row); err != nil {
			return err
		}
		rowNum++
	}
	m.rowCursors[sheet] = rowNum
	return nil
}

func (m *MultiStreamWorkbook) FlushSheet(sheet string) error {
	if m.flushed[sheet] {
		return nil
	}
	stream, ok := m.streams[sheet]
	if !ok {
		return fmt.Errorf("excel.MultiStreamWorkbook: unknown sheet %q", sheet)
	}
	if err := stream.Flush(); err != nil {
		return err
	}
	m.flushed[sheet] = true
	return nil
}

func (m *MultiStreamWorkbook) FlushAll() error {
	for sheet := range m.streams {
		if err := m.FlushSheet(sheet); err != nil {
			return err
		}
	}
	return nil
}

func (m *MultiStreamWorkbook) Bytes() ([]byte, error) {
	if err := m.FlushAll(); err != nil {
		return nil, err
	}
	buf, err := m.file.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *MultiStreamWorkbook) Close() error {
	return m.file.Close()
}
