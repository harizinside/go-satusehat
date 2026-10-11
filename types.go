// Package satusehat is a Go port of the @harizinside/satusehat Node SDK:
// every SATU SEHAT (Kemkes) Platform endpoint behind one zero-dependency
// client. Token caching is intentionally left to the caller — `GetAccessToken`
// mints a fresh token, and you persist it (Redis, DB, in-memory) and reattach
// via `Config.Token` / `SetToken` on every request, exactly like the Node
// version.
//
// All 384+ generated functions hang off namespaced sub-clients reachable from
// `c.FHIR.*`, `c.RawatJalan.*`, `c.Gigi.*`, `c.Farmasi.*`,
// `c.KlaimSwasta.*`, and `c.KlaimBpjs.*`; see README for the full list.
package satusehat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Resource is a loosely-typed FHIR R4 resource: `{resourceType, ...}` plus
// arbitrary fields. Use this when you don't have (or don't need) a typed
// struct; the generated code uses this for endpoints where the returned type
// isn't worth a dedicated struct.
type Resource = map[string]any

// Bundle is a FHIR searchset/transaction/collection Bundle. Entries are kept
// as `json.RawMessage` so the caller can decode each `entry.resource` with
// the right concrete type (`Resource`, `Coding`, a generated *Body, …) without
// re-marshalling.
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

// APIError is returned for any non-2xx SATU SEHAT response and for transport
// failures (StatusCode 0); match it with errors.As. The first 512 bytes of the
// body are embedded into `Error()` so the OperationOutcome (or any error payload) shows up in logs without
// forcing a caller to dig into `Body`.
type APIError struct {
	// StatusCode is `0` when the failure happened before a response arrived
	// (DNS, connection refused, TLS error, ...).
	StatusCode int
	// Method / URL mirror what the request was, for logs.
	Method string
	URL    string
	// Body is the raw response body.
	Body []byte
	// Headers are the response headers, lower-cased keys.
	Headers map[string]string
	// Err is the transport error when StatusCode is 0 (or the body read
	// failed); errors.Is(err, context.Canceled) etc. see through it.
	Err error
}

// Unwrap exposes the transport error to errors.Is / errors.As.
func (e *APIError) Unwrap() error { return e.Err }

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "satusehat: %s %s: %d", e.Method, e.URL, e.StatusCode)
	if e.Err != nil {
		fmt.Fprintf(&b, ": %v", e.Err)
	}
	if snippet := bytes.TrimSpace(e.Body); len(snippet) > 0 {
		limit := 512
		if len(snippet) < limit {
			limit = len(snippet)
		}
		fmt.Fprintf(&b, " — %s", snippet[:limit])
	}
	return b.String()
}
