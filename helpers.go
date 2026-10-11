package satusehat

import (
	"bytes"
	"encoding/json"
)

// jsonMarshal is a thin wrapper that disables HTML escaping (matching Node's
// `JSON.stringify` behavior — FHIR resources contain `<`, `>`, `&` in things
// like narrative divs and address lines that we'd rather keep readable).
func jsonMarshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	out := buf.Bytes()
	if n := len(out); n > 0 && out[n-1] == '\n' {
		out = out[:n-1]
	}
	return out, nil
}

// toMap JSON-roundtrips an arbitrary struct into a map[string]any so we can
// merge Extra keys into it. Used by the input types that have a
// `[key: string]: unknown` index signature in TS.
func toMap(v any) (map[string]any, error) {
	data, err := jsonMarshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// decodeToType roundtrips an `any` (from `Client.Request`) into T via JSON.
func decodeToType[T any](raw any) (*T, error) {
	data, err := jsonMarshal(raw)
	if err != nil {
		return nil, err
	}
	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// trimTrailingSlash is the inverse of `strings.TrimRight(s, "/")` minus the
// awkward empty-result corner case.
func trimTrailingSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

// Ptr returns a pointer to v, for the pointer fields in generated request
// bodies: `Active: satusehat.Ptr(false)`.
func Ptr[T any](v T) *T { return &v }

// asResource adapts Request's decoded JSON to a Resource (nil when the
// response body was empty or not an object).
func asResource(v any, err error) (Resource, error) {
	if err != nil {
		return nil, err
	}
	r, _ := v.(map[string]any)
	return r, nil
}

// asBundle adapts Request's decoded JSON to a *Bundle.
func asBundle(v any, err error) (*Bundle, error) {
	if err != nil {
		return nil, err
	}
	return decodeToType[Bundle](v)
}
