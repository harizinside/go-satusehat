// Package coding holds FHIR/Kemkes terminology constants so you rarely hand-type
// a {system, code, display} triple:
//
//	enc := map[string]any{"class": coding.EncounterClass.Ambulatory}
//
// Constants are generated from the Node SDK (see scripts/generate.mjs); the
// helpers in this file are hand-written. ICD-10 / ICD-9-CM live in the
// separate icd10 and icd9cm packages.
package coding

import "fmt"

// Coding is a FHIR Coding.
type Coding struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display,omitempty"`
}

// Identifier is a FHIR Identifier ({system, value} — not a Coding).
type Identifier struct {
	System string `json:"system"`
	Value  string `json:"value"`
	Use    string `json:"use,omitempty"`
}

// KemkesResourceIdentifierSystem builds the org-scoped sys-ids namespace,
// e.g. ("Encounter", orgID, "") → http://sys-ids.kemkes.go.id/Encounter/{orgID}.
func KemkesResourceIdentifierSystem(resourceType, orgID, subPath string) string {
	s := fmt.Sprintf("http://sys-ids.kemkes.go.id/%s/%s", resourceType, orgID)
	if subPath != "" {
		s += "/" + subPath
	}
	return s
}

// NIK, NIKIbu, KK, Paspor and IHSNumber build Patient.identifier entries.
func NIK(v string) Identifier       { return Identifier{System: IdentifierSystem.Nik, Value: v} }
func NIKIbu(v string) Identifier    { return Identifier{System: IdentifierSystem.NikIbu, Value: v} }
func KK(v string) Identifier        { return Identifier{System: IdentifierSystem.Kk, Value: v} }
func Paspor(v string) Identifier    { return Identifier{System: IdentifierSystem.Paspor, Value: v} }
func IHSNumber(v string) Identifier { return Identifier{System: IdentifierSystem.IhsNumber, Value: v} }

// NationalPrescriptionIdentifier builds a No. Resep Nasional identifier (Farmasi flow).
func NationalPrescriptionIdentifier(no string) Identifier {
	return Identifier{System: System.PrescriptionNational, Value: no}
}

// Extension is a FHIR extension carrying a CodeableConcept.
type Extension struct {
	URL                  string `json:"url"`
	ValueCodeableConcept struct {
		Coding []Coding `json:"coding"`
	} `json:"valueCodeableConcept"`
}

// MedicationTypeExtension builds the Medication.extension SATU SEHAT requires
// (e.g. with KemkesMedicationType.NonCompound).
func MedicationTypeExtension(c Coding) Extension {
	e := Extension{URL: MedicationTypeExtensionURL}
	e.ValueCodeableConcept.Coding = []Coding{c}
	return e
}
