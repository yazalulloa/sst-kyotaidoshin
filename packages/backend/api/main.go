package main

import (
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"kyotaidoshin/apartments"
	"kyotaidoshin/api"
	bcv_bucket "kyotaidoshin/bcv-bucket"
	"kyotaidoshin/buildings"
	"kyotaidoshin/debts"
	"kyotaidoshin/expenses_api"
	"kyotaidoshin/extraCharges"
	"kyotaidoshin/rates"
	"kyotaidoshin/receipts"
	"kyotaidoshin/reserveFundsApi"
	"log"
	"net/http"
)

func router() http.Handler {
	newRouter := mux.NewRouter()

	rates.Routes(newRouter)
	bcv_bucket.Routes(newRouter)
	buildings.Routes(newRouter)
	reserveFundsApi.Routes(newRouter)
	apartments.Routes(newRouter)
	extraCharges.Routes(newRouter)
	receipts.Routes(newRouter)
	expenses_api.Routes(newRouter)
	debts.Routes(newRouter)

	//newRouter.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//
	//	_, err := w.Write([]byte("Req: " + " -> " + r.URL.Path + " " + time.Now().String()))
	//	if err != nil {
	//		log.Printf("Error writing response: %v", err)
	//	}
	//})

	newRouter.Use(loggingMiddleware)

	if api.IsDevMode() {
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
		log.Printf("%s %s %s", r.Method, r.RequestURI, r.Referer())
		next.ServeHTTP(w, r)
	})
}

func main() {
	//log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetFlags(log.LstdFlags | log.Llongfile)
	lambda.Start(httpadapter.NewV2(router()).ProxyWithContext)
}
