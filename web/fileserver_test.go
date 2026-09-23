package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestLangRedirect(t *testing.T) {
	r := chi.NewRouter()
	FileServer(r, "/", t.TempDir(), "")

	cases := map[string]string{
		"/fr/user/settings":           "/user/settings?",
		"/fr/user/settings?m=profile": "/user/settings?m=profile&",
		"/fr//evil.com":               "/evil.com?",
		"/fr/%5Cevil.com":             "/%5Cevil.com?",
		"/fr/\\evil.com":              "/%5Cevil.com?",
	}
	for uri, want := range cases {
		req := httptest.NewRequest("GET", uri, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		loc := w.Header().Get("Location")
		if w.Code != http.StatusFound || !strings.HasPrefix(loc, want) {
			t.Errorf("%s: got %d %q, want 302 %q...", uri, w.Code, loc, want)
		}
	}
}
