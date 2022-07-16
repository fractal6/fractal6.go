package web

import (
    "fmt"
    "os"
	"path"
    "path/filepath"
    "strings"
    "net/http"
    "golang.org/x/text/language"
    "github.com/go-chi/chi/v5"
)

var langsAvailable string
var langsD map[string]bool

func init() {
    langsD = make(map[string]bool)
    for _, l := range strings.Split(langsAvailable, " ") {
        langsD[l] = true
    }
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
// FileServer is serving static files
func FileServer(r chi.Router, public string, static string, maxage string) {

	if strings.ContainsAny(public, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	root, _ := filepath.Abs(static)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		panic("Static Documents Directory Not Found")
	}

	fs := http.StripPrefix(public, http.FileServer(http.Dir(root)))

    // Ensure public path is a directory
	if public != "/" && public[len(public)-1] != '/' {
		r.Get(public, http.RedirectHandler(public+"/", 301).ServeHTTP)
		public += "/"
	}

	r.Get(public+"*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Headers
        // --
        var lang string
        var fn string
        // Cache control
        if maxage != "" {
            w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%s", maxage))
        }
        // Langugage
        if langs, _, err := language.ParseAcceptLanguage(r.Header.Get("Accept-Language")); err != nil {
            // Use default lang
            fmt.Println("Accept-Languge parsing error:, ", err.Error())
        } else if len(langs) > 0 {
            for _, l := range langs {
                b, _ := l.Base()
                lg := b.String()
                if langsD[lg] {
                    lang = lg
                    break
                }
            }
        }

        // Check file existence, and redicrect to index if file not exists
        file := strings.Replace(r.RequestURI, public, "/", 1)
        if _, err := os.Stat(root + file); os.IsNotExist(err) {
            if lang == "" {
                fn = "index.html"
            } else {
                fn = fmt.Sprintf("index.%s.html", lang)
            }
            http.ServeFile(w, r, path.Join(root, fn))
			return
		}

		fs.ServeHTTP(w, r)
	}))
}
