// Package icd9cm is the full Kemkes e-klaim ICD-9-CM catalog. It lives in its own
// package so programs that never import it don't carry the data.
package icd9cm

import (
	"strings"
	"sync"

	"github.com/harizinside/go-satusehat/coding"
)

// Count is the number of codes in the catalog.
const Count = len(entries)

var lookup = sync.OnceValue(func() map[string]string {
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		m[e[0]] = e[1]
	}
	return m
})

// Lookup returns the Coding for code, or false if the code is unknown.
func Lookup(code string) (coding.Coding, bool) {
	display, ok := lookup()[code]
	return coding.Coding{System: system, Code: code, Display: display}, ok
}

// Search does a case-insensitive substring match over displays, returning at most limit hits.
func Search(query string, limit int) []coding.Coding {
	q := strings.ToLower(query)
	var hits []coding.Coding
	if q == "" {
		return hits
	}
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e[1]), q) {
			hits = append(hits, coding.Coding{System: system, Code: e[0], Display: e[1]})
			if len(hits) >= limit {
				break
			}
		}
	}
	return hits
}
