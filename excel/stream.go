package excel

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

type StreamSheet struct {
	file      *excelize.File
	stream    *excelize.StreamWriter
	sheetName string
	nextRow   int
	flushed   bool
}

func NewStreamSheet(sheetName string, headers []any) (*StreamSheet, error) {
	s, err := newStreamSheet(sheetName)
	if err != nil {
		return nil, err
	}
	if len(headers) > 0 {
		if err := s.stream.SetRow("A1", headers); err != nil {
			s.Close()
			return nil, err
		}
		s.nextRow = 2
	}
	return s, nil
}

func newStreamSheet(sheetName string) (*StreamSheet, error) {
	file := excelize.NewFile()
	if err := file.SetSheetName("Sheet1", sheetName); err != nil {
		file.Close()
		return nil, err
	}
	stream, err := file.NewStreamWriter(sheetName)
	if err != nil {
		file.Close()
		return nil, err
	}
	return &StreamSheet{
		file:      file,
		stream:    stream,
		sheetName: sheetName,
		nextRow:   1,
	}, nil
}

func NewEmptyStreamSheet(sheetName string) (*StreamSheet, error) {
	return newStreamSheet(sheetName)
}

func (s *StreamSheet) SheetName() string { return s.sheetName }

func (s *StreamSheet) UnderlyingFile() *excelize.File { return s.file }

func (s *StreamSheet) UnderlyingStreamWriter() *excelize.StreamWriter { return s.stream }

func (s *StreamSheet) SetRowAt(cell string, row []any) error {
	return s.stream.SetRow(cell, row)
}

func (s *StreamSheet) MergeCell(topLeft, bottomRight string) error {
	return s.stream.MergeCell(topLeft, bottomRight)
}

func (s *StreamSheet) SetColWidth(colIndex int, width float64) error {
	return s.stream.SetColWidth(colIndex, colIndex, width)
}

func (s *StreamSheet) NewStyle(style *excelize.Style) (int, error) {
	return s.file.NewStyle(style)
}

func (s *StreamSheet) SetNextDataRow(row int) {
	s.nextRow = row
}

func (s *StreamSheet) AppendInterfaceRow(row []interface{}) error {
	cells := make([]any, len(row))
	copy(cells, row)
	return s.AppendRow(cells)
}

func (s *StreamSheet) AppendRow(row []any) error {
	cell, err := excelize.CoordinatesToCellName(1, s.nextRow)
	if err != nil {
		return fmt.Errorf("excel.AppendRow: %w", err)
	}
	if err := s.stream.SetRow(cell, row); err != nil {
		return err
	}
	s.nextRow++
	return nil
}

func (s *StreamSheet) Flush() error {
	if s.flushed {
		return nil
	}
	if err := s.stream.Flush(); err != nil {
		return err
	}
	s.flushed = true
	return nil
}

func (s *StreamSheet) Bytes() ([]byte, error) {
	if err := s.Flush(); err != nil {
		return nil, err
	}
	buf, err := s.file.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *StreamSheet) Close() error {
	return s.file.Close()
}
