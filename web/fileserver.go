/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2022 Fractale Co
 *
 * This file is part of Fractale.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Fractale.  If not, see <http://www.gnu.org/licenses/>.
 */

package web

import (
	"fmt"
    "time"
    "strconv"
	"os"
	"path"
	"path/filepath"
	"strings"
	"net/http"
	"github.com/go-chi/chi/v5"
	"golang.org/x/text/language"
	"fractale/fractal6.go/db"
	"fractale/fractal6.go/web/auth"

)

var DEFAULT_LANG string = "en"
var langsAvailable string = "en_fr"// Replaced at build time. See Makefile
var langsD map[string]bool

func init() {
    langsD = make(map[string]bool)
    for _, l := range strings.Split(langsAvailable, "_") {
        if l == "" { continue }
        langsD[l] = true
    }
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
// FileServer is serving static files
func FileServer(r chi.Router, publicUri string, location string, maxage string) {

	if strings.ContainsAny(publicUri, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	root, _ := filepath.Abs(location)
	if _, err := os.Stat(root); os.IsNotExist(err) {
        panic("Public Documents Directory Not Found: "+location)
	}

	fs := http.StripPrefix(publicUri, http.FileServer(http.Dir(root)))

    // Ensure the given publicUri given is a directory
	if publicUri != "/" && publicUri[len(publicUri)-1] != '/' {
		r.Get(publicUri, http.RedirectHandler(publicUri+"/", 301).ServeHTTP)
		publicUri += "/"
	}

	r.Get(publicUri+"*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Headers
        // --
        // Set Cache control
        if maxage != "" {
            w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%s", maxage))
        }

        // Redirect to appropriate Language
        // Check file existence, and redirect to index if file not exists
        var lang string = DEFAULT_LANG // Default language
        fn := strings.Replace(r.RequestURI, publicUri, "/", 1)
        if fi, err := os.Stat(root + fn); err == nil && !fi.IsDir() {
            // Serve requested file if path exists on filesystem and is not dir.
            fs.ServeHTTP(w, r)
        } else if p := strings.Split(fn, "/"); len(p) > 1 && langsD[p[1]] {
            // Redirect to URI with Lang set in Cookie
            lang = p[1]
            http.SetCookie(w, &http.Cookie{
                Name: "lang",
                Value: lang,
                Path: "/",
                MaxAge: 7776000,
            })
            // Hack to by bypass (301 - Not Modified) state.
            rand := strconv.FormatInt(time.Now().Unix(), 10)
            http.Redirect(w, r, strings.TrimPrefix(r.RequestURI + "?" + rand, "/"+lang ), 301)
        } else {
            // Serve the index.html from asked or preferred Lang
            // 1. use user setting if logged.
            // 2. use cookie seting if present.
            // 3. user browser preference if present.
            // 4. use default language.
            if _, uctx, err := auth.GetUserContext(r.Context()); err == nil {
                // Lang may not be updated
                if l, err := db.GetDB().GetFieldByEq("User.username", uctx.Username, "User.lang"); err != nil {
                    lang = string(uctx.Lang)
                } else {
                    lang = l.(string)
                }
                lang = strings.ToLower(lang)
            } else if c, err := r.Cookie("lang"); err == nil && langsD[c.Value] {
                lang = c.Value
            } else if langs, _, err := language.ParseAcceptLanguage(r.Header.Get("Accept-Language")); err != nil {
                fmt.Println("Accept-Languge parsing error:, ", err.Error())
            } else if len(langs) > 0 { // Try to get the preferred language
                for _, l := range langs {
                    b, _ := l.Base()
                    lg := b.String()
                    if langsD[lg] {
                        lang = lg
                        break
                    }
                }
            }
            // -- Lang to Path
            if strings.HasPrefix(fn, "/static/") {
                fn = fmt.Sprintf("%s/%s", lang, fn)
            } else {
                fn = fmt.Sprintf("%s/index.html", lang)
            }

            http.ServeFile(w, r, path.Join(root, fn))
        }

	}))
}
