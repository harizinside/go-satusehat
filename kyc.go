package satusehat

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/harizinside/go-satusehat/internal/core"
)

// KycKeyPair holds the PEM-encoded RSA keys used by SATU SEHAT's KYC flow.
type KycKeyPair struct {
	PublicKey  string // PEM-encoded public key (PKIX)
	PrivateKey string // PEM-encoded private key (PKCS#8)
}

// GenerateKycKeyPair creates a fresh 2048-bit RSA keypair. SATU SEHAT expects
// a brand-new keypair per request — store the private key alongside the
// matching frameToken for the duration of one KYC session.
func GenerateKycKeyPair() (KycKeyPair, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return KycKeyPair{}, fmt.Errorf("kyc: generate rsa key: %w", err)
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return KycKeyPair{}, fmt.Errorf("kyc: marshal public key: %w", err)
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return KycKeyPair{}, fmt.Errorf("kyc: marshal private key: %w", err)
	}
	return KycKeyPair{
		PublicKey:  string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})),
		PrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})),
	}, nil
}

// EncryptKycMessage builds a SATU SEHAT KYC envelope: RSA-OAEP-wrap a random
// AES-256 key with the recipient's RSA public key, AES-256-GCM-encrypt the
// message, concatenate `wrappedKey(256) + iv(12) + ciphertext + tag(16)`,
// base64, and wrap in `-----BEGIN/END ENCRYPTED MESSAGE-----`. Matches
// SATU SEHAT's official PHP client (phpseclib3 RSA OAEP, default hash sha256)
// byte-for-byte.
func EncryptKycMessage(message, recipientPublicKeyPEM string) (string, error) {
	pub, err := parseRSAPublicKey(recipientPublicKeyPEM)
	if err != nil {
		return "", err
	}

	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return "", fmt.Errorf("kyc: read aes key: %w", err)
	}
	iv := make([]byte, 12)
	if _, err := rand.Read(iv); err != nil {
		return "", fmt.Errorf("kyc: read iv: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("kyc: new aes cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("kyc: new gcm: %w", err)
	}
	ciphertext := aead.Seal(nil, iv, []byte(message), nil)
	tag := ciphertext[len(ciphertext)-aead.Overhead():]
	body := ciphertext[:len(ciphertext)-aead.Overhead()]

	// RSA-OAEP-SHA256. Go's RSA implementation accepts the hash function via
	// crypto.SHA256 directly (no need to invoke a constructor).
	wrappedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, aesKey, nil)
	if err != nil {
		return "", fmt.Errorf("kyc: rsa-oaep wrap: %w", err)
	}

	out := make([]byte, 0, len(wrappedKey)+len(iv)+len(body)+len(tag))
	out = append(out, wrappedKey...)
	out = append(out, iv...)
	out = append(out, body...)
	out = append(out, tag...)

	b64 := base64.StdEncoding.EncodeToString(out)
	return beginTag + "\r\n" + chunkSplit(b64, 76) + endTag, nil
}

// DecryptKycMessage reverses `EncryptKycMessage` using the matching RSA
// private key.
func DecryptKycMessage(message, privateKeyPEM string) (string, error) {
	priv, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return "", err
	}

	body := stripKycEnvelope(message)
	payload, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return "", fmt.Errorf("kyc: base64 decode: %w", err)
	}
	if len(payload) < 256+12+16 {
		return "", fmt.Errorf("kyc: payload too short (%d bytes)", len(payload))
	}

	wrappedKey := payload[:256]
	rest := payload[256:]
	iv := rest[:12]
	tag := rest[len(rest)-16:]
	ciphertext := rest[12 : len(rest)-16]

	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, wrappedKey, nil)
	if err != nil {
		return "", fmt.Errorf("kyc: rsa-oaep unwrap: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("kyc: new aes cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("kyc: new gcm: %w", err)
	}
	plain, err := aead.Open(nil, iv, append(ciphertext, tag...), nil)
	if err != nil {
		return "", fmt.Errorf("kyc: gcm open: %w", err)
	}
	return string(plain), nil
}

// GenerateKycUrl POSTs to `{kycBaseUrl}/generate-url`. See the Node SDK
// source for the full protocol (SATU SEHAT's envelope: generate a fresh
// RSA keypair, send `{agent_name, agent_nik, public_key}` RSA-OAEP +
// AES-256-GCM encrypted with SATU SEHAT's own public key, as text/plain;
// the response comes back encrypted with that same fresh public key, so
// this function decrypts it with the matching private key before returning).
func GenerateKycUrl(ctx context.Context, client *Client, params GenerateKycUrlParams, options GenerateUrlOptions) (any, error) {
	kp, err := GenerateKycKeyPair()
	if err != nil {
		return nil, err
	}
	plaintext, err := encodeJSON(map[string]any{
		"agent_name": params.AgentName,
		"agent_nik":  params.AgentNIK,
		"public_key": kp.PublicKey,
	})
	if err != nil {
		return nil, err
	}
	encryptedBody, err := EncryptKycMessage(plaintext, params.SatuSehatPublicKey)
	if err != nil {
		return nil, err
	}

	resp, err := client.RequestURL(ctx, "POST", kycBase(client, options.BaseURL)+"/generate-url", &RequestOptions{
		Body:    encryptedBody,
		Headers: map[string]string{"Content-Type": "text/plain"},
	})
	if err != nil {
		return nil, err
	}

	decrypted, err := DecryptKycMessage(toString(resp), kp.PrivateKey)
	if err != nil {
		return nil, err
	}
	decoded, err := decodeJSONBytes([]byte(decrypted))
	if err != nil {
		return nil, err
	}
	return map[string]any{"data": decoded, "privateKey": kp.PrivateKey}, nil
}

// GenerateKycUrlParams is the input for `GenerateKycUrl`.
type GenerateKycUrlParams struct {
	AgentName          string
	AgentNIK           string
	SatuSehatPublicKey string
}

// GenerateUrlOptions lets callers override the KYC base URL.
type GenerateUrlOptions struct {
	BaseURL string
}

// GenerateChallengeCode POSTs to `{kycBaseUrl}/challenge-code`. Needs the
// same RSA+AES-GCM envelope as `generate-url`, but *without* a `public_key`
// in the body — the server encrypts the reply using the agent's key already
// registered via `generate-url`, identified by `frameToken` (the `token`
// field from that response).
func GenerateChallengeCode(ctx context.Context, client *Client, params GenerateChallengeCodeParams, options ChallengeCodeOptions) (any, error) {
	plaintext, err := encodeJSON(map[string]any{
		"metadata": params.Metadata,
		"data":     params.Data,
	})
	if err != nil {
		return nil, err
	}
	encryptedBody, err := EncryptKycMessage(plaintext, params.SatuSehatPublicKey)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Content-Type":  "text/plain",
		"X-Frame-Token": params.FrameToken,
	}
	if options.DebugMode {
		headers["X-Debug-Mode"] = "1"
	}

	resp, err := client.RequestURL(ctx, "POST", kycBase(client, options.BaseURL)+"/challenge-code", &RequestOptions{
		Body:    encryptedBody,
		Headers: headers,
	})
	if err != nil {
		return nil, err
	}

	decrypted, err := DecryptKycMessage(toString(resp), params.PrivateKey)
	if err != nil {
		return nil, err
	}
	return decodeJSONBytes([]byte(decrypted))
}

// GenerateChallengeCodeParams is the input for `GenerateChallengeCode`.
type GenerateChallengeCodeParams struct {
	Metadata           map[string]any
	Data               map[string]any
	SatuSehatPublicKey string
	FrameToken         string
	PrivateKey         string
}

// ChallengeCodeOptions lets callers set debug mode or override the base URL.
type ChallengeCodeOptions struct {
	DebugMode bool
	BaseURL   string
}

// kycBase is the per-call override or, when empty, Config.KycBaseURL.
func kycBase(c *Client, override string) string {
	if override == "" {
		override = c.config.KycBaseURL
	}
	return trimTrailingSlash(override)
}

// — internals ————————————————————————————————————————————————————————————

const (
	beginTag = "-----BEGIN ENCRYPTED MESSAGE-----"
	endTag   = "-----END ENCRYPTED MESSAGE-----"
)

func parseRSAPublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("kyc: not a PEM-encoded public key")
	}
	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		if rsaPub, ok := pub.(*rsa.PublicKey); ok {
			return rsaPub, nil
		}
		return nil, errors.New("kyc: public key is not RSA")
	}
	if pub, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return pub, nil
	}
	return nil, errors.New("kyc: failed to parse public key as PKIX or PKCS#1")
}

func parseRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("kyc: not a PEM-encoded private key")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
		return nil, errors.New("kyc: private key is not RSA")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	return nil, errors.New("kyc: failed to parse private key as PKCS#8 or PKCS#1")
}

func chunkSplit(s string, size int) string {
	var b strings.Builder
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		b.WriteString(s[i:end])
		b.WriteString("\r\n")
	}
	return b.String()
}

func stripKycEnvelope(s string) string {
	s = strings.ReplaceAll(s, beginTag, "")
	s = strings.ReplaceAll(s, endTag, "")
	// Trim all whitespace, mirroring PHP's `preg_replace('/[\r\n\s]/', ...)`.
	var b strings.Builder
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r', '\f', '\v':
			// drop
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case json.RawMessage:
		return string(x)
	default:
		return fmt.Sprint(v)
	}
}

func encodeJSON(v any) (string, error) {
	data, err := core.MarshalJSON(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodeJSONBytes(b []byte) (any, error) {
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("kyc: json decode: %w", err)
	}
	return out, nil
}
