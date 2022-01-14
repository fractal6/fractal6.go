package middleware

import (
    "fmt"
	"net/http"
    "bytes"
    "io/ioutil"
    "encoding/json"
    "zerogov/fractal6.go/db"
    webauth "zerogov/fractal6.go/web/auth"
)


//CheckRecursiveQueryRights check if the query can be executed where a
// a the body is expected to be string/nameid.
func CheckRecursiveQueryRights(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        //ctx := r.Context()
        //uctx := webauth.GetUserContextOrEmpty(r.Context())
        //var q string

        //// Keep this to reset the body reader later
        //body, _ := ioutil.ReadAll(r.Body)

        //// reset the body reader
        //r.Body = ioutil.NopCloser(bytes.NewReader(body))
        //// Get the JSON body and decode into UserCreds
        //err := json.NewDecoder(r.Body).Decode(&q)
        //if err != nil {
        //    // Body structure error
        //    http.Error(w, err.Error(), 400)
        //    return
        //}
        //// reset the body reader agin
        //r.Body = ioutil.NopCloser(bytes.NewReader(body))

        //// This test is not enough, as private node will be return below.
        //input := map[string]string{"key":"nameid", "value": q}
        //res, err := db.GetDB().Get(uctx, "node", input)
        //if err != nil { http.Error(w, err.Error(), 400); return }

        //// Failed silently, or with discretion....
        //if res == "" {
        //    w.Write([]byte("[]"))
        //    return
        //}

		//next.ServeHTTP(w, r.WithContext(ctx))
	})
}

//CheckTensionQueryRights remove all unauthorize nameids where a
// the body represent a {db.TensionQuery}.
func CheckTensionQueryRights(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        uctx := webauth.GetUserContextOrEmpty(r.Context())
        var q struct{Nameids []string}

        // Keep this to reset the body reader later
        body, _ := ioutil.ReadAll(r.Body)

        // reset the body reader
        r.Body = ioutil.NopCloser(bytes.NewReader(body))
        // Get the JSON body and decode into UserCreds
        err := json.NewDecoder(r.Body).Decode(&q)
        if err != nil {
            // Body structure error
            http.Error(w, err.Error(), 400)
            return
        }
        // reset the body reader agin
        r.Body = ioutil.NopCloser(bytes.NewReader(body))

        newNameids, err := db.GetDB().QueryAuthFilter(uctx, "node", "nameid", q.Nameids)
        // to be completed, rewrite body !?
        fmt.Println(newNameids)


        //// Failed silently, or with discretion....
        //if len(newNameids) == 0 {
        //    w.Write([]byte("[]"))
        //    return
        //}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

