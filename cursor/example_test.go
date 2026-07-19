package cursor_test

import (
	"fmt"

	"github.com/hallode/golib/v2/cursor"
)

// Item is a row type from a repository, ordered by ID. Implementing CurPage
// is the only requirement for a type to work with GetPagination.
type Item struct {
	ID   int
	Name string
}

func (i Item) Cursor() any { return i.ID }

var allItems = []Item{
	{ID: 1, Name: "alpha"},
	{ID: 2, Name: "bravo"},
	{ID: 3, Name: "charlie"},
	{ID: 4, Name: "delta"},
	{ID: 5, Name: "echo"},
}

// fetchPage stands in for a repository query. The key contract callers must
// follow: decode the incoming cursor to know the scan direction, then fetch
// limit+1 rows in that direction so GetPagination can detect whether another
// page exists.
func fetchPage(encodedCursor string, limit int) cursor.PaginationResponse[Item] {
	if encodedCursor == "" {
		return cursor.GetPagination(cursor.PaginationRequest[Item]{
			Limit: limit,
			Data:  take(allItems, limit+1),
		})
	}

	cur, err := cursor.DecodeCursor(encodedCursor)
	if err != nil {
		panic(err)
	}
	rawID, err := cur.GetID()
	if err != nil {
		panic(err)
	}
	id := int(rawID.(int64)) // GetID returns int64 for numeric cursor values.

	// A real backend does this in the query: forward scan keeps ORDER BY id ASC
	// and filters id > cursor; backward scan flips to ORDER BY id DESC (here:
	// iterate in reverse) and filters id < cursor. GetPagination reverses the
	// backward page back into ascending display order.
	var filtered []Item
	if cur.Next {
		for _, it := range allItems {
			if it.ID > id {
				filtered = append(filtered, it)
			}
		}
	} else {
		for i := len(allItems) - 1; i >= 0; i-- {
			if allItems[i].ID < id {
				filtered = append(filtered, allItems[i])
			}
		}
	}

	return cursor.GetPagination(cursor.PaginationRequest[Item]{
		Cursor: encodedCursor,
		Limit:  limit,
		Next:   cur.Next,
		Data:   take(filtered, limit+1),
	})
}

func take(items []Item, n int) []Item {
	if len(items) > n {
		return items[:n]
	}
	return items
}

func ids(items []Item) []int {
	out := make([]int, len(items))
	for i, it := range items {
		out[i] = it.ID
	}
	return out
}

// Example demonstrates the full request/response cycle: a client walks
// forward through pages using each response's Next cursor, then walks back
// using Prev, landing on the same page it came from.
func ExampleGetPagination() {
	const limit = 2

	page1 := fetchPage("", limit)
	fmt.Println("page1:", ids(page1.Data), "hasNext:", page1.Next != "")

	page2 := fetchPage(page1.Next, limit)
	fmt.Println("page2:", ids(page2.Data), "hasNext:", page2.Next != "", "hasPrev:", page2.Prev != "")

	page3 := fetchPage(page2.Next, limit)
	fmt.Println("page3:", ids(page3.Data), "hasNext:", page3.Next != "")

	backToPage2 := fetchPage(page3.Prev, limit)
	fmt.Println("back to page2:", ids(backToPage2.Data))

	// Output:
	// page1: [1 2] hasNext: true
	// page2: [3 4] hasNext: true hasPrev: true
	// page3: [5] hasNext: false
	// back to page2: [3 4]
}
