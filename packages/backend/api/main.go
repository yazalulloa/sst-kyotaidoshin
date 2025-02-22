package main

import (
	"database/sql"
	"db"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"kyotaidoshin/apartments"
	"kyotaidoshin/api"
	bcv_bucket "kyotaidoshin/bcv-bucket"
	"kyotaidoshin/buildings"
	"kyotaidoshin/extraCharges"
	"kyotaidoshin/rates"
	"kyotaidoshin/receipts"
	"kyotaidoshin/reserveFunds"
	"log"
	"net/http"
	"strings"
	"time"
)

func handler() (string, error) {
	bucket, err := resource.Get("bcv-bucket", "name")
	if err != nil {
		log.Print("Error getting bucket name")
		return "", err
	}

	var versionStmt *sql.Stmt
	var version string
	err = db.MakeStmt(versionStmt, "SELECT sqlite_version() AS version").QueryRow().Scan(&version)
	if err != nil {
		log.Print("Error querying db")
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("hello ")
	sb.WriteString(bucket.(string))
	sb.WriteString(" ")
	sb.WriteString(version)
	sb.WriteString(" ")

	return sb.String(), nil
}

func router() http.Handler {
	newRouter := mux.NewRouter()

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

	rates.Routes(newRouter)
	bcv_bucket.Routes(newRouter)
	buildings.Routes(newRouter)
	reserveFunds.Routes(newRouter)
	apartments.Routes(newRouter)
	extraCharges.Routes(newRouter)
	receipts.Routes(newRouter)

	newRouter.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		msg, err := handler()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = w.Write([]byte(msg + " -> " + r.URL.Path + " " + time.Now().String()))
	})

	newRouter.Use(loggingMiddleware)

	if api.IsDevMode() {
		return newRouter
	}

	return CSRF(newRouter)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		//referer, _ := url.Parse(r.Referer())
		//log.Printf("%s %s %s\n%v\n", r.Method, r.RequestURI, referer.Host, r.Header)
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

func main() {
	//log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetFlags(log.LstdFlags | log.Llongfile)
	lambda.Start(httpadapter.NewV2(router()).ProxyWithContext)
}
