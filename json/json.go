// Package json is a drop-in replacement for encoding/json with a configurable
// backend. Call Init("sonic") at startup to enable bytedance/sonic; the default
// is the standard library.
package json

import (
	"bytes"
	stdjson "encoding/json"
	"io"
)

// Re-exported types so callers need only import this package.
type (
	Decoder    = stdjson.Decoder
	Encoder    = stdjson.Encoder
	RawMessage = stdjson.RawMessage
	Number     = stdjson.Number
	Delim      = stdjson.Delim
	Token      = stdjson.Token
)

// Marshal encodes v to JSON using the active codec.
func Marshal(v any) ([]byte, error) { return _codec.Load().marshal(v) }

// Unmarshal decodes JSON data into v using the active codec.
func Unmarshal(data []byte, v any) error { return _codec.Load().unmarshal(data, v) }

// Valid reports whether data is valid JSON.
func Valid(data []byte) bool { return stdjson.Valid(data) }

// Compact appends the JSON-encoded src without insignificant whitespace to dst.
func Compact(dst *bytes.Buffer, src []byte) error { return stdjson.Compact(dst, src) }

// NewDecoder returns a streaming JSON decoder reading from r.
func NewDecoder(r io.Reader) *stdjson.Decoder { return stdjson.NewDecoder(r) }

// NewEncoder returns a streaming JSON encoder writing to w.
func NewEncoder(w io.Writer) *stdjson.Encoder { return stdjson.NewEncoder(w) }
