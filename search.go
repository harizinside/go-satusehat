package satusehat

import (
	"context"
	"net/url"

	"github.com/harizinside/go-satusehat/coding"
)

// PersonSearchParams holds FHIR search parameters shared by Patient and
// Practitioner lookups. All fields are optional; empty values are skipped
// (see `appendQuery`).
type PersonSearchParams struct {
	Identifier  string            `url:"identifier,omitempty"`
	Name        string            `url:"name,omitempty"`
	Birthdate   string            `url:"birthdate,omitempty"`
	Gender      string            `url:"gender,omitempty"`
	ID          string            `url:"_id,omitempty"`
	LastUpdated string            `url:"_lastUpdated,omitempty"`
	Count       int               `url:"_count,omitempty"`
	Extra       map[string]string `url:"-"`
}

// NIKIdentifier builds the `identifier` value for a NIK lookup:
// `https://fhir.kemkes.go.id/id/nik|<nik>`.
func NIKIdentifier(nik string) string {
	return coding.IdentifierSystem.Nik + "|" + nik
}

// SearchResource is the shared implementation for `SearchPatient` /
// `SearchPractitioner`. Pass `resource` as "Patient" or "Practitioner".
func searchResource(ctx context.Context, c *Client, resource string, p PersonSearchParams) (any, error) {
	q := map[string]any{}
	if p.Identifier != "" {
		q["identifier"] = p.Identifier
	}
	if p.Name != "" {
		q["name"] = p.Name
	}
	if p.Birthdate != "" {
		q["birthdate"] = p.Birthdate
	}
	if p.Gender != "" {
		q["gender"] = p.Gender
	}
	if p.ID != "" {
		q["_id"] = p.ID
	}
	if p.LastUpdated != "" {
		q["_lastUpdated"] = p.LastUpdated
	}
	if p.Count != 0 {
		q["_count"] = p.Count
	}
	for k, v := range p.Extra {
		if v != "" {
			q[k] = v
		}
	}
	return c.Request(ctx, "GET", "/"+resource, &RequestOptions{Query: q})
}

// SearchPatient GETs `/Patient?identifier=...` — returns a FHIR searchset
// Bundle.
func SearchPatient(ctx context.Context, c *Client, p PersonSearchParams) (any, error) {
	return searchResource(ctx, c, "Patient", p)
}

// GetPatientById GETs `/Patient/{id}` — returns the Patient resource.
func GetPatientById(ctx context.Context, c *Client, id string) (any, error) {
	return c.Request(ctx, "GET", "/Patient/"+url.PathEscape(id), nil)
}

// SearchPractitioner GETs `/Practitioner?identifier=...` — returns a FHIR
// searchset Bundle.
func SearchPractitioner(ctx context.Context, c *Client, p PersonSearchParams) (any, error) {
	return searchResource(ctx, c, "Practitioner", p)
}

// GetPractitionerById GETs `/Practitioner/{id}` — returns the Practitioner.
func GetPractitionerById(ctx context.Context, c *Client, id string) (any, error) {
	return c.Request(ctx, "GET", "/Practitioner/"+url.PathEscape(id), nil)
}
