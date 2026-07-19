// Package cursor implements storage-agnostic cursor-based (keyset) pagination:
// opaque base64 cursor encode/decode plus GetPagination, which trims a fetched
// page and derives the next/prev cursors.
//
// GetPagination post-processes rows you fetch; it does not run the query. The
// caller must: fetch Limit+1 rows (the extra one detects a further page); read
// Cursor.Next to pick the scan direction (forward ">", backward "<" on the sort
// key); and on a backward scan (Next == false) flip the query's ORDER BY, not
// just the operator — GetPagination reverses that page back into display order.
// See the package example for a full backend implementation.
//
// Cursors carry a single keyset value, so sort on a unique key (or append a
// tiebreaker); a non-unique sort column can skip or duplicate rows across pages.
package cursor

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"slices"
	"time"
)

type (
	// CurPage is implemented by row types that can produce the value used to
	// build the next/prev cursor (typically the row's primary key or sort key).
	CurPage interface {
		Cursor() any
	}

	// Cursor is the decoded form of the opaque string passed between client and
	// server. ID is the keyset value; Next indicates the scan direction that
	// produced it (true = forward/next page, false = backward/prev page).
	Cursor struct {
		ID   any  `json:"id"`
		Next bool `json:"next"`
	}

	// PaginationRequest describes a fetched page prior to trimming/reversal.
	//
	//   - Cursor: the raw opaque cursor from the client ("" on the first page).
	//   - Limit:  the page size the client asked for.
	//   - Next:   the scan direction, taken from the decoded cursor's Next field.
	//   - Data:   the fetched rows — MUST contain up to Limit+1 rows (over-fetch
	//     by one) so a further page can be detected. See the package "Query
	//     contract" docs.
	PaginationRequest[T CurPage] struct {
		Cursor string
		Limit  int
		Next   bool
		Data   []T
	}

	// PaginationResponse is the trimmed page plus opaque next/prev cursors
	// (empty string when there is no further page in that direction).
	PaginationResponse[T any] struct {
		Data []T
		Next string
		Prev string
	}
)

// GetID returns the cursor's keyset value, coercing JSON-decoded types back to
// their natural Go form: a numeric ID becomes int64 when it fits (full int64
// precision is preserved, so Snowflake-style IDs are safe) and otherwise
// float64; an RFC3339 string becomes time.Time; any other string is returned
// as-is.
func (c *Cursor) GetID() (any, error) {
	switch v := c.ID.(type) {
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i, nil
		}
		if f, err := v.Float64(); err == nil {
			return f, nil
		}
		return nil, errors.New("cursor: invalid numeric id")
	case float64:
		// Only occurs for a Cursor decoded without UseNumber; kept for safety.
		return int64(v), nil
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t, nil
		}
		return v, nil
	default:
		return nil, errors.New("unknown cursor")
	}
}

// CreateCursor builds a Cursor from a keyset value and scan direction.
func CreateCursor(id any, next bool) *Cursor {
	return &Cursor{
		ID:   id,
		Next: next,
	}
}

// GeneratePager wraps data with the encoded next/prev cursors.
func GeneratePager[T any](data []T, next *Cursor, prev *Cursor) PaginationResponse[T] {
	return PaginationResponse[T]{
		Data: data,
		Next: encodeCursor(next),
		Prev: encodeCursor(prev),
	}
}

func encodeCursor(cursor *Cursor) string {
	if cursor == nil {
		return ""
	}
	serializedCursor, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(serializedCursor)
}

// DecodeCursor decodes an opaque cursor string produced by GeneratePager.
// Numbers are decoded as json.Number (via UseNumber) so large integer IDs keep
// full int64 precision instead of being rounded through float64.
func DecodeCursor(cursor string) (*Cursor, error) {
	decodedCursor, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(bytes.NewReader(decodedCursor))
	dec.UseNumber()

	var cur Cursor
	if err = dec.Decode(&cur); err != nil {
		return nil, err
	}
	return &cur, nil
}

// GetPagination trims a fetched page (which must contain up to Limit+1 rows)
// down to Limit rows and derives the next/prev cursors from the request's
// cursor state (first page, forward scan, or backward scan).
func GetPagination[T CurPage](request PaginationRequest[T]) (response PaginationResponse[T]) {
	var (
		nextCur *Cursor
		prevCur *Cursor
	)

	data := request.Data
	dataLen := len(data)

	if dataLen == 0 {
		return
	}

	isFirstPage := request.Cursor == ""
	hasPagination := dataLen > request.Limit

	if hasPagination {
		data = data[:request.Limit]
	}

	if !isFirstPage && !request.Next {
		slices.Reverse(data)
	}

	if isFirstPage {
		if hasPagination {
			nextCur = CreateCursor(data[request.Limit-1].Cursor(), true)
		}
		return GeneratePager(data, nextCur, nil)
	}

	if request.Next {
		if hasPagination {
			nextCur = CreateCursor(data[request.Limit-1].Cursor(), true)
		}
		prevCur = CreateCursor(data[0].Cursor(), false)
		return GeneratePager(data, nextCur, prevCur)
	}

	if dataLen > request.Limit-1 {
		nextCur = CreateCursor(data[request.Limit-1].Cursor(), true)
	}

	if hasPagination {
		prevCur = CreateCursor(data[0].Cursor(), false)
	}

	return GeneratePager(data, nextCur, prevCur)
}
