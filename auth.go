package satusehat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// AccessToken is the result of `GetAccessToken`. SATU SEHAT documents
// `expires_in` as a number of seconds, but the live server sometimes returns
// it as a string — `ExpiresIn` is always a `time.Duration`, parsed through
// both shapes.
type AccessToken struct {
	AccessToken string
	ExpiresIn   time.Duration  // 0 when the server didn't include it
	TokenType   string         // "Bearer" when empty in the response
	ExpiresAt   time.Time      // zero when `ExpiresIn` is 0
	Raw         map[string]any // the original response, parsed
}

// CredentialsOverride lets callers pass per-call credentials without
// mutating the client config.
type CredentialsOverride struct {
	ClientID     string
	ClientSecret string
}

// GetAccessToken mints a fresh OAuth2 client-credentials token from
// `{authBaseUrl}/accesstoken?grant_type=client_credentials`.
//
// The SDK deliberately does NOT cache or auto-refresh: store the returned
// token wherever your app already keeps state (Redis, DB, memory) and pass
// it back via the client's `Config.Token` / `SetToken()` on every call.
func GetAccessToken(ctx context.Context, c *Client, creds CredentialsOverride) (*AccessToken, error) {
	clientID := creds.ClientID
	if clientID == "" {
		clientID = c.config.ClientID
	}
	clientSecret := creds.ClientSecret
	if clientSecret == "" {
		clientSecret = c.config.ClientSecret
	}
	if clientID == "" || clientSecret == "" {
		return nil, errors.New("satusehat: GetAccessToken requires clientId/clientSecret — pass them to the client config or as the CredentialsOverride argument")
	}

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)

	endpoint := trimTrailingSlash(c.config.AuthBaseURL) + "/accesstoken?grant_type=client_credentials"

	authOff := false
	resp, err := c.RequestURL(ctx, "POST", endpoint, &RequestOptions{
		Auth:    &authOff,
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:    form.Encode(),
	})
	if err != nil {
		return nil, err
	}

	raw, ok := resp.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("satusehat: token response not a JSON object: %T", resp)
	}

	accessToken, _ := raw["access_token"].(string)
	if accessToken == "" {
		buf, _ := json.Marshal(raw)
		return nil, fmt.Errorf("satusehat: token response missing access_token: %s", string(buf))
	}

	result := &AccessToken{
		AccessToken: accessToken,
		Raw:         raw,
		TokenType:   stringFromRaw(raw["token_type"]),
	}

	switch v := raw["expires_in"].(type) {
	case float64:
		result.ExpiresIn = time.Duration(v * float64(time.Second))
	case string:
		if secs, err := strconv.ParseFloat(v, 64); err == nil {
			result.ExpiresIn = time.Duration(secs * float64(time.Second))
		}
	}
	if result.ExpiresIn > 0 {
		result.ExpiresAt = time.Now().Add(result.ExpiresIn)
	}
	return result, nil
}

// Authenticate mints a token and immediately installs it on the client.
func Authenticate(ctx context.Context, c *Client, creds CredentialsOverride) (*AccessToken, error) {
	tok, err := GetAccessToken(ctx, c, creds)
	if err != nil {
		return nil, err
	}
	c.SetToken(tok.AccessToken)
	return tok, nil
}

func stringFromRaw(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case json.RawMessage:
		return string(x)
	}
	return fmt.Sprint(v)
}
