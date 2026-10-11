package satusehat

import (
	"context"
	"encoding/json"
)

// Region is one record in the Master Data wilayah responses.
type Region struct {
	Code       string `json:"code"`
	ParentCode string `json:"parent_code"`
	BPSCode    string `json:"bps_code"`
	Name       string `json:"name"`
}

// RegionListResult is the shape shared by all Master Data wilayah responses.
type RegionListResult struct {
	Status  int         `json:"status"`
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    []Region    `json:"data"`
	Meta    *RegionMeta `json:"meta,omitempty"`
}

// RegionMeta is the optional `meta` field of RegionListResult.
type RegionMeta struct {
	ItemCount *int           `json:"item_count,omitempty"`
	Page      *RegionPage    `json:"page,omitempty"`
	Cursors   *RegionCursors `json:"cursors,omitempty"`
}

// RegionPage holds the pagination metadata.
type RegionPage struct {
	IsCursor  *bool `json:"is_cursor,omitempty"`
	Current   *int  `json:"current,omitempty"`
	Previous  *int  `json:"previous,omitempty"`
	Next      *int  `json:"next,omitempty"`
	Limit     *int  `json:"limit,omitempty"`
	TotalPage *int  `json:"total_page,omitempty"`
	Total     *int  `json:"total,omitempty"`
}

// RegionCursors is the optional cursor metadata.
type RegionCursors struct {
	Next     *string `json:"next,omitempty"`
	Previous *string `json:"previous,omitempty"`
}

// GetProvinces GETs `/masterdata/v1/provinces?codes=...`.
func GetProvinces(ctx context.Context, client *Client, params struct{ Codes string }) (*RegionListResult, error) {
	var q map[string]any
	if params.Codes != "" {
		q = map[string]any{"codes": params.Codes}
	}
	raw, err := client.RequestMasterData(ctx, "GET", "/masterdata/v1/provinces", &RequestOptions{Query: q})
	if err != nil {
		return nil, err
	}
	return decodeRegionList(raw)
}

// GetCities GETs `/masterdata/v1/cities?province_codes=...&codes=...`.
func GetCities(ctx context.Context, client *Client, params struct{ ProvinceCodes, Codes string }) (*RegionListResult, error) {
	q := map[string]any{}
	if params.ProvinceCodes != "" {
		q["province_codes"] = params.ProvinceCodes
	}
	if params.Codes != "" {
		q["codes"] = params.Codes
	}
	raw, err := client.RequestMasterData(ctx, "GET", "/masterdata/v1/cities", &RequestOptions{Query: q})
	if err != nil {
		return nil, err
	}
	return decodeRegionList(raw)
}

// GetDistricts GETs `/masterdata/v1/districts?city_codes=...&codes=...`.
func GetDistricts(ctx context.Context, client *Client, params struct{ CityCodes, Codes string }) (*RegionListResult, error) {
	q := map[string]any{}
	if params.CityCodes != "" {
		q["city_codes"] = params.CityCodes
	}
	if params.Codes != "" {
		q["codes"] = params.Codes
	}
	raw, err := client.RequestMasterData(ctx, "GET", "/masterdata/v1/districts", &RequestOptions{Query: q})
	if err != nil {
		return nil, err
	}
	return decodeRegionList(raw)
}

// GetSubDistricts GETs `/masterdata/v1/sub-districts?district_codes=...&codes=...`.
func GetSubDistricts(ctx context.Context, client *Client, params struct{ DistrictCodes, Codes string }) (*RegionListResult, error) {
	q := map[string]any{}
	if params.DistrictCodes != "" {
		q["district_codes"] = params.DistrictCodes
	}
	if params.Codes != "" {
		q["codes"] = params.Codes
	}
	raw, err := client.RequestMasterData(ctx, "GET", "/masterdata/v1/sub-districts", &RequestOptions{Query: q})
	if err != nil {
		return nil, err
	}
	return decodeRegionList(raw)
}

// RegionPageParams is the cursor-pagination input for the v2 endpoints.
type RegionPageParams struct {
	CurrentPage   int    `url:"current_page,omitempty"`
	Next          string `url:"next,omitempty"`
	Prev          string `url:"prev,omitempty"`
	Codes         string `url:"codes,omitempty"`
	ProvinceCodes string `url:"province_codes,omitempty"`
	CityCodes     string `url:"city_codes,omitempty"`
	DistrictCodes string `url:"district_codes,omitempty"`
}

// GetProvincesV2 GETs `/masterdata/v2/provinces` (cursor paginated).
func GetProvincesV2(ctx context.Context, client *Client, params RegionPageParams) (*RegionListResult, error) {
	raw, err := client.RequestMasterData(ctx, "GET", "/masterdata/v2/provinces", &RequestOptions{Query: params})
	if err != nil {
		return nil, err
	}
	return decodeRegionList(raw)
}

// GetCitiesV2 GETs `/masterdata/v2/cities`.
func GetCitiesV2(ctx context.Context, client *Client, params RegionPageParams) (*RegionListResult, error) {
	raw, err := client.RequestMasterData(ctx, "GET", "/masterdata/v2/cities", &RequestOptions{Query: params})
	if err != nil {
		return nil, err
	}
	return decodeRegionList(raw)
}

// GetDistrictsV2 GETs `/masterdata/v2/districts`.
func GetDistrictsV2(ctx context.Context, client *Client, params RegionPageParams) (*RegionListResult, error) {
	raw, err := client.RequestMasterData(ctx, "GET", "/masterdata/v2/districts", &RequestOptions{Query: params})
	if err != nil {
		return nil, err
	}
	return decodeRegionList(raw)
}

// GetSubDistrictsV2 GETs `/masterdata/v2/sub-districts`.
func GetSubDistrictsV2(ctx context.Context, client *Client, params RegionPageParams) (*RegionListResult, error) {
	raw, err := client.RequestMasterData(ctx, "GET", "/masterdata/v2/sub-districts", &RequestOptions{Query: params})
	if err != nil {
		return nil, err
	}
	return decodeRegionList(raw)
}

// MasterSarana is one row of the master-sarana index.
type MasterSarana struct {
	KodeSatusehat      string    `json:"kode_satusehat"`
	KodeSarana         string    `json:"kode_sarana"`
	Nama               string    `json:"nama"`
	Telp               *string   `json:"telp,omitempty"`
	Email              *string   `json:"email,omitempty"`
	Website            *string   `json:"website,omitempty"`
	Longitude          *float64  `json:"longitude,omitempty"`
	Latitude           *float64  `json:"latitude,omitempty"`
	Operasional        *bool     `json:"operasional,omitempty"`
	Alamat             *string   `json:"alamat,omitempty"`
	SaranaAdministrasi *Resource `json:"sarana_administrasi,omitempty"`
}

// MasterSaranaResult is the response shape of the master-sarana endpoint.
type MasterSaranaResult struct {
	StatusCode int            `json:"status_code"`
	Message    string         `json:"message"`
	Page       int            `json:"page"`
	TotalPage  int            `json:"total_page"`
	Data       []MasterSarana `json:"data"`
}

// MasterSaranaParams is the input for `GetMasterSarana`.
type MasterSaranaParams struct {
	Limit                int    `url:"limit,omitempty"`
	Page                 int    `url:"page,omitempty"`
	LowerBoundUpdatedAt  string `url:"lower_bound_updated_at,omitempty"`
	UpperBoundUpdatedAt  string `url:"upper_bound_updated_at,omitempty"`
	KodeSatusehat        string `url:"kode_satusehat,omitempty"`
	KodeSarana           string `url:"kode_sarana,omitempty"`
	JenisSarana          string `url:"jenis_sarana,omitempty"`
	Nama                 string `url:"nama,omitempty"`
	KodeProvinsi         string `url:"kode_provinsi,omitempty"`
	KodeKabkota          string `url:"kode_kabkota,omitempty"`
	KodeKecamatan        string `url:"kode_kecamatan,omitempty"`
	StatusAktif          *bool  `url:"status_aktif,omitempty"`
	StatusSarana         string `url:"status_sarana,omitempty"`
	SumberIdentifier     string `url:"sumber_identifier,omitempty"`
	IdentifierKodeSarana string `url:"identifier_kode_sarana,omitempty"`
}

// GetMasterSarana GETs `/masterdata/v1/mastersaranaindex/mastersarana`.
func GetMasterSarana(ctx context.Context, client *Client, params MasterSaranaParams) (*MasterSaranaResult, error) {
	raw, err := client.RequestMasterData(ctx, "GET", "/masterdata/v1/mastersaranaindex/mastersarana", &RequestOptions{Query: params})
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return &MasterSaranaResult{}, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var out MasterSaranaResult
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func decodeRegionList(raw any) (*RegionListResult, error) {
	if raw == nil {
		return &RegionListResult{}, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var out RegionListResult
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
