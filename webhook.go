package satusehat

import (
	"context"
	"encoding/json"
)

// WebhookMethod is the `meta.method` field on a Katalog Webhook (Klaim
// Swasta) envelope.
type WebhookMethod string

const (
	WebhookCoverageEligibilityResponseSubmission WebhookMethod = "coverageEligibilityResponseSubmission"
	WebhookChargeItemSubmission                  WebhookMethod = "chargeItemSubmission"
	WebhookChargeItemResponseSubmission          WebhookMethod = "chargeItemResponseSubmission"
	WebhookBillingStatusSubmission               WebhookMethod = "billingStatusSubmission"
	WebhookClaimBundle                           WebhookMethod = "claimBundle"
	WebhookClaimResponseSubmission               WebhookMethod = "claimResponseSubmission"
	WebhookPaymentNoticeSubmission               WebhookMethod = "paymentNoticeSubmission"
	WebhookPaymentReconciliationSubmission       WebhookMethod = "paymentReconciliationSubmission"
)

// SatuSehatWebhookPayload is the envelope SATU SEHAT posts to subscriber
// webhook URLs.
type SatuSehatWebhookPayload struct {
	Data          map[string]any `json:"data"`
	Meta          map[string]any `json:"meta"`
	TransactionID string         `json:"transaction_id"`
	Timestamp     string         `json:"timestamp"`
	Raw           []byte         `json:"-"` // preserved verbatim
}

// UnmarshalJSON keeps the original bytes so callers can echo them back
// unchanged when acknowledging via `SendWebhookCallback`.
func (w *SatuSehatWebhookPayload) UnmarshalJSON(data []byte) error {
	type shadow SatuSehatWebhookPayload
	var s shadow
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*w = SatuSehatWebhookPayload(s)
	w.Raw = append([]byte(nil), data...)
	return nil
}

// ParseWebhookPayload narrows a raw webhook body into the typed envelope. If
// the body is not the expected shape it returns nil.
func ParseWebhookPayload(raw []byte) *SatuSehatWebhookPayload {
	if len(raw) == 0 {
		return nil
	}
	var p SatuSehatWebhookPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil
	}
	if p.Data == nil || p.Meta == nil {
		return nil
	}
	p.Raw = append([]byte(nil), raw...)
	return &p
}

// ParseWebhookPayloadAs parses and additionally verifies that the envelope's
// `meta.method` matches `method`.
func ParseWebhookPayloadAs(raw []byte, method WebhookMethod) *SatuSehatWebhookPayload {
	p := ParseWebhookPayload(raw)
	if p == nil {
		return nil
	}
	if got, _ := p.Meta["method"].(string); got != string(method) {
		return nil
	}
	return p
}

// SendWebhookCallback acknowledges the webhook by POSTing the original
// envelope to `callbackUrl`. Authentication is skipped (matching the TS
// SDK): the callback URL is whatever the server provided during handshake.
func SendWebhookCallback(ctx context.Context, client *Client, callbackURL string, payload SatuSehatWebhookPayload) (any, error) {
	// Echo the original bytes when we have them so unmodelled fields survive.
	var body any = payload.Raw
	if len(payload.Raw) == 0 {
		type plain SatuSehatWebhookPayload
		body = plain(payload)
	}
	authOff := false
	return client.RequestURL(ctx, "POST", callbackURL, &RequestOptions{
		Auth:    &authOff,
		Body:    body,
		Headers: map[string]string{"Content-Type": "application/json"},
	})
}
