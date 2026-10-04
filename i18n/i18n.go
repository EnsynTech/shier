package i18n

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const (
	// LangEN is English.
	LangEN = "en"
	// LangZH is Simplified Chinese.
	LangZH = "zh"
)

var (
	once      sync.Once
	loadErr   error
	catalogs  = map[string]map[string]string{}
	supported = map[string]bool{LangEN: true, LangZH: true}
)

// DefaultLang is used when no preference is found.
const DefaultLang = LangZH

// LangCookie stores an explicit 中/EN choice and overrides Accept-Language.
const LangCookie = "shier_lang"

func localeDirs() []string {
	return []string{
		filepath.Join("i18n", "locales"),
		"locales",
	}
}

func load() {
	once.Do(func() {
		var base string
		for _, dir := range localeDirs() {
			if _, err := ioutil.ReadFile(filepath.Join(dir, LangEN+".json")); err == nil {
				base = dir
				break
			}
		}
		if base == "" {
			loadErr = os.ErrNotExist
			return
		}
		for _, lang := range []string{LangEN, LangZH} {
			path := filepath.Join(base, lang+".json")
			data, err := ioutil.ReadFile(path)
			if err != nil {
				loadErr = err
				return
			}
			m := map[string]string{}
			if err := json.Unmarshal(data, &m); err != nil {
				loadErr = err
				return
			}
			catalogs[lang] = m
		}
	})
}

// LoadError returns any error encountered while loading locale files.
func LoadError() error {
	load()
	return loadErr
}

// Normalize returns a supported language code, or DefaultLang.
func Normalize(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "zh-cn" || lang == "zh-hans" || lang == "zh-tw" || lang == "zh-hant" || lang == "cn" {
		return LangZH
	}
	if strings.HasPrefix(lang, "zh") {
		return LangZH
	}
	if lang == "en-us" || lang == "en-gb" || strings.HasPrefix(lang, "en") {
		return LangEN
	}
	if supported[lang] {
		return lang
	}
	return DefaultLang
}

// matchLang maps a single Accept-Language tag to a supported language, or "".
func matchLang(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if tag == "" || tag == "*" {
		return ""
	}
	if strings.HasPrefix(tag, "zh") {
		return LangZH
	}
	if strings.HasPrefix(tag, "en") {
		return LangEN
	}
	return ""
}

// FromRequest resolves the language from the shier_lang cookie, then Accept-Language.
func FromRequest(r *http.Request) string {
	if c, err := r.Cookie(LangCookie); err == nil {
		switch strings.ToLower(strings.TrimSpace(c.Value)) {
		case LangEN:
			return LangEN
		case LangZH:
			return LangZH
		}
	}
	accept := r.Header.Get("Accept-Language")
	if accept == "" {
		return DefaultLang
	}

	type pref struct {
		lang string
		q    float64
	}
	var prefs []pref
	for _, part := range strings.Split(accept, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		tag := part
		q := 1.0
		if i := strings.Index(part, ";"); i >= 0 {
			tag = strings.TrimSpace(part[:i])
			params := part[i+1:]
			for _, p := range strings.Split(params, ";") {
				p = strings.TrimSpace(p)
				if strings.HasPrefix(strings.ToLower(p), "q=") {
					if v, err := strconv.ParseFloat(strings.TrimSpace(p[2:]), 64); err == nil {
						q = v
					}
				}
			}
		}
		if lang := matchLang(tag); lang != "" {
			prefs = append(prefs, pref{lang: lang, q: q})
		}
	}
	if len(prefs) == 0 {
		return DefaultLang
	}
	best := prefs[0]
	for _, p := range prefs[1:] {
		if p.q > best.q {
			best = p
		}
	}
	return best.lang
}

// T returns the translation for key in the given language.
// Falls back to English, then the key itself.
func T(lang, key string) string {
	load()
	lang = Normalize(lang)
	if m, ok := catalogs[lang]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	if lang != LangEN {
		if m, ok := catalogs[LangEN]; ok {
			if v, ok := m[key]; ok {
				return v
			}
		}
	}
	return key
}

// Catalog returns a copy of all strings for a language (for JS injection).
func Catalog(lang string) map[string]string {
	load()
	lang = Normalize(lang)
	src := catalogs[lang]
	if src == nil {
		src = catalogs[LangEN]
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// SetLangCookie records an explicit language choice for one year.
func SetLangCookie(w http.ResponseWriter, lang string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     LangCookie,
		Value:    lang,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}

// FuncMap returns a template FuncMap with a T function bound to lang.
func FuncMap(lang string) map[string]interface{} {
	lang = Normalize(lang)
	return map[string]interface{}{
		"T": func(key string) string {
			return T(lang, key)
		},
	}
}
