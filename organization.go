package satusehat

import (
	"context"
	"net/url"

	"github.com/harizinside/go-satusehat/coding"
	"github.com/harizinside/go-satusehat/internal/core"
)

// OrganizationInput is a FHIR R4 Organization as accepted by SATUSEHAT
// (POST /Organization). All fields optional except `ResourceType` — see the
// SATUSEHAT FHIR docs for required combinations, e.g. a Poli organization
// needs `PartOf` pointing at its parent org.
type OrganizationInput struct {
	ResourceType string         `json:"resourceType"`
	Identifier   []Resource     `json:"identifier,omitempty"`
	Active       *bool          `json:"active,omitempty"`
	Type         []Resource     `json:"type,omitempty"`
	Name         string         `json:"name,omitempty"`
	Alias        []string       `json:"alias,omitempty"`
	Telecom      []Resource     `json:"telecom,omitempty"`
	Address      []Resource     `json:"address,omitempty"`
	PartOf       *RefIdentifier `json:"partOf,omitempty"`
	Extra        map[string]any `json:"-"`
}

// MarshalJSON merges Extra into the top-level object so callers can stash
// unmodelled fields without losing them.
func (o OrganizationInput) MarshalJSON() ([]byte, error) {
	type plain OrganizationInput
	return core.MarshalWithExtra(plain(o), o.Extra)
}

// LocationInput is a FHIR R4 Location as accepted by SATUSEHAT (POST
// /Location).
type LocationInput struct {
	ResourceType         string         `json:"resourceType"`
	Identifier           []Resource     `json:"identifier,omitempty"`
	Status               string         `json:"status,omitempty"`
	Name                 string         `json:"name,omitempty"`
	Alias                []string       `json:"alias,omitempty"`
	Mode                 string         `json:"mode,omitempty"`
	Type                 []Resource     `json:"type,omitempty"`
	PhysicalType         *Resource      `json:"physicalType,omitempty"`
	ManagingOrganization *RefIdentifier `json:"managingOrganization,omitempty"`
	PartOf               *RefIdentifier `json:"partOf,omitempty"`
	Position             *Position      `json:"position,omitempty"`
	Extra                map[string]any `json:"-"`
}

// MarshalJSON merges Extra into the top-level object.
func (l LocationInput) MarshalJSON() ([]byte, error) {
	type plain LocationInput
	return core.MarshalWithExtra(plain(l), l.Extra)
}

// RefIdentifier pairs a `reference` ("Organization/<id>") with an
// `identifier` (the org-scoped sys-ids). Either is acceptable per the FHIR
// spec; SATUSEHAT examples usually show `reference`.
type RefIdentifier struct {
	Reference  string             `json:"reference,omitempty"`
	Identifier *coding.Identifier `json:"identifier,omitempty"`
}

// Position is the FHIR Location.position object (WGS84 lat/lon + altitude).
type Position struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
	Altitude  float64 `json:"altitude,omitempty"`
}

// CreateOrganization POSTs to /Organization.
func CreateOrganization(ctx context.Context, c *Client, body OrganizationInput) (any, error) {
	return c.Request(ctx, "POST", "/Organization", &RequestOptions{Body: body})
}

// CreateLocation POSTs to /Location.
func CreateLocation(ctx context.Context, c *Client, body LocationInput) (any, error) {
	return c.Request(ctx, "POST", "/Location", &RequestOptions{Body: body})
}

// GetOrganization reads a single Organization by id.
func GetOrganization(ctx context.Context, c *Client, id string) (any, error) {
	return c.Request(ctx, "GET", "/Organization/"+url.PathEscape(id), nil)
}

// GetLocation reads a single Location by id.
func GetLocation(ctx context.Context, c *Client, id string) (any, error) {
	return c.Request(ctx, "GET", "/Location/"+url.PathEscape(id), nil)
}
