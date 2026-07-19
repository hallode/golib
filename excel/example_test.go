package excel_test

import (
	"fmt"

	"github.com/hallode/golib/excel"
)

func ExampleGenerateExcel() {
	data, filename, err := excel.GenerateExcel(&excel.ExportConfig{
		SheetName: "Users",
		Filename:  "users.xlsx",
		Headers:   []string{"ID", "Name"},
		Rows: [][]string{
			{"1", "Ada"},
			{"2", "Linus"},
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("file:", filename)
	fmt.Println("non-empty:", len(data) > 0)
	// Output:
	// file: users.xlsx
	// non-empty: true
}
