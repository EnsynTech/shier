package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestT(t *testing.T) {
	if err := LoadError(); err != nil {
		t.Fatalf("load locales: %v", err)
	}
	if got := T(LangZH, "nav.dashboard"); got != "仪表盘" {
		t.Fatalf("zh nav.dashboard = %q", got)
	}
	if got := T(LangEN, "nav.dashboard"); got != "Dashboard" {
		t.Fatalf("en nav.dashboard = %q", got)
	}
	if got := T(LangEN, "missing.key"); got != "missing.key" {
		t.Fatalf("missing key = %q", got)
	}
}

func TestFromRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if got := FromRequest(r); got != LangEN {
		t.Fatalf("accept-language en = %q", got)
	}

	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	if got := FromRequest(r2); got != LangZH {
		t.Fatalf("accept-language zh = %q", got)
	}

	r3 := httptest.NewRequest(http.MethodGet, "/", nil)
	r3.Header.Set("Accept-Language", "fr-FR,fr;q=0.9")
	if got := FromRequest(r3); got != DefaultLang {
		t.Fatalf("unsupported language fallback = %q", got)
	}

	// Prefer higher q even if listed later
	r4 := httptest.NewRequest(http.MethodGet, "/", nil)
	r4.Header.Set("Accept-Language", "zh-CN;q=0.5,en-US;q=0.9")
	if got := FromRequest(r4); got != LangEN {
		t.Fatalf("q-value preference = %q", got)
	}
}
