package main

import (
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"kyotaidoshin/apartments"
	bcv_bucket "kyotaidoshin/bcv-bucket"
	"kyotaidoshin/buildings"
	"kyotaidoshin/debts"
	"kyotaidoshin/expenses_api"
	"kyotaidoshin/extraCharges"
	"kyotaidoshin/rates"
	"kyotaidoshin/receipts"
	"kyotaidoshin/reserveFundsApi"
	"kyotaidoshin/start"
	"kyotaidoshin/users"
	"kyotaidoshin/util"
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
		log.Printf("Logging out")

		//
		//http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	}).Methods("GET")

	start.Routes(newRouter)
	rates.Routes(newRouter)
	bcv_bucket.Routes(newRouter)
	buildings.Routes(newRouter)
	reserveFundsApi.Routes(newRouter)
	apartments.Routes(newRouter)
	extraCharges.Routes(newRouter)
	receipts.Routes(newRouter)
	expenses_api.Routes(newRouter)
	debts.Routes(newRouter)
	users.Routes(newRouter)

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

func redirectToAuthServer(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("%s://%s", r.URL.Scheme, r.URL.Host)
	w.Header().Add("HX-Redirect", url)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Unauthorized"))
}

func authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookieAccessToken, err := r.Cookie("access_token")
		if err != nil {
			redirectToAuthServer(w, r)
			return
		}

		if cookieAccessToken == nil {
			redirectToAuthServer(w, r)
			return
		}

		accessToken := cookieAccessToken.Value

		if accessToken == "" {
			redirectToAuthServer(w, r)
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
			redirectToAuthServer(w, r)
			return
		}

		r = r.WithContext(newCtx)

		//err = auth_client.Verify(accessToken, refreshToken)
		//if err != nil {
		//	log.Printf("Failed to verify token: %v", err)
		//	redirectToAuthServer(w, r)
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
