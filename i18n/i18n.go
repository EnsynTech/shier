package i18n

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// LangEN is English.
	LangEN = "en"
	// LangZH is Simplified Chinese.
	LangZH = "zh"
	cookieName = "gophish_lang"
)

var (
	once      sync.Once
	loadErr   error
	catalogs  = map[string]map[string]string{}
	supported = map[string]bool{LangEN: true, LangZH: true}
)

// DefaultLang is used when no preference is found.
const DefaultLang = LangZH

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
	if lang == "zh-cn" || lang == "zh-hans" || lang == "cn" {
		return LangZH
	}
	if lang == "en-us" || lang == "en-gb" {
		return LangEN
	}
	if supported[lang] {
		return lang
	}
	return DefaultLang
}

// FromRequest resolves the language from cookie, then Accept-Language, then default.
func FromRequest(r *http.Request) string {
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		return Normalize(c.Value)
	}
	accept := r.Header.Get("Accept-Language")
	if accept == "" {
		return DefaultLang
	}
	// Take the first tag (e.g. "zh-CN,zh;q=0.9,en;q=0.8")
	parts := strings.Split(accept, ",")
	if len(parts) == 0 {
		return DefaultLang
	}
	tag := strings.TrimSpace(strings.Split(parts[0], ";")[0])
	if strings.HasPrefix(strings.ToLower(tag), "zh") {
		return LangZH
	}
	if strings.HasPrefix(strings.ToLower(tag), "en") {
		return LangEN
	}
	return DefaultLang
}

// SetCookie stores the language preference.
func SetCookie(w http.ResponseWriter, lang string) {
	lang = Normalize(lang)
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    lang,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
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

// FuncMap returns a template FuncMap with a T function bound to lang.
func FuncMap(lang string) map[string]interface{} {
	lang = Normalize(lang)
	return map[string]interface{}{
		"T": func(key string) string {
			return T(lang, key)
		},
	}
}
