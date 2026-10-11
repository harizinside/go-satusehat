package satusehat

import (
	"context"

	"github.com/harizinside/go-satusehat/internal/core"
)

// RmeLinkInput is the body of an RME health-link request.
type RmeLinkInput struct {
	PatientID        string         `json:"patient_id"`
	PatientName      string         `json:"patient_name"`
	PractitionerID   string         `json:"practitioner_id"`
	PractitionerName string         `json:"practitioner_name"`
	OrganizationID   string         `json:"organization_id"`
	OrganizationName string         `json:"organization_name"`
	Extra            map[string]any `json:"-"`
}

// MarshalJSON merges Extra into the top-level object.
func (r RmeLinkInput) MarshalJSON() ([]byte, error) {
	type plain RmeLinkInput
	return core.MarshalWithExtra(plain(r), r.Extra)
}

// RmeOptions lets callers override the RME host.
type RmeOptions struct {
	BaseURL string
}

// CreateHealthLink POSTs to `{baseUrl}/ssrme/v2/ntl/chl` ("Membuat consent
// health link"): create the patient's consented health link into the
// national RME.
func CreateHealthLink(ctx context.Context, client *Client, body RmeLinkInput, options RmeOptions) (any, error) {
	return client.RequestURL(ctx, "POST", rmeURL(client, "chl", options.BaseURL), &RequestOptions{Body: body})
}

// ShowHealthLink POSTs to `{baseUrl}/ssrme/v2/ntl/shl` ("Membuka RME
// Nasional"): open the national RME for the patient (returns the deep-link
// payload).
func ShowHealthLink(ctx context.Context, client *Client, body RmeLinkInput, options RmeOptions) (any, error) {
	return client.RequestURL(ctx, "POST", rmeURL(client, "shl", options.BaseURL), &RequestOptions{Body: body})
}

func rmeURL(client *Client, path string, override string) string {
	base := override
	if base == "" {
		base = client.config.RmeBaseURL
	}
	return trimTrailingSlash(base) + "/ssrme/v2/ntl/" + path
}
