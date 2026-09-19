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
	r.AddCookie(&http.Cookie{Name: cookieName, Value: "en"})
	if got := FromRequest(r); got != LangEN {
		t.Fatalf("cookie lang = %q", got)
	}

	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	if got := FromRequest(r2); got != LangZH {
		t.Fatalf("accept-language = %q", got)
	}
}
