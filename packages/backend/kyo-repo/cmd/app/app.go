package main

import (
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/posthog/posthog-go"
	"github.com/yaz/kyo-repo/internal/apartments"
	"github.com/yaz/kyo-repo/internal/api"
	"github.com/yaz/kyo-repo/internal/api/compress"
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
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"unicode/utf8"
)

func router() http.Handler {
	newRouter := mux.NewRouter()

	newRouter.Use(mainApiMetricMiddleware)
	//newRouter.Use(loggingMiddleware)
	newRouter.Use(authenticationMiddleware)
	newRouter.Use(compress.Middleware)

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
		accessTokenCookie.SameSite = http.SameSiteStrictMode
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

	//return handlers.CombinedLoggingHandler(os.Stdout, newRouter)
	return handlers.CustomLoggingHandler(os.Stdout, newRouter, writeCombinedLog)
	//return handlers.CompressHandler(newRouter)

	//newRouter.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//
	//	_, err := w.Write([]byte("Req: " + " -> " + r.URL.Path + " " + time.Now().String()))
	//	if err != nil {
	//		log.Printf("Error writing response: %v", err)
	//	}
	//})

	//if util.IsDevMode() {
	//	return handlers.CompressHandler(newRouter)
	//}
	//
	//CSRF := csrf.Protect([]byte("32-byte-long-auth-key"),
	//	csrf.TrustedOrigins([]string{
	//		"localhost:5173",
	//	}),
	//	csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//		err := csrf.FailureReason(r)
	//
	//		log.Printf("CSRF failure: %v", err)
	//		http.Error(w, fmt.Sprintf("%s - %s",
	//			http.StatusText(http.StatusForbidden), err),
	//			http.StatusForbidden)
	//	})),
	//)
	//
	//return CSRF(newRouter)
}

const lowerhex = "0123456789abcdef"

func appendQuoted(buf []byte, s string) []byte {
	var runeTmp [utf8.UTFMax]byte
	for width := 0; len(s) > 0; s = s[width:] { //nolint: wastedassign //TODO: why width starts from 0and reassigned as 1
		r := rune(s[0])
		width = 1
		if r >= utf8.RuneSelf {
			r, width = utf8.DecodeRuneInString(s)
		}
		if width == 1 && r == utf8.RuneError {
			buf = append(buf, `\x`...)
			buf = append(buf, lowerhex[s[0]>>4])
			buf = append(buf, lowerhex[s[0]&0xF])
			continue
		}
		if r == rune('"') || r == '\\' { // always backslashed
			buf = append(buf, '\\')
			buf = append(buf, byte(r))
			continue
		}
		if strconv.IsPrint(r) {
			n := utf8.EncodeRune(runeTmp[:], r)
			buf = append(buf, runeTmp[:n]...)
			continue
		}
		switch r {
		case '\a':
			buf = append(buf, `\a`...)
		case '\b':
			buf = append(buf, `\b`...)
		case '\f':
			buf = append(buf, `\f`...)
		case '\n':
			buf = append(buf, `\n`...)
		case '\r':
			buf = append(buf, `\r`...)
		case '\t':
			buf = append(buf, `\t`...)
		case '\v':
			buf = append(buf, `\v`...)
		default:
			switch {
			case r < ' ':
				buf = append(buf, `\x`...)
				buf = append(buf, lowerhex[s[0]>>4])
				buf = append(buf, lowerhex[s[0]&0xF])
			case r > utf8.MaxRune:
				r = 0xFFFD
				fallthrough
			case r < 0x10000:
				buf = append(buf, `\u`...)
				for s := 12; s >= 0; s -= 4 {
					buf = append(buf, lowerhex[r>>uint(s)&0xF])
				}
			default:
				buf = append(buf, `\U`...)
				for s := 28; s >= 0; s -= 4 {
					buf = append(buf, lowerhex[r>>uint(s)&0xF])
				}
			}
		}
	}
	return buf
}

func buildCommonLogLine(req *http.Request, url url.URL, ts time.Time, status int, size int) []byte {
	username := "-"
	if url.User != nil {
		if name := url.User.Username(); name != "" {
			username = name
		}
	}

	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		host = req.RemoteAddr
	}

	uri := req.RequestURI

	// Requests using the CONNECT method over HTTP/2.0 must use
	// the authority field (aka r.Host) to identify the target.
	// Refer: https://httpwg.github.io/specs/rfc7540.html#CONNECT
	if req.ProtoMajor == 2 && req.Method == "CONNECT" {
		uri = req.Host
	}
	if uri == "" {
		uri = url.RequestURI()
	}

	buf := make([]byte, 0, 3*(len(host)+len(username)+len(req.Method)+len(uri)+len(req.Proto)+50)/2)
	buf = append(buf, host...)
	buf = append(buf, " - "...)
	buf = append(buf, username...)
	buf = append(buf, " ["...)
	buf = append(buf, ts.Format("02/Jan/2006:15:04:05 -0700")...)
	buf = append(buf, `] "`...)
	buf = append(buf, req.Method...)
	buf = append(buf, " "...)
	buf = appendQuoted(buf, uri)
	buf = append(buf, " "...)
	buf = append(buf, req.Proto...)
	buf = append(buf, `" `...)
	buf = append(buf, strconv.Itoa(status)...)
	buf = append(buf, " "...)
	buf = append(buf, strconv.Itoa(size)...)
	return buf
}

func writeCombinedLog(writer io.Writer, params handlers.LogFormatterParams) {
	buf := buildCommonLogLine(params.Request, params.URL, params.TimeStamp, params.StatusCode, params.Size)
	buf = append(buf, fmt.Sprintf(" %dms", time.Since(params.TimeStamp).Milliseconds())...)
	buf = append(buf, ` "`...)
	buf = appendQuoted(buf, params.Request.Referer())
	buf = append(buf, `" "`...)
	buf = appendQuoted(buf, params.Request.UserAgent())
	buf = append(buf, '"', '\n')
	_, _ = writer.Write(buf)
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

		//log.Printf("Access token: %s", accessToken)
		//log.Printf("Refresh token: %s", refreshToken)

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

func mainApiMetricMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		defer func() {
			duration := time.Since(startTime)
			//log.Printf("Request %s %s took %s", r.Method, r.URL.Path, duration)
			//util.RecordApiMetric(r.Method, r.URL.Path, duration)

			client, err := util.GetPosthogClient()
			if err != nil {
				log.Printf("Failed to get Posthog client: %v", err)
				return
			}

			if client != nil {
				c := *client

				err = c.Enqueue(posthog.Capture{
					DistinctId: util.UuidV7(),
					Event:      "api_time",
					Properties: posthog.NewProperties().
						Set("path", r.URL.Path).
						Set("method", r.Method).
						Set("duration", duration.Milliseconds()), // Duration in seconds
				})

				if err != nil {
					log.Printf("Failed asd to enqueue Posthog event: %v path %s", err, r.URL.Path)
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// Static initialization
// SIGTERM Handler: https://docs.aws.amazon.com/lambda/latest/operatorguide/static-initialization.html
func init() {
	// Create a chan to receive os signal
	var c = make(chan os.Signal)
	// Listening for os signals that can be handled,reference: https://docs.aws.amazon.com/lambda/latest/dg/runtimes-extensions-api.html
	// Termination Signals: https://www.gnu.org/software/libc/manual/html_node/Termination-Signals.html
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGHUP)
	// do something when os signal received
	go func() {
		for s := range c {
			switch s {
			// if lambda runtime received SIGTERM signal,perform actual clean up work here.
			case syscall.SIGTERM:
				fmt.Println("[runtime] SIGTERM received")
				fmt.Println("[runtime] Graceful shutdown in progress ...")
				fmt.Println("[runtime] Graceful shutdown completed")
				os.Exit(0)
				// else if lambda runtime received other signal
			default:
				fmt.Println("[runtime] Other signal received")
				fmt.Println("[runtime] Graceful shutdown in progress ...")
				fmt.Println("[runtime] Graceful shutdown completed")
				os.Exit(0)
			}
		}
	}()
}

func main() {
	//log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetFlags(log.LstdFlags | log.Llongfile)
	lambda.StartWithOptions(httpadapter.NewV2(router()).ProxyWithContext)
}
