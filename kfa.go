package satusehat

import (
	"context"
)

// PriceJknParams is the input for `GetPriceJkn`.
type PriceJknParams struct {
	Page        int    `url:"page,omitempty"`
	Limit       int    `url:"limit,omitempty"`
	KFACode     string `url:"kfa_code,omitempty"`
	RegionCode  string `url:"region_code,omitempty"`
	DocumentRef string `url:"document_ref,omitempty"`
}

// GetPriceJkn GETs `/kfa/farmalkes-price-jkn` — JKN price list for farmasi &
// alkes. No example response ships in the collection; the return is `any`.
func GetPriceJkn(ctx context.Context, client *Client, params PriceJknParams) (any, error) {
	return client.RequestMasterData(ctx, "GET", "/kfa/farmalkes-price-jkn", &RequestOptions{Query: params})
}

// KfaListParams is the input for the paginated KFA product / ATC / tag
// endpoints.
type KfaListParams struct {
	Page int `url:"page,omitempty"`
	Size int `url:"size,omitempty"`
}

// KfaProductListResult is the (loosely-typed) shape shared by the KFA list
// endpoints. Per the original TS source, downstream fields are free-form.
type KfaProductListResult struct {
	Pagination *Resource  `json:"pagination,omitempty"`
	ItemData   []Resource `json:"item_data,omitempty"`
}

// GetAtcMetadata is DEPRECATED: returns 404 against the live server and is no
// longer in the current official KFA docs.
//
// Deprecated: Confirmed dead: returns a real `404 Not Found` against the live
// server, and the current official KFA docs no longer list this endpoint at
// all — ATC data now comes back as fields inside
// `GetProductDetailV2`/`GetAllProductsV2` responses instead.
func GetAtcMetadata(ctx context.Context, client *Client, params KfaListParams) (*KfaProductListResult, error) {
	return decodeKfaProductList(client.RequestMasterData(ctx, "GET", "/kfa/atc", &RequestOptions{Query: params}))
}

// KfaProductByAtcParams extends KfaListParams with the ATC filter.
type KfaProductByAtcParams struct {
	Page    int    `url:"page,omitempty"`
	Size    int    `url:"size,omitempty"`
	ATCCode string `url:"atc_code,omitempty"`
	Level   int    `url:"level,omitempty"`
}

// GetProductsByAtc is DEPRECATED, same reason as `GetAtcMetadata`.
//
// Deprecated: Use `GetAllProductsV2` instead.
func GetProductsByAtc(ctx context.Context, client *Client, params KfaProductByAtcParams) (*KfaProductListResult, error) {
	return decodeKfaProductList(client.RequestMasterData(ctx, "GET", "/kfa/products/atc", &RequestOptions{Query: params}))
}

// GetTagMetadata is DEPRECATED, same reason as `GetAtcMetadata`.
//
// Deprecated: Use `GetAllProductsV2` instead.
func GetTagMetadata(ctx context.Context, client *Client, params KfaListParams) (*KfaProductListResult, error) {
	return decodeKfaProductList(client.RequestMasterData(ctx, "GET", "/kfa/tags", &RequestOptions{Query: params}))
}

// KfaProductByTagParams extends KfaListParams with the tag filter.
type KfaProductByTagParams struct {
	Page    int    `url:"page,omitempty"`
	Size    int    `url:"size,omitempty"`
	TagCode string `url:"tag_code,omitempty"`
	Level   int    `url:"level,omitempty"`
}

// GetProductsByTag is DEPRECATED, same reason as `GetAtcMetadata`.
//
// Deprecated: Use `GetAllProductsV2` instead.
func GetProductsByTag(ctx context.Context, client *Client, params KfaProductByTagParams) (*KfaProductListResult, error) {
	return decodeKfaProductList(client.RequestMasterData(ctx, "GET", "/kfa/products/tag", &RequestOptions{Query: params}))
}

// KfaProductDetailParams is the input for `GetProductDetailV2`.
type KfaProductDetailParams struct {
	Identifier string `url:"identifier,omitempty"`
	Code       string `url:"code,omitempty"`
}

// GetProductDetailV2 GETs `/kfa-v2/products?identifier=kfa&code=...`.
func GetProductDetailV2(ctx context.Context, client *Client, params KfaProductDetailParams) (*KfaProductListResult, error) {
	return decodeKfaProductList(client.RequestMasterData(ctx, "GET", "/kfa-v2/products", &RequestOptions{Query: params}))
}

// KfaProductAllParams is the input for `GetAllProductsV2`.
type KfaProductAllParams struct {
	Page        int    `url:"page,omitempty"`
	Size        int    `url:"size,omitempty"`
	ProductType string `url:"product_type,omitempty"`
}

// GetAllProductsV2 GETs `/kfa-v2/products/all` — paginated full KFA catalog.
func GetAllProductsV2(ctx context.Context, client *Client, params KfaProductAllParams) (*KfaProductListResult, error) {
	return decodeKfaProductList(client.RequestMasterData(ctx, "GET", "/kfa-v2/products/all", &RequestOptions{Query: params}))
}

// AlkesPageBody is the request body for the V3 Alkes endpoints. Both fields
// are optional; the postman example defaults to `{page:1, size:10}`.
type AlkesPageBody struct {
	Page  *int           `json:"page,omitempty"`
	Size  *int           `json:"size,omitempty"`
	Extra map[string]any `json:"-"`
}

// MarshalJSON merges Extra into the top-level object.
func (a AlkesPageBody) MarshalJSON() ([]byte, error) {
	type plain AlkesPageBody
	return marshalWithExtra(plain(a), a.Extra)
}

// AlkesOptions lets callers override the Authorization header and the base URL
// for the V3 Alkes endpoints. The original collection hardcodes a Basic-auth
// credential and points at an internal-only host; this SDK sends the
// configured OAuth2 Bearer token by default and lets you override both.
type AlkesOptions struct {
	Authorization string
	BaseURL       string
}

// GetAlkesTemplates POSTs to `/kfa-v3/alkes/template`.
func GetAlkesTemplates(ctx context.Context, client *Client, body AlkesPageBody, options AlkesOptions) (any, error) {
	return alkesPost(ctx, client, "/kfa-v3/alkes/template", body, options)
}

// GetAlkesProducts POSTs to `/kfa-v3/alkes/products`.
func GetAlkesProducts(ctx context.Context, client *Client, body AlkesPageBody, options AlkesOptions) (any, error) {
	return alkesPost(ctx, client, "/kfa-v3/alkes/products", body, options)
}

func alkesPost(ctx context.Context, client *Client, path string, body AlkesPageBody, options AlkesOptions) (any, error) {
	headers := map[string]string{}
	if options.Authorization != "" {
		headers["Authorization"] = options.Authorization
	}
	base := options.BaseURL
	if base == "" {
		base = client.config.MasterDataBaseURL
	}
	if len(path) == 0 || path[0] != '/' {
		path = "/" + path
	}
	return client.RequestURL(ctx, "POST", trimTrailingSlash(base)+path, &RequestOptions{Body: body, Headers: headers})
}

func decodeKfaProductList(raw any, err error) (*KfaProductListResult, error) {
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return &KfaProductListResult{}, nil
	}
	return decodeToType[KfaProductListResult](raw)
}
