package satusehat

import "context"

// Namespaces is the bag of sub-clients embedded in `Client`, so every module
// is a field: c.Auth, c.KYC, c.FHIR.Encounter, c.RawatJalan.Anamnesis, ...
// The generated ones come from `scripts/generate.mjs`.
type Namespaces struct {
	// Standalone helpers (matching the TS SDK's top-level `auth`,
	// `organization`, `search`, ... namespaces).
	Auth         AuthNamespace
	Organization OrganizationNamespace
	Search       SearchNamespace
	MasterData   MasterDataNamespace
	KFA          KFANamespace
	KYC          KycNamespace
	RME          RMENamespace
	Bundle       BundleNamespace
	Webhook      WebhookNamespace

	// Generated: per-FHIR-resource and per-workflow sub-namespaces.
	*GeneratedNamespaces
}

// newNamespaces builds the namespace tree for a Client.
func newNamespaces(c *Client) Namespaces {
	return Namespaces{
		Auth:                AuthNamespace{c: c},
		Organization:        OrganizationNamespace{c: c},
		Search:              SearchNamespace{c: c},
		MasterData:          MasterDataNamespace{c: c},
		KFA:                 KFANamespace{c: c},
		KYC:                 KycNamespace{c: c},
		RME:                 RMENamespace{c: c},
		Bundle:              BundleNamespace{c: c},
		Webhook:             WebhookNamespace{c: c},
		GeneratedNamespaces: buildGeneratedNamespaces(c),
	}
}

// — hand-written namespace shells ——————————————————————————————————————

// AuthNamespace wraps `GetAccessToken`/`Authenticate`.
type AuthNamespace struct{ c *Client }

// GetAccessToken mints a fresh OAuth2 client-credentials token.
func (n AuthNamespace) GetAccessToken(ctx context.Context, creds CredentialsOverride) (*AccessToken, error) {
	return GetAccessToken(ctx, n.c, creds)
}

// Authenticate mints a token and installs it on the client.
func (n AuthNamespace) Authenticate(ctx context.Context, creds CredentialsOverride) (*AccessToken, error) {
	return Authenticate(ctx, n.c, creds)
}

// OrganizationNamespace wraps `Organization`/`Location` CRUD.
type OrganizationNamespace struct{ c *Client }

// Create POSTs to /Organization.
func (n OrganizationNamespace) Create(ctx context.Context, body OrganizationInput) (any, error) {
	return CreateOrganization(ctx, n.c, body)
}

// CreateLocation POSTs to /Location.
func (n OrganizationNamespace) CreateLocation(ctx context.Context, body LocationInput) (any, error) {
	return CreateLocation(ctx, n.c, body)
}

// Get reads a single Organization by id.
func (n OrganizationNamespace) Get(ctx context.Context, id string) (any, error) {
	return GetOrganization(ctx, n.c, id)
}

// GetLocation reads a single Location by id.
func (n OrganizationNamespace) GetLocation(ctx context.Context, id string) (any, error) {
	return GetLocation(ctx, n.c, id)
}

// SearchNamespace wraps `Patient`/`Practitioner` lookups.
type SearchNamespace struct{ c *Client }

// Patient GETs `/Patient?identifier=...`.
func (n SearchNamespace) Patient(ctx context.Context, p PersonSearchParams) (any, error) {
	return SearchPatient(ctx, n.c, p)
}

// PatientByID GETs `/Patient/{id}`.
func (n SearchNamespace) PatientByID(ctx context.Context, id string) (any, error) {
	return GetPatientById(ctx, n.c, id)
}

// Practitioner GETs `/Practitioner?identifier=...`.
func (n SearchNamespace) Practitioner(ctx context.Context, p PersonSearchParams) (any, error) {
	return SearchPractitioner(ctx, n.c, p)
}

// PractitionerByID GETs `/Practitioner/{id}`.
func (n SearchNamespace) PractitionerByID(ctx context.Context, id string) (any, error) {
	return GetPractitionerById(ctx, n.c, id)
}

// MasterDataNamespace groups the wilayah + sarana endpoints.
type MasterDataNamespace struct{ c *Client }

// Provinces GETs `/masterdata/v1/provinces`.
func (n MasterDataNamespace) Provinces(ctx context.Context, codes string) (*RegionListResult, error) {
	return GetProvinces(ctx, n.c, struct{ Codes string }{Codes: codes})
}

// Cities GETs `/masterdata/v1/cities`.
func (n MasterDataNamespace) Cities(ctx context.Context, provinceCodes, codes string) (*RegionListResult, error) {
	return GetCities(ctx, n.c, struct{ ProvinceCodes, Codes string }{ProvinceCodes: provinceCodes, Codes: codes})
}

// Districts GETs `/masterdata/v1/districts`.
func (n MasterDataNamespace) Districts(ctx context.Context, cityCodes, codes string) (*RegionListResult, error) {
	return GetDistricts(ctx, n.c, struct{ CityCodes, Codes string }{CityCodes: cityCodes, Codes: codes})
}

// SubDistricts GETs `/masterdata/v1/sub-districts`.
func (n MasterDataNamespace) SubDistricts(ctx context.Context, districtCodes, codes string) (*RegionListResult, error) {
	return GetSubDistricts(ctx, n.c, struct{ DistrictCodes, Codes string }{DistrictCodes: districtCodes, Codes: codes})
}

// ProvincesV2 GETs `/masterdata/v2/provinces` (cursor paginated).
func (n MasterDataNamespace) ProvincesV2(ctx context.Context, p RegionPageParams) (*RegionListResult, error) {
	return GetProvincesV2(ctx, n.c, p)
}

// CitiesV2 GETs `/masterdata/v2/cities`.
func (n MasterDataNamespace) CitiesV2(ctx context.Context, p RegionPageParams) (*RegionListResult, error) {
	return GetCitiesV2(ctx, n.c, p)
}

// DistrictsV2 GETs `/masterdata/v2/districts`.
func (n MasterDataNamespace) DistrictsV2(ctx context.Context, p RegionPageParams) (*RegionListResult, error) {
	return GetDistrictsV2(ctx, n.c, p)
}

// SubDistrictsV2 GETs `/masterdata/v2/sub-districts`.
func (n MasterDataNamespace) SubDistrictsV2(ctx context.Context, p RegionPageParams) (*RegionListResult, error) {
	return GetSubDistrictsV2(ctx, n.c, p)
}

// Sarana GETs `/masterdata/v1/mastersaranaindex/mastersarana`.
func (n MasterDataNamespace) Sarana(ctx context.Context, p MasterSaranaParams) (*MasterSaranaResult, error) {
	return GetMasterSarana(ctx, n.c, p)
}

// KFANamespace wraps the price / products / alkes helpers.
type KFANamespace struct{ c *Client }

// Price GETs `/kfa/farmalkes-price-jkn`.
func (n KFANamespace) Price(ctx context.Context, p PriceJknParams) (any, error) {
	return GetPriceJkn(ctx, n.c, p)
}

// Products is DEPRECATED — see the per-method docs.
//
// Deprecated: returns 404 against the live server.
func (n KFANamespace) Atc(ctx context.Context, p KfaListParams) (*KfaProductListResult, error) {
	return GetAtcMetadata(ctx, n.c, p)
}

// ProductsByATC is DEPRECATED.
//
// Deprecated: returns 404 against the live server.
func (n KFANamespace) ProductsByATC(ctx context.Context, p KfaProductByAtcParams) (*KfaProductListResult, error) {
	return GetProductsByAtc(ctx, n.c, p)
}

// Tags is DEPRECATED.
//
// Deprecated: returns 404 against the live server.
func (n KFANamespace) Tags(ctx context.Context, p KfaListParams) (*KfaProductListResult, error) {
	return GetTagMetadata(ctx, n.c, p)
}

// ProductsByTag is DEPRECATED.
//
// Deprecated: returns 404 against the live server.
func (n KFANamespace) ProductsByTag(ctx context.Context, p KfaProductByTagParams) (*KfaProductListResult, error) {
	return GetProductsByTag(ctx, n.c, p)
}

// ProductDetail GETs `/kfa-v2/products?identifier=kfa&code=...`.
func (n KFANamespace) ProductDetail(ctx context.Context, p KfaProductDetailParams) (*KfaProductListResult, error) {
	return GetProductDetailV2(ctx, n.c, p)
}

// AllProducts GETs `/kfa-v2/products/all`.
func (n KFANamespace) AllProducts(ctx context.Context, p KfaProductAllParams) (*KfaProductListResult, error) {
	return GetAllProductsV2(ctx, n.c, p)
}

// AlkesTemplates POSTs to `/kfa-v3/alkes/template`.
func (n KFANamespace) AlkesTemplates(ctx context.Context, body AlkesPageBody, options AlkesOptions) (any, error) {
	return GetAlkesTemplates(ctx, n.c, body, options)
}

// AlkesProducts POSTs to `/kfa-v3/alkes/products`.
func (n KFANamespace) AlkesProducts(ctx context.Context, body AlkesPageBody, options AlkesOptions) (any, error) {
	return GetAlkesProducts(ctx, n.c, body, options)
}

// KycNamespace wraps the KYC helpers.
type KycNamespace struct{ c *Client }

// GenerateKeyPair returns a fresh 2048-bit RSA keypair.
func (n KycNamespace) GenerateKeyPair() (KycKeyPair, error) {
	return GenerateKycKeyPair()
}

// EncryptMessage wraps a message with the recipient's RSA public key.
func (n KycNamespace) EncryptMessage(message, recipientPublicKeyPEM string) (string, error) {
	return EncryptKycMessage(message, recipientPublicKeyPEM)
}

// DecryptMessage reverses EncryptKycMessage with the matching RSA private key.
func (n KycNamespace) DecryptMessage(message, privateKeyPEM string) (string, error) {
	return DecryptKycMessage(message, privateKeyPEM)
}

// GenerateURL POSTs to `{kycBaseUrl}/generate-url`.
func (n KycNamespace) GenerateURL(ctx context.Context, p GenerateKycUrlParams, o GenerateUrlOptions) (any, error) {
	return GenerateKycUrl(ctx, n.c, p, o)
}

// ChallengeCode POSTs to `{kycBaseUrl}/challenge-code`.
func (n KycNamespace) ChallengeCode(ctx context.Context, p GenerateChallengeCodeParams, o ChallengeCodeOptions) (any, error) {
	return GenerateChallengeCode(ctx, n.c, p, o)
}

// RMENamespace wraps the RME health-link endpoints.
type RMENamespace struct{ c *Client }

// CreateHealthLink POSTs to `{baseUrl}/ssrme/v2/ntl/chl`.
func (n RMENamespace) CreateHealthLink(ctx context.Context, body RmeLinkInput, options RmeOptions) (any, error) {
	return CreateHealthLink(ctx, n.c, body, options)
}

// ShowHealthLink POSTs to `{baseUrl}/ssrme/v2/ntl/shl`.
func (n RMENamespace) ShowHealthLink(ctx context.Context, body RmeLinkInput, options RmeOptions) (any, error) {
	return ShowHealthLink(ctx, n.c, body, options)
}

// BundleNamespace wraps the Bundle builder helpers.
type BundleNamespace struct{ c *Client }

// Build assembles a FHIR transaction Bundle from typed entries.
func (n BundleNamespace) Build(entries []BundleEntryInput) FhirTransactionBundle {
	return BuildTransactionBundle(entries)
}

// Submit POSTs the Bundle to `/`.
func (n BundleNamespace) Submit(ctx context.Context, b FhirTransactionBundle) (any, error) {
	return SubmitBundle(ctx, n.c, b)
}

// SubmitVisit is the build + submit wrapper.
func (n BundleNamespace) SubmitVisit(ctx context.Context, entries []BundleEntryInput) (any, error) {
	return SubmitVisitBundle(ctx, n.c, entries)
}

// WebhookNamespace wraps the webhook payload helpers.
type WebhookNamespace struct{ c *Client }

// Parse reads a raw webhook body.
func (WebhookNamespace) Parse(raw []byte) *SatuSehatWebhookPayload {
	return ParseWebhookPayload(raw)
}

// ParseAs additionally verifies meta.method.
func (WebhookNamespace) ParseAs(raw []byte, method WebhookMethod) *SatuSehatWebhookPayload {
	return ParseWebhookPayloadAs(raw, method)
}

// SendCallback acknowledges the webhook to the server's callback URL.
func (n WebhookNamespace) SendCallback(ctx context.Context, callbackURL string, payload SatuSehatWebhookPayload) (any, error) {
	return SendWebhookCallback(ctx, n.c, callbackURL, payload)
}
