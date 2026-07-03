package excel

import (
	"bytes"
	"io"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestStreamSheet_AppendRowAndBytes(t *testing.T) {
	sheet, err := NewStreamSheet("Data", []any{"ID", "Name"})
	if err != nil {
		t.Fatal(err)
	}
	defer sheet.Close()

	if err := sheet.AppendRow([]any{1, "Branch A"}); err != nil {
		t.Fatal(err)
	}
	if err := sheet.AppendRow([]any{2, "Branch B"}); err != nil {
		t.Fatal(err)
	}

	body, err := sheet.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 {
		t.Fatal("expected non-empty xlsx bytes")
	}

	f, err := excelize.OpenReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	v, err := f.GetCellValue("Data", "B3")
	if err != nil {
		t.Fatal(err)
	}
	if v != "Branch B" {
		t.Errorf("B3 = %q, want Branch B", v)
	}
}

func TestNewStreamSheetFromLayout_twoRows(t *testing.T) {
	sheet, err := NewStreamSheetFromLayout("S", HeaderLayout{
		Row1: []any{"Group"},
		Row2: []any{"Col1", "Col2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sheet.Close()

	if sheet.nextRow != 3 {
		t.Errorf("nextRow = %d, want 3", sheet.nextRow)
	}
}

func TestNewStreamSheet_emptyBody(t *testing.T) {
	sheet, err := NewStreamSheet("X", []any{"H"})
	if err != nil {
		t.Fatal(err)
	}
	defer sheet.Close()
	body, err := sheet.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if len(body) < 100 {
		t.Fatalf("unexpected small body: %d bytes", len(body))
	}
	_, err = excelize.OpenReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	_ = io.EOF
}
