package satusehat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/harizinside/go-satusehat/internal/core"
)

// Environment picks between staging and production endpoints.
type Environment string

const (
	Staging    Environment = "staging"
	Production Environment = "production"
)

// EnvironmentDefaults holds the per-environment base URLs. Every SATU SEHAT
// family (OAuth2, FHIR, KYC, RME, Master Data + KFA) shares the same staging
// host `api-satusehat-stg.dto.kemkes.go.id`. The Master Data + KFA host is
// configurable separately from `baseUrl` because the *paths* differ
// (`/masterdata/v1/...`, `/kfa/...` vs the `/fhir-r4/v1/...` prefix baked
// into `baseUrl`), not because the hosts do.
type EnvironmentDefaults struct {
	BaseURL           string
	AuthBaseURL       string
	KycBaseURL        string
	MasterDataBaseURL string
	RmeBaseURL        string
}

var environmentDefaults = map[Environment]EnvironmentDefaults{
	Production: {
		BaseURL:           "https://api-satusehat.kemkes.go.id/fhir-r4/v1",
		AuthBaseURL:       "https://api-satusehat.kemkes.go.id/oauth2/v1",
		KycBaseURL:        "https://api-satusehat.kemkes.go.id/kyc/v1",
		MasterDataBaseURL: "https://api-satusehat.kemkes.go.id",
		RmeBaseURL:        "https://api-satusehat.kemkes.go.id",
	},
	Staging: {
		BaseURL:           "https://api-satusehat-stg.dto.kemkes.go.id/fhir-r4/v1",
		AuthBaseURL:       "https://api-satusehat-stg.dto.kemkes.go.id/oauth2/v1",
		KycBaseURL:        "https://api-satusehat-stg.dto.kemkes.go.id/kyc/v1",
		MasterDataBaseURL: "https://api-satusehat-stg.dto.kemkes.go.id",
		RmeBaseURL:        "https://api-satusehat-stg.dto.kemkes.go.id",
	},
}

// Config is the user-facing configuration for `New`. Only Environment and
// ClientID/Secret are required to *mint* one; token caching itself is the
// caller's responsibility — see package docs.
type Config struct {
	Environment       Environment
	ClientID          string
	ClientSecret      string
	Token             string
	BaseURL           string // override FHIR base (host + `/fhir-r4/v1`)
	AuthBaseURL       string // override OAuth2 host
	KycBaseURL        string // override KYC host
	MasterDataBaseURL string // override Master Data + KFA host
	RmeBaseURL        string // override RME host (bare host, no `/fhir-r4/v1` prefix)
	HTTPClient        *http.Client
}

// Client is the SATU SEHAT entry point. Every request flows through `Do` or
// `Request`; namespace sub-clients (`c.FHIR`, `c.RawatJalan`, ...) live alongside
// it (see `namespaces.go`).
type Client struct {
	// Namespaces exposes every module as a field: c.Auth, c.FHIR.Encounter,
	// c.RawatJalan.Anamnesis, ...
	Namespaces

	config  Config
	tokenMu sync.RWMutex
	token   string
}

// New builds a Client. If `Environment` is empty, defaults to Production.
func New(cfg Config) *Client {
	env := cfg.Environment
	if env == "" {
		env = Production
	}
	defaults := environmentDefaults[env]
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaults.BaseURL
	}
	if cfg.AuthBaseURL == "" {
		cfg.AuthBaseURL = defaults.AuthBaseURL
	}
	if cfg.KycBaseURL == "" {
		cfg.KycBaseURL = defaults.KycBaseURL
	}
	if cfg.MasterDataBaseURL == "" {
		cfg.MasterDataBaseURL = defaults.MasterDataBaseURL
	}
	if cfg.RmeBaseURL == "" {
		cfg.RmeBaseURL = defaults.RmeBaseURL
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	c := &Client{config: cfg, token: cfg.Token}
	c.Namespaces = newNamespaces(c)
	return c
}

// SetToken replaces the access token used for subsequent requests. Safe to
// call from multiple goroutines; in-flight requests keep using whatever
// token they already snapshotted.
func (c *Client) SetToken(token string) {
	c.tokenMu.Lock()
	c.token = token
	c.tokenMu.Unlock()
}

// ClearToken drops the access token.
func (c *Client) ClearToken() {
	c.SetToken("")
}

// Config returns a copy of the resolved configuration (token excluded).
func (c *Client) Config() Config {
	return Config{
		Environment:       c.config.Environment,
		ClientID:          c.config.ClientID,
		ClientSecret:      c.config.ClientSecret,
		BaseURL:           c.config.BaseURL,
		AuthBaseURL:       c.config.AuthBaseURL,
		KycBaseURL:        c.config.KycBaseURL,
		MasterDataBaseURL: c.config.MasterDataBaseURL,
		RmeBaseURL:        c.config.RmeBaseURL,
		HTTPClient:        c.config.HTTPClient,
	}
}

// Request is the low-level helper. `pathOrURL` is either relative to
// `BaseURL` (the normal case — `BaseURL` already carries the `/fhir-r4/v1`
// prefix) or an absolute URL (used for the OAuth2, KYC, RME, Master Data,
// webhook-callback hosts). The returned `any` is whatever the server JSON
// decoded to: a `Resource` for most endpoints, `*Bundle` for searches, or
// nil for empty 204 responses.
func (c *Client) Request(ctx context.Context, method, pathOrURL string, opts *RequestOptions) (any, error) {
	url := pathOrURL
	if !strings.HasPrefix(strings.ToLower(url), "http://") && !strings.HasPrefix(strings.ToLower(url), "https://") {
		base := strings.TrimRight(c.config.BaseURL, "/")
		if !strings.HasPrefix(pathOrURL, "/") {
			pathOrURL = "/" + pathOrURL
		}
		url = base + pathOrURL
	}
	return c.requestURL(ctx, method, url, opts)
}

// RequestMasterData resolves the path against the Master Data + KFA host
// instead of the FHIR one. Identical to `Request` otherwise.
func (c *Client) RequestMasterData(ctx context.Context, method, path string, opts *RequestOptions) (any, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	url := strings.TrimRight(c.config.MasterDataBaseURL, "/") + path
	return c.requestURL(ctx, method, url, opts)
}

// RequestURL is `Request` but always treats the target as an absolute URL.
// Used by the KYC, RME, webhook-callback, and OAuth2 flows.
func (c *Client) RequestURL(ctx context.Context, method, fullURL string, opts *RequestOptions) (any, error) {
	return c.requestURL(ctx, method, fullURL, opts)
}

// RequestOptions mirrors the TS `RequestOptions` shape: `query`, `body`,
// `headers`, `auth`. `Auth` defaults to true; pass `Auth: Ptr(false)` for the
// OAuth2 token endpoint (which must NOT carry a Bearer header).
type RequestOptions = core.RequestOptions

// requestURL is the shared low-level path used by Request/RequestMasterData/
// RequestURL and by the generated methods.
func (c *Client) requestURL(ctx context.Context, method, fullURL string, opts *RequestOptions) (any, error) {
	if opts == nil {
		opts = &RequestOptions{}
	}
	fullURL = appendQuery(fullURL, opts.Query)

	headers := make(map[string]string, len(opts.Headers)+1)
	for k, v := range opts.Headers {
		headers[k] = v
	}

	auth := true
	if opts.Auth != nil {
		auth = *opts.Auth
	}
	if auth {
		c.tokenMu.RLock()
		tok := c.token
		c.tokenMu.RUnlock()
		if tok != "" {
			if !hasHeader(headers, "Authorization") {
				headers["Authorization"] = "Bearer " + tok
			}
		}
	}

	var bodyBytes []byte
	if opts.Body != nil {
		var bodyStr string
		switch v := opts.Body.(type) {
		case string:
			bodyStr = v
		case []byte:
			bodyStr = string(v)
		case io.Reader:
			data, err := io.ReadAll(v)
			if err != nil {
				return nil, fmt.Errorf("satusehat: read body: %w", err)
			}
			bodyStr = string(data)
		default:
			data, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("satusehat: marshal body: %w", err)
			}
			bodyStr = string(data)
		}
		bodyBytes = []byte(bodyStr)
		if !hasHeader(headers, "Content-Type") {
			// FHIR PATCH is JSON Patch (RFC 6902) — every generated *Patch*
			// function sends a FhirPatchOperation[] body, which needs this
			// content type, not plain application/json.
			if strings.EqualFold(method, "PATCH") {
				headers["Content-Type"] = "application/json-patch+json"
			} else {
				headers["Content-Type"] = "application/json"
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("satusehat: build request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, &APIError{Method: method, URL: fullURL, Headers: map[string]string{}, Err: err}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &APIError{StatusCode: resp.StatusCode, Method: method, URL: fullURL, Body: body, Headers: flattenHeaders(resp.Header), Err: err}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Method:     method,
			URL:        fullURL,
			Body:       body,
			Headers:    flattenHeaders(resp.Header),
		}
	}

	if len(bytes.TrimSpace(body)) == 0 {
		return nil, nil
	}
	return decodeJSON(body)
}

func hasHeader(h map[string]string, key string) bool {
	for k := range h {
		if strings.EqualFold(k, key) {
			return true
		}
	}
	return false
}

func flattenHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, vs := range h {
		out[strings.ToLower(k)] = strings.Join(vs, ", ")
	}
	return out
}

// decodeJSON returns a `Resource` when the top-level value is an object, a
// `*Bundle` when it looks like one, otherwise the raw value.
func decodeJSON(body []byte) (any, error) {
	var anyVal any
	if err := json.Unmarshal(body, &anyVal); err != nil {
		// Non-JSON: return the raw bytes so the caller can string() them.
		return json.RawMessage(body), nil
	}
	return anyVal, nil
}

// appendQuery mirrors `appendQuery` in the TS SDK. Maps skip nil, nil
// pointers and "" (like TS skipping undefined/null/""); `false` and 0 are
// sent. Structs (`url:"name"` tags) also skip non-pointer zero values, so use
// a pointer field (e.g. `*bool`) when false/0 must be sent.
func appendQuery(base string, q any) string {
	if q == nil {
		return base
	}
	values := url.Values{}
	switch v := q.(type) {
	case url.Values:
		values = v
	case map[string][]string:
		values = url.Values(v)
	case map[string]string:
		for k, val := range v {
			if val != "" {
				values.Set(k, val)
			}
		}
	case map[string]any:
		for k, val := range v {
			if s, ok := queryValue(reflect.ValueOf(val)); ok {
				values.Set(k, s)
			}
		}
	default:
		rv := reflect.ValueOf(q)
		for rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Struct {
			t := rv.Type()
			for i := range t.NumField() {
				f := t.Field(i)
				tag, _ := f.Tag.Lookup("url")
				if tag == "" || tag == "-" {
					continue
				}
				name, _, _ := strings.Cut(tag, ",")
				if name == "" {
					continue
				}
				fv := rv.Field(i)
				if fv.Kind() != reflect.Ptr && fv.IsZero() {
					continue
				}
				if s, ok := queryValue(fv); ok {
					values.Set(name, s)
				}
			}
		}
	}
	encoded := values.Encode()
	if encoded == "" {
		return base
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + encoded
}

func stringifyQueryValue(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case int:
		return strconv.FormatInt(int64(x), 10)
	case int8:
		return strconv.FormatInt(int64(x), 10)
	case int16:
		return strconv.FormatInt(int64(x), 10)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case uint:
		return strconv.FormatUint(uint64(x), 10)
	case uint8:
		return strconv.FormatUint(uint64(x), 10)
	case uint16:
		return strconv.FormatUint(uint64(x), 10)
	case uint32:
		return strconv.FormatUint(uint64(x), 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	case float32:
		return strconv.FormatFloat(float64(x), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case fmt.Stringer:
		return x.String()
	case json.RawMessage:
		return string(x)
	default:
		return fmt.Sprint(v)
	}
}

// queryValue renders one query value, dereferencing pointers. It reports
// false for nil, nil pointers and empty strings.
func queryValue(v reflect.Value) (string, bool) {
	for v.IsValid() && (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return "", false
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return "", false
	}
	s := stringifyQueryValue(v.Interface())
	return s, s != ""
}
