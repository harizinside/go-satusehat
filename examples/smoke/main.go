// Package main is a smoke test against the live staging API. Run with:
//
//	SATUSEHAT_CLIENT_ID=... \
//	SATUSEHAT_CLIENT_SECRET=... \
//	go run ./examples/smoke
//
// It exercises: get an access token, fetch provinces (master data v1), and
// look up a Patient by NIK if the env var `SATUSEHAT_TEST_NIK` is set.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/harizinside/go-satusehat"
)

func main() {
	id := os.Getenv("SATUSEHAT_CLIENT_ID")
	sec := os.Getenv("SATUSEHAT_CLIENT_SECRET")
	if id == "" || sec == "" {
		fmt.Fprintln(os.Stderr, "set SATUSEHAT_CLIENT_ID and SATUSEHAT_CLIENT_SECRET")
		os.Exit(2)
	}

	c := satusehat.New(satusehat.Config{
		Environment:  satusehat.Staging,
		ClientID:     id,
		ClientSecret: sec,
	})
	ctx := context.Background()

	tok, err := c.Auth.GetAccessToken(ctx, satusehat.CredentialsOverride{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "token:", err)
		os.Exit(1)
	}
	fmt.Printf("token issued, expires_in=%v\n", tok.ExpiresIn)
	c.SetToken(tok.AccessToken)

	provinces, err := c.MasterData.Provinces(ctx, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "provinces:", err)
		os.Exit(1)
	}
	fmt.Printf("provinces: %d\n", len(provinces.Data))

	if nik := os.Getenv("SATUSEHAT_TEST_NIK"); nik != "" {
		res, err := c.Search.Patient(ctx, satusehat.PersonSearchParams{
			Identifier: satusehat.NIKIdentifier(nik),
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "patient:", err)
			os.Exit(1)
		}
		buf, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println("patient:", string(buf))
	}
}
