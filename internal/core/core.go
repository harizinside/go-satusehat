// Package core holds the types and helpers shared by the root satusehat
// package and the generated per-group packages (fhir, rawatjalan, ...).
// It sits below all of them so none has to import another; the root package
// re-exports every type here under the same name.
package core

import (
	"bytes"
	"context"
	"encoding/json"
)

// Resource is a loosely-typed FHIR R4 resource: `{resourceType, ...}` plus
// arbitrary fields.
type Resource = map[string]any

// Bundle is a FHIR searchset/transaction/collection Bundle. Entries are kept
// as `json.RawMessage` so the caller can decode each `entry.resource` with
// the right concrete type without re-marshalling.
type Bundle struct {
	ResourceType string        `json:"resourceType"`
	Type         string        `json:"type,omitempty"`
	Total        int           `json:"total,omitempty"`
	Link         []Resource    `json:"link,omitempty"`
	Entry        []BundleEntry `json:"entry,omitempty"`
}

// BundleEntry is one entry inside a Bundle. `Resource` is raw JSON so the
// caller decides how to decode it.
type BundleEntry struct {
	FullURL  string          `json:"fullUrl,omitempty"`
	Resource json.RawMessage `json:"resource,omitempty"`
	Search   *BundleSearch   `json:"search,omitempty"`
}

// BundleSearch is the FHIR Bundle.entry.search object (mode/score).
type BundleSearch struct {
	Mode  string  `json:"mode,omitempty"`
	Score float64 `json:"score,omitempty"`
}

// PatchOp is one FHIR JSON Patch operation (RFC 6902). `Value` is `any`
// because patch values can be strings, numbers, objects, arrays, booleans or
// null; `From` is set for `move`/`copy`.
type PatchOp struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
	From  string `json:"from,omitempty"`
}

// RequestOptions mirrors the TS `RequestOptions` shape: `query`, `body`,
// `headers`, `auth`. `Auth` defaults to true; pass `Auth: Ptr(false)` for the
// OAuth2 token endpoint (which must NOT carry a Bearer header).
type RequestOptions struct {
	Query   any
	Body    any
	Headers map[string]string
	Auth    *bool
}

// Requester is what generated services call; *satusehat.Client implements it.
type Requester interface {
	Request(ctx context.Context, method, pathOrURL string, opts *RequestOptions) (any, error)
}

// AsResource adapts Request's decoded JSON to a Resource (nil when the
// response body was empty or not an object).
func AsResource(v any, err error) (Resource, error) {
	if err != nil {
		return nil, err
	}
	r, _ := v.(map[string]any)
	return r, nil
}

// AsBundle adapts Request's decoded JSON to a *Bundle.
func AsBundle(v any, err error) (*Bundle, error) {
	if err != nil {
		return nil, err
	}
	return DecodeTo[Bundle](v)
}

// MarshalJSON is json.Marshal without HTML escaping (matching Node's
// `JSON.stringify` — FHIR resources contain `<`, `>`, `&` in narrative divs
// and address lines that we'd rather keep readable).
func MarshalJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// DecodeTo roundtrips an `any` (from `Request`) into T via JSON.
func DecodeTo[T any](raw any) (*T, error) {
	data, err := MarshalJSON(raw)
	if err != nil {
		return nil, err
	}
	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MarshalWithExtra encodes payload and merges extra into the top-level object
// (extra wins on conflict), mirroring TS `[key: string]: unknown` inputs.
// payload must be a method-less alias of the caller's type, otherwise its
// MarshalJSON would recurse back into here.
func MarshalWithExtra(payload any, extra map[string]any) ([]byte, error) {
	if len(extra) == 0 {
		return MarshalJSON(payload)
	}
	m, err := DecodeTo[map[string]any](payload)
	if err != nil {
		return nil, err
	}
	for k, v := range extra {
		(*m)[k] = v
	}
	return MarshalJSON(*m)
}
