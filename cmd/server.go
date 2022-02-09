package cmd

import (
    //"fmt"
    "log"
    "time"
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
    "github.com/spf13/viper"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    "fractale/fractal6.go/web"
    "fractale/fractal6.go/web/auth"
    handle6 "fractale/fractal6.go/web/handlers"
    middle6 "fractale/fractal6.go/web/middleware"
)

var tkMaster *auth.Jwt
var buildMode string

func init() {
    // Get env mode
    if buildMode == "" {
        buildMode = "DEV"
    } else {
        buildMode = "PROD"
    }

    // Jwt init
    tkMaster = auth.GetTokenMaster()
}

// RunServer launch the server
func RunServer() {
    HOST := viper.GetString("server.host")
    PORT := viper.GetString("server.port")
    gqlConfig := viper.GetStringMap("graphql")
    instrumentation := viper.GetBool("server.prometheus_instrumentation")

    r := chi.NewRouter()

    var allowedOrigins []string
    if buildMode == "PROD" {
        allowedOrigins = append(allowedOrigins, "https://fractale.co")
    } else {
        allowedOrigins = append(allowedOrigins, "http://localhost:8000")
    }

	// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
	cors := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		//AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		//AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		//AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		//ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})

    // Middleware stack
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
	r.Use(cors.Handler)
    //r.Use(middle6.RequestContextMiddleware) // Set context info
    // JWT
    //r.Use(jwtauth.Verifier(tkMaster.GetAuth())) // Seek, verify and validate JWT token
    r.Use(middle6.JwtVerifier(tkMaster.GetAuth())) // Seek, verify and validate JWT token
    r.Use(middle6.JwtDecode) // Set user claims
    // Log request
    r.Use(middleware.Logger)
    // Recover from panic
    r.Use(middleware.Recoverer)
    // Set a timeout value on the request context (ctx), that will signal
    // through ctx.Done() that the request has timed out and further
    // processing should be stopped.
    r.Use(middleware.Timeout(60 * time.Second))

    // Auth API
    r.Group(func(r chi.Router) {
        //r.Use(middle6.EnsurePostMethod)
        r.Route("/auth", func(r chi.Router) {
            // User
            r.Post("/signup", handle6.Signup)
            r.Post("/login", handle6.Login)
            r.Post("/tokenack", handle6.TokenAck)
            r.Post("/resetpasswordchallenge", handle6.ResetPasswordChallenge)
            r.Post("/resetpassword", handle6.ResetPassword)
            r.Post("/resetpassword2", handle6.ResetPassword2)
            r.Post("/uuidcheck", handle6.UuidCheck)

            // Organisation
            r.Post("/createorga", handle6.CreateOrga)
        })
    })

    // Http/Rest API
    r.Group(func(r chi.Router) {
        r.Route("/q", func(r chi.Router) {

            // Special recursive query
            r.Group(func(r chi.Router) {
                // Those data are not secured by now, and anyone can
                // query them recursively, but as there are not sensitive
                // and set them public for now.
                //r.Use(middle6.CheckRecursiveQueryRights)
                r.Post("/sub_nodes", handle6.SubNodes)
                r.Post("/sub_members", handle6.SubMembers)
                r.Post("/top_labels", handle6.TopLabels)
                r.Post("/sub_labels", handle6.SubLabels)
                r.Post("/top_roles", handle6.TopRoles)
                r.Post("/sub_roles", handle6.SubRoles)
            })

            // Special tension query (nested filters and counts)
            r.Group(func(r chi.Router) {
                // The filtering is done directly in the query resolver as
                // doing it here required to rewrite the body, which seems difficult ?!
                //r.Use(middle6.CheckTensionQueryRights)
                r.Post("/tensions_int", handle6.TensionsInt)
                r.Post("/tensions_ext", handle6.TensionsExt)
                r.Post("/tensions_all", handle6.TensionsAll)
                r.Post("/tensions_count", handle6.TensionsCount)
            })
        })
    })

    // Graphql API
    r.Post("/api", handle6.GraphqlHandler(gqlConfig))

    // Serve static files
    web.FileServer(r, "/data/", "./data")

    // Serve Prometheus instrumentation
	if instrumentation {
        go func() {
            for {
                handle6.InstrumentationMeasures()
                time.Sleep(time.Duration(time.Second * 500))
            }
        }()
        secured := r.Group(nil)
        secured.Use(middle6.CheckBearer)
		secured.Handle("/metrics", promhttp.Handler())
	}


    // Serve Graphql Playground & introspection
    if buildMode == "DEV" {
        r.Get("/playground", handle6.PlaygroundHandler("/api"))
        r.Get("/ping", handle6.Ping)

        // Serve frontend static files
        //web.FileServer(r, "/", "./public")

        // Overwrite gql config
        gqlConfig["introspection"] = true
    }

    address := HOST + ":" + PORT
    log.Printf("Running (%s) @ http://%s", buildMode, address)
    http.ListenAndServe(address, r)
}


