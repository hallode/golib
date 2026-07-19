package json_test

import (
	"strings"
	"sync"
	"testing"

	customjson "github.com/hallode/golib/v2/json"
)

type sampleStruct struct {
	Name    string   `json:"name"`
	Age     int      `json:"age"`
	Score   float64  `json:"score,omitempty"`
	Active  bool     `json:"active"`
	Tags    []string `json:"tags,omitempty"`
	Ignored string   `json:"-"`
}

var codecs = []struct {
	name string
	kind string
}{
	{"default", "default"},
	{"sonic", "sonic"},
	{"empty_falls_back", ""},
	{"unknown_falls_back", "unknown"},
}

func TestMarshal(t *testing.T) {
	input := sampleStruct{
		Name:    "test",
		Age:     30,
		Score:   9.5,
		Active:  true,
		Tags:    []string{"a", "b"},
		Ignored: "should-not-appear",
	}

	for _, c := range codecs {
		t.Run(c.name, func(t *testing.T) {
			customjson.Init(c.kind)

			data, err := customjson.Marshal(input)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}
			if !customjson.Valid(data) {
				t.Fatal("output is not valid JSON")
			}

			// Round-trip check.
			var out sampleStruct
			if err := customjson.Unmarshal(data, &out); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			if out.Name != input.Name || out.Age != input.Age || out.Active != input.Active {
				t.Errorf("round-trip mismatch: got %+v", out)
			}
			if out.Ignored != "" {
				t.Error("ignored field should not be populated")
			}
		})
	}
}

func TestUnmarshal(t *testing.T) {
	data := []byte(`{"name":"hello","age":42,"active":false}`)

	for _, c := range codecs {
		t.Run(c.name, func(t *testing.T) {
			customjson.Init(c.kind)

			var out sampleStruct
			if err := customjson.Unmarshal(data, &out); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			if out.Name != "hello" || out.Age != 42 || out.Active != false {
				t.Errorf("unexpected result: %+v", out)
			}
		})
	}
}

func TestUnmarshal_Invalid(t *testing.T) {
	for _, c := range codecs {
		t.Run(c.name, func(t *testing.T) {
			customjson.Init(c.kind)

			var out sampleStruct
			if err := customjson.Unmarshal([]byte(`{invalid`), &out); err == nil {
				t.Fatal("expected error for invalid JSON")
			}
		})
	}
}

func TestMarshal_Nil(t *testing.T) {
	for _, c := range codecs {
		t.Run(c.name, func(t *testing.T) {
			customjson.Init(c.kind)

			data, err := customjson.Marshal(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(data) != "null" {
				t.Errorf("expected null, got %s", data)
			}
		})
	}
}

func TestMarshal_HTMLEscape(t *testing.T) {
	// sonic.ConfigStd must escape HTML characters like encoding/json.
	input := map[string]string{"url": "https://example.com?a=1&b=2<>"}

	for _, c := range codecs {
		t.Run(c.name, func(t *testing.T) {
			customjson.Init(c.kind)

			data, err := customjson.Marshal(input)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}
			// encoding/json and sonic.ConfigStd both escape & < >
			s := string(data)
			for _, ch := range []string{"&", "<", ">"} {
				if strings.Contains(s, ch) {
					t.Errorf("codec %q: HTML char %q not escaped in output: %s", c.kind, ch, s)
				}
			}
		})
	}
}

// TestMarshal_SonicFallback verifies sonic falls back to stdjson for types
// sonic cannot handle (e.g. a channel) without crashing.
func TestMarshal_SonicFallback(t *testing.T) {
	customjson.Init("sonic")

	ch := make(chan int)
	_, err := customjson.Marshal(ch)
	if err == nil {
		t.Fatal("expected error marshalling channel, got nil")
	}
}

// TestInit_Concurrent verifies no data race when Init and Marshal run concurrently.
func TestInit_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	kinds := []string{"sonic", "default", "sonic", "default"}
	for _, k := range kinds {
		wg.Go(func() { customjson.Init(k) })
	}
	// Also hammer Marshal concurrently.
	for range 8 {
		wg.Go(func() {
			_, _ = customjson.Marshal(sampleStruct{Name: "x"})
		})
	}
	wg.Wait()
}

func TestRawMessage(t *testing.T) {
	type wrapper struct {
		Data customjson.RawMessage `json:"data"`
	}
	raw := customjson.RawMessage(`{"key":"value"}`)
	w := wrapper{Data: raw}

	for _, c := range codecs {
		t.Run(c.name, func(t *testing.T) {
			customjson.Init(c.kind)

			data, err := customjson.Marshal(w)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}
			var out wrapper
			if err := customjson.Unmarshal(data, &out); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			if string(out.Data) != string(raw) {
				t.Errorf("RawMessage mismatch: got %s want %s", out.Data, raw)
			}
		})
	}
}
