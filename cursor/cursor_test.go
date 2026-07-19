package cursor_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hallode/golib/cursor"
)

type row struct{ id int }

func (r row) Cursor() any { return r.id }

func rows(ids ...int) []row {
	out := make([]row, len(ids))
	for i, id := range ids {
		out[i] = row{id: id}
	}
	return out
}

func decode(t *testing.T, encoded string) *cursor.Cursor {
	t.Helper()
	c, err := cursor.DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("DecodeCursor(%q) error: %v", encoded, err)
	}
	return c
}

func TestCreateCursor(t *testing.T) {
	c := cursor.CreateCursor(42, true)
	if c.ID != 42 || c.Next != true {
		t.Fatalf("CreateCursor(42, true) = %+v", c)
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	resp := cursor.GeneratePager(rows(1), cursor.CreateCursor(5, true), cursor.CreateCursor(1, false))

	next := decode(t, resp.Next)
	if id, _ := next.GetID(); id != int64(5) || !next.Next {
		t.Fatalf("decoded next cursor = %+v", next)
	}

	prev := decode(t, resp.Prev)
	if id, _ := prev.GetID(); id != int64(1) || prev.Next {
		t.Fatalf("decoded prev cursor = %+v", prev)
	}
}

func TestGeneratePagerNilCursorsEncodeEmpty(t *testing.T) {
	resp := cursor.GeneratePager(rows(1), nil, nil)
	if resp.Next != "" || resp.Prev != "" {
		t.Fatalf("expected empty next/prev, got Next=%q Prev=%q", resp.Next, resp.Prev)
	}
}

func TestDecodeCursorInvalidBase64(t *testing.T) {
	if _, err := cursor.DecodeCursor("not-base64!!!"); err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecodeCursorInvalidJSON(t *testing.T) {
	// valid base64, but not valid JSON once decoded.
	if _, err := cursor.DecodeCursor("bm90LWpzb24="); err == nil {
		t.Fatal("expected error for invalid JSON payload")
	}
}

func TestCursorGetID(t *testing.T) {
	tests := []struct {
		name    string
		id      any
		want    any
		wantErr bool
	}{
		{name: "json.Number becomes int64", id: json.Number("7"), want: int64(7)},
		{name: "float64 fallback becomes int64", id: float64(7), want: int64(7)},
		{name: "RFC3339 string becomes time.Time", id: "2024-01-02T15:04:05Z", want: time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC)},
		{name: "non-date string stays string", id: "abc-123", want: "abc-123"},
		{name: "unsupported type errors", id: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cursor.Cursor{ID: tt.id}
			got, err := c.GetID()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tm, ok := tt.want.(time.Time); ok {
				if !got.(time.Time).Equal(tm) {
					t.Fatalf("GetID() = %v, want %v", got, tm)
				}
				return
			}
			if got != tt.want {
				t.Fatalf("GetID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncodeDecodeInt64Precision(t *testing.T) {
	// A Snowflake-style ID larger than 2^53 would lose precision if decoded
	// through float64. It must survive an encode/decode round-trip intact.
	const snowflake int64 = 1503020304050607080

	resp := cursor.GeneratePager(rows(1), cursor.CreateCursor(snowflake, true), nil)

	got, err := decode(t, resp.Next).GetID()
	if err != nil {
		t.Fatalf("GetID error: %v", err)
	}
	if got != snowflake {
		t.Fatalf("GetID() = %v (%T), want %d (int64)", got, got, snowflake)
	}
}

func TestGetPagination(t *testing.T) {
	tests := []struct {
		name       string
		data       []row
		reqCursor  string
		limit      int
		next       bool
		wantIDs    []int
		wantHasNxt bool
		wantHasPrv bool
	}{
		{
			name:       "empty data returns zero value",
			data:       nil,
			limit:      2,
			wantIDs:    nil,
			wantHasNxt: false,
			wantHasPrv: false,
		},
		{
			name:       "first page without extra row has no next",
			data:       rows(1, 2),
			limit:      2,
			wantIDs:    []int{1, 2},
			wantHasNxt: false,
			wantHasPrv: false,
		},
		{
			name:       "first page with extra row has next only",
			data:       rows(1, 2, 3),
			limit:      2,
			wantIDs:    []int{1, 2},
			wantHasNxt: true,
			wantHasPrv: false,
		},
		{
			name:       "forward scan middle page has next and prev",
			data:       rows(3, 4, 5),
			reqCursor:  "cur",
			next:       true,
			limit:      2,
			wantIDs:    []int{3, 4},
			wantHasNxt: true,
			wantHasPrv: true,
		},
		{
			name:       "forward scan last page has prev only",
			data:       rows(3, 4),
			reqCursor:  "cur",
			next:       true,
			limit:      2,
			wantIDs:    []int{3, 4},
			wantHasNxt: false,
			wantHasPrv: true,
		},
		{
			name:       "backward scan middle page has next and prev, data reversed back to order",
			data:       rows(5, 4, 3),
			reqCursor:  "cur",
			next:       false,
			limit:      2,
			wantIDs:    []int{4, 5},
			wantHasNxt: true,
			wantHasPrv: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := cursor.GetPagination(cursor.PaginationRequest[row]{
				Cursor: tt.reqCursor,
				Limit:  tt.limit,
				Next:   tt.next,
				Data:   tt.data,
			})

			gotIDs := make([]int, len(resp.Data))
			for i, r := range resp.Data {
				gotIDs[i] = r.id
			}
			if len(gotIDs) != len(tt.wantIDs) {
				t.Fatalf("Data ids = %v, want %v", gotIDs, tt.wantIDs)
			}
			for i := range gotIDs {
				if gotIDs[i] != tt.wantIDs[i] {
					t.Fatalf("Data ids = %v, want %v", gotIDs, tt.wantIDs)
				}
			}

			if hasNext := resp.Next != ""; hasNext != tt.wantHasNxt {
				t.Fatalf("Next = %q (present=%v), want present=%v", resp.Next, hasNext, tt.wantHasNxt)
			}
			if hasPrev := resp.Prev != ""; hasPrev != tt.wantHasPrv {
				t.Fatalf("Prev = %q (present=%v), want present=%v", resp.Prev, hasPrev, tt.wantHasPrv)
			}
		})
	}
}
