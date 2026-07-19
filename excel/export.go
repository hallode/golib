package excel

import (
	"fmt"

	"github.com/hallode/golib/v2/json"

	"github.com/xuri/excelize/v2"
)

type ExportConfig struct {
	SheetName string
	Filename  string
	Headers   []string
	Rows      [][]string
}

type DynamicExportConfig struct {
	SheetName string
	Filename  string
	Headers   []string
	FieldKeys []string
	DataRows  []ExportDataRow
}

type ExportDataRow struct {
	RawData      string
	ErrorMessage string
}

func GenerateExcel(config *ExportConfig) ([]byte, string, error) {
	f := excelize.NewFile()
	defer f.Close()

	f.SetSheetName("Sheet1", config.SheetName)

	rowID, err := f.NewStreamWriter(config.SheetName)
	if err != nil {
		return nil, "", err
	}

	headers := make([]any, len(config.Headers))
	for i, h := range config.Headers {
		headers[i] = h
	}
	if err := rowID.SetRow("A1", headers); err != nil {
		return nil, "", err
	}

	for i, row := range config.Rows {
		rowData := make([]any, len(row))
		for j, v := range row {
			rowData[j] = v
		}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		if err := rowID.SetRow(cell, rowData); err != nil {
			return nil, "", err
		}
	}

	if err := rowID.Flush(); err != nil {
		return nil, "", err
	}

	buffer, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}

	return buffer.Bytes(), config.Filename, nil
}

func getValueFromJSON(data map[string]any, key string) string {
	if val, ok := data[key]; ok && val != nil {
		return fmt.Sprint(val)
	}
	return ""
}

type DynamicStreamingExcel struct {
	f         *excelize.File
	rowID     *excelize.StreamWriter
	rowNum    int
	fieldKeys []string
	filename  string
}

func NewDynamicStreamingExcel(sheetName, filename string, headers, fieldKeys []string) (*DynamicStreamingExcel, error) {
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", sheetName)

	rowID, err := f.NewStreamWriter(sheetName)
	if err != nil {
		f.Close()
		return nil, err
	}

	headerRow := make([]any, len(headers))
	for i, h := range headers {
		headerRow[i] = h
	}
	if err := rowID.SetRow("A1", headerRow); err != nil {
		f.Close()
		return nil, err
	}

	return &DynamicStreamingExcel{
		f:         f,
		rowID:     rowID,
		rowNum:    2,
		fieldKeys: fieldKeys,
		filename:  filename,
	}, nil
}

func (s *DynamicStreamingExcel) WriteRows(rawDataList, errorMessages []string) error {
	for i := range rawDataList {
		var rawData map[string]any
		if err := json.Unmarshal([]byte(rawDataList[i]), &rawData); err != nil {
			rawData = make(map[string]any)
		}

		row := make([]any, len(s.fieldKeys))
		for colIdx, key := range s.fieldKeys {
			if key == "@error" {
				row[colIdx] = errorMessages[i]
			} else {
				row[colIdx] = getValueFromJSON(rawData, key)
			}
		}

		cell, _ := excelize.CoordinatesToCellName(1, s.rowNum)
		if err := s.rowID.SetRow(cell, row); err != nil {
			return err
		}
		s.rowNum++
	}

	return nil
}

func (s *DynamicStreamingExcel) Close() ([]byte, string, error) {
	if err := s.rowID.Flush(); err != nil {
		s.f.Close()
		return nil, "", err
	}

	buffer, err := s.f.WriteToBuffer()
	s.f.Close()
	if err != nil {
		return nil, "", err
	}

	return buffer.Bytes(), s.filename, nil
}

func GenerateDynamicExcelStreaming(filename string, headers, fieldKeys []string, fetchBatches func(func([]string, []string) bool) error) ([]byte, string, error) {
	writer, err := NewDynamicStreamingExcel("Failed Items", filename, headers, fieldKeys)
	if err != nil {
		return nil, "", err
	}

	if err = fetchBatches(func(rawDataList, errorMessages []string) bool {
		if err := writer.WriteRows(rawDataList, errorMessages); err != nil {
			return false
		}
		return true
	}); err != nil {
		writer.f.Close()
		return nil, "", err
	}

	body, filename, err := writer.Close()
	return body, filename, err
}
