package main

import (
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/yaz/kyo-repo/internal/apartments"
	"github.com/yaz/kyo-repo/internal/api"
	bcv_bucket "github.com/yaz/kyo-repo/internal/bcv-bucket"
	"github.com/yaz/kyo-repo/internal/buildings"
	"github.com/yaz/kyo-repo/internal/debts"
	"github.com/yaz/kyo-repo/internal/expenses_api"
	"github.com/yaz/kyo-repo/internal/extraCharges"
	"github.com/yaz/kyo-repo/internal/isr"
	"github.com/yaz/kyo-repo/internal/permissions"
	"github.com/yaz/kyo-repo/internal/rates"
	"github.com/yaz/kyo-repo/internal/receipts"
	"github.com/yaz/kyo-repo/internal/reserveFundsApi"
	"github.com/yaz/kyo-repo/internal/roles"
	"github.com/yaz/kyo-repo/internal/start"
	"github.com/yaz/kyo-repo/internal/telegram_api"
	"github.com/yaz/kyo-repo/internal/users"
	"github.com/yaz/kyo-repo/internal/util"
	"log"
	"net/http"
	"time"
)

func router() http.Handler {
	newRouter := mux.NewRouter()

	newRouter.Use(loggingMiddleware)
	newRouter.Use(authenticationMiddleware)

	newRouter.HandleFunc("/api/logged_in", func(w http.ResponseWriter, r *http.Request) {

		log.Printf("Logged in: %s", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	newRouter.HandleFunc("/api/logout", func(w http.ResponseWriter, r *http.Request) {

		accessTokenCookie, err := r.Cookie("access_token")
		if err != nil {
			http.Error(w, "Failed to get access token", http.StatusInternalServerError)
			return
		}

		accessTokenCookie.HttpOnly = true
		accessTokenCookie.Secure = true
		accessTokenCookie.SameSite = http.SameSiteNoneMode
		accessTokenCookie.Expires = time.Now().Add(-48 * time.Hour)
		accessTokenCookie.Path = "/"
		http.SetCookie(w, accessTokenCookie)
		url := fmt.Sprintf("%s://%s", r.URL.Scheme, r.URL.Host)
		// TODO handle non htmx requests
		w.Header().Add("HX-Redirect", url)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Logout"))
		//log.Printf("Logging out")

		//
		//http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	}).Methods("GET")

	holder := api.NewRouterHolder(newRouter)

	start.Routes(newRouter)
	rates.Routes(holder)
	bcv_bucket.Routes(holder)
	buildings.Routes(holder)
	reserveFundsApi.Routes(holder)
	apartments.Routes(holder)
	extraCharges.Routes(holder)
	receipts.Routes(holder)
	expenses_api.Routes(holder)
	debts.Routes(holder)
	users.Routes(holder)
	permissions.Routes(holder)
	roles.Routes(holder)
	isr.Routes(holder)
	telegram_api.Routes(holder)

	//newRouter.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//
	//	_, err := w.Write([]byte("Req: " + " -> " + r.URL.Path + " " + time.Now().String()))
	//	if err != nil {
	//		log.Printf("Error writing response: %v", err)
	//	}
	//})

	if util.IsDevMode() {
		return newRouter
	}

	CSRF := csrf.Protect([]byte("32-byte-long-auth-key"),
		csrf.TrustedOrigins([]string{
			"localhost:5173",
		}),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := csrf.FailureReason(r)

			log.Printf("CSRF failure: %v", err)
			http.Error(w, fmt.Sprintf("%s - %s",
				http.StatusText(http.StatusForbidden), err),
				http.StatusForbidden)
		})),
	)

	return CSRF(newRouter)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//referer, _ := url.Parse(r.Referer())

		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookieAccessToken, err := r.Cookie("access_token")
		if err != nil {
			api.RedirectToAuthServer(w, r)
			return
		}

		if cookieAccessToken == nil {
			api.RedirectToAuthServer(w, r)
			return
		}

		accessToken := cookieAccessToken.Value

		if accessToken == "" {
			api.RedirectToAuthServer(w, r)
			return
		}

		cookieRefreshToken, err := r.Cookie("refresh_token")
		var refreshToken string
		if cookieRefreshToken != nil {
			refreshToken = cookieRefreshToken.Value
		}

		newCtx, err := util.Verify(r.Context(), accessToken, refreshToken)
		if err != nil {
			log.Printf("Failed to verify token: %v", err)
			api.RedirectToAuthServer(w, r)
			return
		}

		r = r.WithContext(newCtx)

		//err = auth_client.Verify(accessToken, refreshToken)
		//if err != nil {
		//	log.Printf("Failed to verify token: %v", err)
		//	api.RedirectToAuthServer(w, r)
		//	return
		//}

		next.ServeHTTP(w, r)
	})
}

func main() {
	//log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetFlags(log.LstdFlags | log.Llongfile)
	lambda.Start(httpadapter.NewV2(router()).ProxyWithContext)
}
