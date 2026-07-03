package excel

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestGenerateDynamicExcelStreaming_validWorkbook(t *testing.T) {
	body, filename, err := GenerateDynamicExcelStreaming(
		"error-items.xlsx",
		[]string{"Branch Code", "District Code", "Error Message"},
		[]string{"branch_code", "district_code", "@error"},
		func(writeBatch func([]string, []string) bool) error {
			ok := writeBatch(
				[]string{`{"branch_code":"B001","district_code":"D001"}`},
				[]string{"invalid district"},
			)
			if !ok {
				t.Fatal("writeBatch returned false")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if filename != "error-items.xlsx" {
		t.Fatalf("filename = %q, want error-items.xlsx", filename)
	}

	f, err := excelize.OpenReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	assertCellValue(t, f, "Failed Items", "A1", "Branch Code")
	assertCellValue(t, f, "Failed Items", "A2", "B001")
	assertCellValue(t, f, "Failed Items", "B2", "D001")
	assertCellValue(t, f, "Failed Items", "C2", "invalid district")
}

func assertCellValue(t *testing.T, f *excelize.File, sheet, cell, want string) {
	t.Helper()

	got, err := f.GetCellValue(sheet, cell)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s = %q, want %q", cell, got, want)
	}
}
