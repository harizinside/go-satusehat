package satusehat

// trimTrailingSlash is the inverse of `strings.TrimRight(s, "/")` minus the
// awkward empty-result corner case.
func trimTrailingSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

// Ptr returns a pointer to v, for the pointer fields in generated request
// bodies: `Active: satusehat.Ptr(false)`.
func Ptr[T any](v T) *T { return &v }
