package satusehat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/harizinside/go-satusehat/coding"
	"github.com/harizinside/go-satusehat/fhir"
	"github.com/harizinside/go-satusehat/rawatjalan"
)

// Input types with Extra used to recurse forever in MarshalJSON (fatal stack overflow).
func TestMarshalWithExtra_NoRecursion(t *testing.T) {
	cases := []struct {
		v    any
		want string
	}{
		{OrganizationInput{ResourceType: "Organization", Extra: map[string]any{"x": 1}}, `{"resourceType":"Organization","x":1}`},
		{LocationInput{ResourceType: "Location", Name: "Poli"}, `{"resourceType":"Location","name":"Poli"}`},
		{RmeLinkInput{PatientID: "P1", Extra: map[string]any{"x": 1}}, `"x":1`},
		{AlkesPageBody{Page: Ptr(2)}, `"page":2`},
		{fhir.SlotInput{Extra: map[string]any{"x": 1}}, `"x":1`},
	}
	for _, tc := range cases {
		b, err := json.Marshal(tc.v)
		if err != nil || !strings.Contains(string(b), tc.want) {
			t.Fatalf("%T: %s %v, want %s", tc.v, b, err, tc.want)
		}
	}
}

// Unset nested objects must not be sent as {} (choice[x] siblings), and an
// explicit false must survive.
func TestGeneratedBody_OmitsUnsetObjects(t *testing.T) {
	b, _ := json.Marshal(rawatjalan.ConditionKeluhanUtamaBody{OnsetDateTime: "2024-01-01"})
	if string(b) != `{"onsetDateTime":"2024-01-01"}` {
		t.Fatalf("got %s", b)
	}
	var body rawatjalan.ConditionKeluhanUtamaBody
	body.Subject.Reference = "Patient/1"
	b, _ = json.Marshal(body)
	if string(b) != `{"subject":{"reference":"Patient/1"}}` {
		t.Fatalf("got %s", b)
	}
	b, _ = json.Marshal(fhir.PatientCreateByNikBody{DeceasedBoolean: Ptr(false)})
	if string(b) != `{"deceasedBoolean":false}` {
		t.Fatalf("got %s", b)
	}
}

// KYC must default to Config.KycBaseURL and decrypt the armored text reply.
func TestKYC_GenerateURL_DefaultBase(t *testing.T) {
	ss, _ := GenerateKycKeyPair() // stands in for SATU SEHAT's published key
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/kyc/v1/generate-url" {
			t.Fatalf("path %s", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		plain, err := DecryptKycMessage(string(raw), ss.PrivateKey)
		if err != nil {
			t.Fatalf("server decrypt: %v", err)
		}
		var req struct {
			PublicKey string `json:"public_key"`
		}
		_ = json.Unmarshal([]byte(plain), &req)
		reply, _ := EncryptKycMessage(`{"token":"frame-1"}`, req.PublicKey)
		_, _ = w.Write([]byte(reply))
	})
	defer srv.Close()

	res, err := GenerateKycUrl(context.Background(), c, GenerateKycUrlParams{AgentName: "a", AgentNIK: "1", SatuSehatPublicKey: ss.PublicKey}, GenerateUrlOptions{})
	if err != nil {
		t.Fatalf("GenerateKycUrl: %v", err)
	}
	if data := res.(map[string]any)["data"].(map[string]any); data["token"] != "frame-1" {
		t.Fatalf("got %v", data)
	}
}

func TestTransportError_Unwraps(t *testing.T) {
	c, srv := newTestServer(t, func(http.ResponseWriter, *http.Request) {})
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Request(ctx, "GET", "/Patient", nil)
	var api *APIError
	if !errors.Is(err, context.Canceled) || !errors.As(err, &api) || api.StatusCode != 0 {
		t.Fatalf("got %v", err)
	}
}

// *bool false is sent, "" is skipped, ids are path-escaped.
func TestQueryAndPathEncoding(t *testing.T) {
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/mastersarana"):
			if got := r.URL.RawQuery; got != "status_aktif=false" {
				t.Fatalf("query %q", got)
			}
		case strings.Contains(r.URL.Path, "/Encounter/"):
			if r.URL.EscapedPath() != "/fhir-r4/v1/Encounter/a%2Fb" {
				t.Fatalf("path %s", r.URL.EscapedPath())
			}
		case r.URL.Path == "/fhir-r4/v1/Patient":
			if got := r.URL.RawQuery; got != "a=1" {
				t.Fatalf("query %q", got)
			}
		}
		_, _ = w.Write([]byte(`{}`))
	})
	defer srv.Close()
	ctx := context.Background()
	if _, err := GetMasterSarana(ctx, c, MasterSaranaParams{StatusAktif: Ptr(false)}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.FHIR.Encounter.EncounterById(ctx, "a/b"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Request(ctx, "GET", "/Patient", &RequestOptions{Query: map[string]any{"a": 1, "b": "", "c": (*string)(nil)}}); err != nil {
		t.Fatal(err)
	}
}

func TestRefIdentifierShape(t *testing.T) {
	b, _ := json.Marshal(RefIdentifier{Identifier: &coding.Identifier{System: "s", Value: "v"}})
	if string(b) != `{"identifier":{"system":"s","value":"v"}}` {
		t.Fatalf("got %s", b)
	}
}

func TestWebhookCallback_EchoesRaw(t *testing.T) {
	raw := `{"data":{"fhir":{}},"meta":{"method":"claimBundle","extra":1},"transaction_id":"t","timestamp":"x","unmodelled":true}`
	c, srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		got, _ := io.ReadAll(r.Body)
		if string(got) != raw || r.Header.Get("Authorization") != "" {
			t.Fatalf("got %s auth=%q", got, r.Header.Get("Authorization"))
		}
	})
	defer srv.Close()
	c.SetToken("tok")
	p := ParseWebhookPayloadAs([]byte(raw), WebhookClaimBundle)
	if p == nil {
		t.Fatal("parse failed")
	}
	if _, err := c.Webhook.SendCallback(context.Background(), srv.URL+"/cb", *p); err != nil {
		t.Fatal(err)
	}
}
