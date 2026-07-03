package excel

type MergeRange struct {
	TopLeft, BottomRight string
}

type HeaderLayout struct {
	Row1, Row2 []any
	Merges     []MergeRange
	ColWidths  map[int]float64 // 1-based column index -> width
}

func NewStreamSheetFromLayout(sheetName string, layout HeaderLayout) (*StreamSheet, error) {
	s, err := NewEmptyStreamSheet(sheetName)
	if err != nil {
		return nil, err
	}

	for col, w := range layout.ColWidths {
		if err := s.SetColWidth(col, w); err != nil {
			s.Close()
			return nil, err
		}
	}

	for _, m := range layout.Merges {
		if err := s.MergeCell(m.TopLeft, m.BottomRight); err != nil {
			s.Close()
			return nil, err
		}
	}

	if len(layout.Row1) > 0 {
		if err := s.SetRowAt("A1", layout.Row1); err != nil {
			s.Close()
			return nil, err
		}
	}

	if len(layout.Row2) > 0 {
		if err := s.SetRowAt("A2", layout.Row2); err != nil {
			s.Close()
			return nil, err
		}
		s.SetNextDataRow(3)
	} else if len(layout.Row1) > 0 {
		s.SetNextDataRow(2)
	}

	return s, nil
}
