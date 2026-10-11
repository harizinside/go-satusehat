package satusehat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/harizinside/go-satusehat/farmasi"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c := New(Config{
		Environment:       Staging,
		BaseURL:           srv.URL + "/fhir-r4/v1",
		AuthBaseURL:       srv.URL + "/oauth2/v1",
		MasterDataBaseURL: srv.URL,
		KycBaseURL:        srv.URL + "/kyc/v1",
		RmeBaseURL:        srv.URL,
	})
	return c, srv
}

// Token — exercises both `expires_in` shapes (number and string) and
// verifies the bearer header is attached to subsequent calls.
func TestGetAccessToken_NumberExpiresIn(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/accesstoken") {
			if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
				t.Fatalf("expected form content-type, got %q", got)
			}
			if got := r.URL.Query().Get("grant_type"); got != "client_credentials" {
				t.Fatalf("expected grant_type=client_credentials, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "tok-123",
				"expires_in":   3600,
				"token_type":   "Bearer",
			})
			return
		}
		t.Fatalf("unexpected path: %s", r.URL.Path)
	})
	defer srv.Close()

	tok, err := GetAccessToken(context.Background(), c, CredentialsOverride{
		ClientID:     "id",
		ClientSecret: "secret",
	})
	if err != nil {
		t.Fatalf("GetAccessToken: %v", err)
	}
	if tok.AccessToken != "tok-123" {
		t.Fatalf("expected tok-123, got %q", tok.AccessToken)
	}
	if tok.ExpiresIn.String() != "1h0m0s" {
		t.Fatalf("expected 1h0m0s, got %v", tok.ExpiresIn)
	}
}

// Token — `expires_in` comes back as a string in some responses.
func TestGetAccessToken_StringExpiresIn(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"abc","expires_in":"7200"}`))
	})
	defer srv.Close()

	tok, err := GetAccessToken(context.Background(), c, CredentialsOverride{
		ClientID: "id", ClientSecret: "secret",
	})
	if err != nil {
		t.Fatalf("GetAccessToken: %v", err)
	}
	if tok.ExpiresIn.String() != "2h0m0s" {
		t.Fatalf("expected 2h0m0s, got %v", tok.ExpiresIn)
	}
}

// Content-Type is `application/json-patch+json` on PATCH (FHIR JSON Patch).
func TestPatchContentType(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json-patch+json" {
			t.Fatalf("expected json-patch content-type, got %q", ct)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer srv.Close()

	_, err := c.Request(context.Background(), "PATCH", "/Patient/123",
		&RequestOptions{Body: []PatchOp{{Op: "replace", Path: "/name/0/family", Value: "Smith"}}})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

// Query encoding skips zero/empty values (matches `appendQuery` in TS).
func TestQueryEncoding_ZeroSkipped(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Has("count") {
			t.Fatalf("zero count should not appear")
		}
		if q.Get("name") != "Alice" {
			t.Fatalf("expected name=Alice, got %q", q.Get("name"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"resourceType":"Bundle"}`))
	})
	defer srv.Close()

	_, err := c.Request(context.Background(), "GET", "/Patient", &RequestOptions{
		Query: PersonSearchParams{Name: "Alice", Count: 0},
	})
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
}

// Non-2xx response surfaces as `*APIError` with body snippet.
func TestAPIError_BodyIncluded(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"invalid"}]}`))
	})
	defer srv.Close()

	_, err := c.Request(context.Background(), "POST", "/Patient", &RequestOptions{Body: Resource{}})
	var api *APIError
	if !errors.As(err, &api) {
		t.Fatalf("expected APIError, got %T %v", err, err)
	}
	if api.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", api.StatusCode)
	}
	if !strings.Contains(api.Error(), "OperationOutcome") {
		t.Fatalf("expected body snippet in error, got %q", api.Error())
	}
}

// KYC envelope round-trip: a generated keypair + message must survive
// `Encrypt`/`Decrypt`.
func TestKYC_RoundTrip(t *testing.T) {
	kp, err := GenerateKycKeyPair()
	if err != nil {
		t.Fatalf("GenerateKycKeyPair: %v", err)
	}
	const msg = `{"hello":"world"}`
	encrypted, err := EncryptKycMessage(msg, kp.PublicKey)
	if err != nil {
		t.Fatalf("EncryptKycMessage: %v", err)
	}
	if !strings.HasPrefix(encrypted, "-----BEGIN ENCRYPTED MESSAGE-----") {
		t.Fatalf("missing envelope header: %q", encrypted)
	}
	decrypted, err := DecryptKycMessage(encrypted, kp.PrivateKey)
	if err != nil {
		t.Fatalf("DecryptKycMessage: %v", err)
	}
	if decrypted != msg {
		t.Fatalf("expected %q, got %q", msg, decrypted)
	}
}

// `BuildTransactionBundle` defaults URL to `resource.resourceType`.
func TestBuildTransactionBundle(t *testing.T) {
	b := BuildTransactionBundle([]BundleEntryInput{
		{Resource: Resource{"resourceType": "Encounter", "status": "in-progress"}},
		{Resource: Resource{"resourceType": "Condition"}, FullURL: "urn:uuid:abc"},
	})
	if b.ResourceType != "Bundle" || b.Type != "transaction" {
		t.Fatalf("unexpected bundle: %+v", b)
	}
	if len(b.Entry) != 2 {
		t.Fatalf("expected 2 entries")
	}
	if b.Entry[0].Request.Method != BundlePOST || b.Entry[0].Request.URL != "Encounter" {
		t.Fatalf("first entry: %+v", b.Entry[0])
	}
	if b.Entry[1].FullURL != "urn:uuid:abc" {
		t.Fatalf("second entry fullUrl: %q", b.Entry[1].FullURL)
	}
}

// `RequestMasterData` resolves paths to the Master Data host.
func TestRequestMasterData(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/masterdata/v1/") {
			t.Fatalf("expected masterdata path, got %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"status":200,"error":false,"message":"ok","data":[]}`))
	})
	defer srv.Close()
	c.SetToken("test-token")
	_, err := c.RequestMasterData(context.Background(), "GET", "/masterdata/v1/provinces", nil)
	if err != nil {
		t.Fatalf("RequestMasterData: %v", err)
	}
}

// PATCH on a generated service method (id + operations).
func TestGeneratedService_Patch(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/fhir-r4/v1/Encounter/") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"resourceType":"Encounter"}`))
	})
	defer srv.Close()

	c.SetToken("test-token")
	res, err := c.FHIR.Encounter.EncounterPatchArray(context.Background(), "X", []PatchOp{
		{Op: "add", Path: "/status", Value: "finished"},
	})
	if err != nil {
		t.Fatalf("EncounterPatchArray: %v", err)
	}
	if res == nil {
		t.Fatalf("expected non-nil result")
	}
}

// Search method with query params (round-trips TS _include key).
func TestGeneratedService_SearchDocumentReference(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("_include") != "MedicationRequest" {
			t.Fatalf("expected _include=MedicationRequest, got %q", q.Get("_include"))
		}
		_, _ = w.Write([]byte(`{"resourceType":"Bundle","type":"searchset"}`))
	})
	defer srv.Close()

	c.SetToken("test-token")
	if _, err := c.Farmasi.Peresepan.SearchDocumentReference(context.Background(), farmasi.PeresepanSearchDocumentReferenceParams{
		Identifier: "X",
		Include:    "MedicationRequest",
	}); err != nil {
		t.Fatalf("SearchDocumentReference: %v", err)
	}
}

// url.Values object passed as Query works too.
func TestQueryValues(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("a") != "1" {
			t.Fatalf("expected a=1, got %q", r.URL.Query().Get("a"))
		}
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()
	v := url.Values{}
	v.Set("a", "1")
	if _, err := c.Request(context.Background(), "GET", "/X", &RequestOptions{Query: v}); err != nil {
		t.Fatalf("Request: %v", err)
	}
}
