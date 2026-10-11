package satusehat

import (
	"context"
)

// BundleRequestMethod is the FHIR Bundle entry.request.method value.
type BundleRequestMethod string

const (
	BundlePOST   BundleRequestMethod = "POST"
	BundlePUT    BundleRequestMethod = "PUT"
	BundlePATCH  BundleRequestMethod = "PATCH"
	BundleGET    BundleRequestMethod = "GET"
	BundleDELETE BundleRequestMethod = "DELETE"
)

// BundleEntryInput is one entry in a transaction Bundle.
type BundleEntryInput struct {
	// Resource is the FHIR resource to include (`resourceType` + fields).
	Resource Resource `json:"resource"`
	// Method defaults to POST (create) — the shape SATUSEHAT's examples use.
	Method BundleRequestMethod `json:"method,omitempty"`
	// URL is required for PUT/PATCH/GET/DELETE; defaults to
	// `Resource.ResourceType` when empty.
	URL string `json:"url,omitempty"`
	// FullURL is the stable local reference (e.g. `urn:uuid:...`) referenced
	// from other entries in the same Bundle.
	FullURL string `json:"fullUrl,omitempty"`
}

// FhirTransactionBundle is a fully-resolved FHIR transaction Bundle.
type FhirTransactionBundle struct {
	ResourceType string                       `json:"resourceType"`
	Type         string                       `json:"type"`
	Entry        []FhirTransactionBundleEntry `json:"entry"`
}

// FhirTransactionBundleEntry is one entry in a FhirTransactionBundle.
type FhirTransactionBundleEntry struct {
	FullURL  string                 `json:"fullUrl,omitempty"`
	Resource Resource               `json:"resource"`
	Request  FhirTransactionRequest `json:"request"`
}

// FhirTransactionRequest is Bundle.entry.request.
type FhirTransactionRequest struct {
	Method BundleRequestMethod `json:"method"`
	URL    string              `json:"url"`
}

// BuildTransactionBundle assembles a FHIR transaction Bundle from typed
// entries.
func BuildTransactionBundle(entries []BundleEntryInput) FhirTransactionBundle {
	out := FhirTransactionBundle{
		ResourceType: "Bundle",
		Type:         "transaction",
		Entry:        make([]FhirTransactionBundleEntry, 0, len(entries)),
	}
	for _, e := range entries {
		method := e.Method
		if method == "" {
			method = BundlePOST
		}
		url := e.URL
		if url == "" {
			if rt, ok := e.Resource["resourceType"].(string); ok {
				url = rt
			}
		}
		entry := FhirTransactionBundleEntry{
			Resource: e.Resource,
			Request:  FhirTransactionRequest{Method: method, URL: url},
		}
		if e.FullURL != "" {
			entry.FullURL = e.FullURL
		}
		out.Entry = append(out.Entry, entry)
	}
	return out
}

// SubmitBundle POSTs the given Bundle to `/`. The server processes entries
// atomically and resolves internal `urn:uuid:` references.
func SubmitBundle(ctx context.Context, client *Client, bundle FhirTransactionBundle) (any, error) {
	return client.Request(ctx, "POST", "/", &RequestOptions{Body: bundle})
}

// SubmitVisitBundle is the convenience build + submit wrapper.
func SubmitVisitBundle(ctx context.Context, client *Client, entries []BundleEntryInput) (any, error) {
	return SubmitBundle(ctx, client, BuildTransactionBundle(entries))
}
