package excel

import "testing"

func TestNextColumn(t *testing.T) {
	tests := []struct {
		col    string
		offset int
		want   string
	}{
		{"A", 0, "A"},
		{"A", 1, "B"},
		{"A", 8, "I"},
		{"Z", 1, "AA"},
	}
	for _, tt := range tests {
		if got := NextColumn(tt.col, tt.offset); got != tt.want {
			t.Errorf("NextColumn(%q, %d) = %q, want %q", tt.col, tt.offset, got, tt.want)
		}
	}
}
