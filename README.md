# go-satusehat

> Go port of [@harizinside/satusehat](https://github.com/harizinside/satusehat) — every SATU SEHAT (Kemkes) Platform endpoint, behind one zero-dependency client.

```go
import "github.com/harizinside/go-satusehat"

c := satusehat.New(satusehat.Config{
    Environment: satusehat.Staging,
    ClientID:    os.Getenv("SATUSEHAT_CLIENT_ID"),
    ClientSecret: os.Getenv("SATUSEHAT_CLIENT_SECRET"),
})
tok, _ := c.Auth.GetAccessToken(ctx, satusehat.CredentialsOverride{}) // caller stores it
c.SetToken(tok.AccessToken)

res, _ := c.FHIR.Encounter.EncounterPatchArray(ctx, "abc-123", []satusehat.PatchOp{
    {Op: "replace", Path: "/status", Value: "finished"},
})
```

The port preserves the design intent of the Node SDK:

- **Zero dependency.** Pure Go stdlib.
- **Token caching is the caller's job.** `GetAccessToken` mints a fresh OAuth2 token, and you reattach via `Config.Token` / `SetToken` on every client build. The SDK never caches or auto-refreshes.
- **One client, namespaced sub-clients.** `c.FHIR`, `c.RawatJalan`, `c.Gigi`, `c.Farmasi`, `c.KlaimSwasta`, `c.KlaimBpjs`.
- **One package per workflow group.** Generated code lives in `fhir/`, `rawatjalan/`, `gigi/`, `farmasi/`, `klaimswasta/`, `klaimbpjs/`; each file mirrors a `src/...` file from the Node SDK (see `scripts/generate.mjs`).

## Installation

```bash
go get github.com/harizinside/go-satusehat   # Go 1.24+
```

### Request bodies

Generated body structs never send unset fields: nested objects use
`omitzero` (fill them by assignment, `body.Subject.Reference = "Patient/1"`),
and bools/numbers are pointers so an explicit `false`/`0` still goes out:

```go
import "github.com/harizinside/go-satusehat/fhir"

body := fhir.PatientCreateByNikBody{DeceasedBoolean: satusehat.Ptr(false)}
```

### Errors

Non-2xx responses and transport failures return `*satusehat.APIError`
(`errors.As`); transport causes unwrap, so `errors.Is(err, context.Canceled)` works.

## API surface

`c.FHIR` is a `fhir.Namespace` with one service per FHIR resource (`fhir.EncounterService`, `fhir.PatientService`, …); request bodies and params live in the same package:

```go
c.FHIR.Encounter.EncounterPatchArray(ctx, id, []satusehat.PatchOp{...})
c.FHIR.Patient.PatientCreateByNik(ctx, body)
c.FHIR.Coverage.CoverageKepesertaanBpjsK(ctx, body)
c.FHIR.Encounter.EncounterById(ctx, id)
c.FHIR.Encounter.EncounterBySubject(ctx, fhir.EncounterBySubjectParams{Subject: "Patient/123"})
```

Workflow namespaces, each its own package (`rawatjalan`, `gigi`, `farmasi`, `klaimswasta`, `klaimbpjs`):

```go
c.RawatJalan.Anamnesis.ConditionKeluhanUtama(ctx, body)
c.Gigi.Anamnesis.ConditionKeluhanUtama(ctx, body)
c.Farmasi.Peresepan.CreateMedicationRequest(ctx, body)
c.KlaimSwasta.PrimaryPayor.ClaimDataKlaimAsuransiSwasta(ctx, body)
c.KlaimBpjs.ContohKlaim.EncounterKunjunganBaru(ctx, body)
```

Hand-written helpers (one-call wrappers around Node SDK equivalents):

```go
c.Auth.GetAccessToken(ctx, satusehat.CredentialsOverride{})   // → *AccessToken
c.Auth.Authenticate(ctx, satusehat.CredentialsOverride{})    // mints + SetToken
c.Organization.Create(ctx, body)                              // POST /Organization
c.Search.Patient(ctx, PersonSearchParams{Identifier: NIKIdentifier(nik)})
c.MasterData.Provinces(ctx, "")                               // /masterdata/v1/provinces
c.MasterData.ProvincesV2(ctx, RegionPageParams{...})         // /masterdata/v2/provinces (paginated)
c.KFA.AllProducts(ctx, KfaProductAllParams{Page: 1, Size: 50})
c.KYC.GenerateURL(ctx, GenerateKycUrlParams{...}, GenerateUrlOptions{})
c.KYC.ChallengeCode(ctx, GenerateChallengeCodeParams{...}, ChallengeCodeOptions{...})
c.RME.CreateHealthLink(ctx, body, RmeOptions{})
c.Bundle.SubmitVisit(ctx, []BundleEntryInput{...})
c.Webhook.Parse(rawJSON)                                      // → *SatuSehatWebhookPayload
```

## ICD-10 / ICD-9-CM

Heavy data tables live in their own sub-packages so programs that don't need them don't pay the import cost.

```go
import "github.com/harizinside/go-satusehat/icd10"

entry, ok := icd10.Lookup("A00")        // → coding.Coding
hits   := icd10.Search("cholera", 5)    // case-insensitive substring
n      := icd10.Count                   // 18,542 codes
```

Same for `github.com/harizinside/go-satusehat/icd9cm`.

## Terminology constants

The `coding` sub-package is a 1:1 mirror of the Node SDK's `fhir/coding/*` exports. 90+ prebuilt `Coding` constants (EncounterClass, MedicationRequestCategory, KemkesMedicationForm, …) plus `Identifier` helpers like `coding.NIK(...)`.

```go
import "github.com/harizinside/go-satusehat/coding"

enc := satusehat.Resource{
    "resourceType": "Encounter",
    "class":        coding.EncounterClass.Ambulatory,
}
```

## Regenerating from the Node SDK

```bash
# from the go-satusehat root
node scripts/generate.mjs ../satusehat
gofmt -w .
go vet ./...
go test ./...
```

The generator reads `src/fhir/*.ts`, `src/rawat-jalan/*.ts`, `src/gigi/*.ts`, `src/farmasi/*.ts`, `src/klaim-swasta/*.ts`, `src/klaim-bpjs/*.ts` via the TypeScript compiler API and produces one Go package per group (one file per TS module plus `namespace.go`) and the root `zz_namespaces.go` that hangs them off `*Client`. Shared types (`Resource`, `Bundle`, `PatchOp`, `RequestOptions`) live in `internal/core` and are re-exported from the root package. The shared source of truth stays the Node SDK — re-run after every Node SDK change.

## Caveats

These are inherited from the Node SDK (see `wiki/Known-Issues.md` there):

- **OAuth2 base URL** defaults to the same staging/production hosts as the FHIR base. Override via `Config.AuthBaseURL` if your tenant differs.
- **PATCH Content-Type** is `application/json-patch+json` automatically (FHIR JSON Patch RFC 6902).
- **Webhooks** have no auth header on `SendWebhookCallback` — the callback URL is whatever the server provided during handshake.
- **`expires_in` from the token endpoint** comes back as either a number or a string depending on server mood; the SDK accepts both.
- **KYC Alkes V3** carries a Basic-auth header per the source collection; if your tenant requires that, pass `AlkesOptions{Authorization: "Basic ..."}`.

## License

MIT — same as the Node SDK.