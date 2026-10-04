package controllers

import "testing"

func TestSafeLocalPath(t *testing.T) {
	cases := map[string]string{
		"":                     "/",
		"/campaigns":           "/campaigns",
		"/login?next=/groups":  "/login?next=/groups",
		"//evil.example":       "/",
		"https://evil.example": "/",
		"/\\evil":              "/",
		"campaigns":            "/",
	}
	for in, want := range cases {
		if got := safeLocalPath(in); got != want {
			t.Errorf("safeLocalPath(%q) = %q, want %q", in, got, want)
		}
	}
}
