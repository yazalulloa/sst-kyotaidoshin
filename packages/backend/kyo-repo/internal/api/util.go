package api

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/a-h/templ"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-jet/jet/v2/sqlite"
	"github.com/gorilla/mux"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"github.com/yaz/kyo-repo/internal/aws_h"
	"github.com/yaz/kyo-repo/internal/db"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	. "github.com/yaz/kyo-repo/internal/db/gen/table"
	"github.com/yaz/kyo-repo/internal/util"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"
)

type HttpMethod string

const (
	GET     HttpMethod = "GET"
	POST    HttpMethod = "POST"
	PUT     HttpMethod = "PUT"
	DELETE  HttpMethod = "DELETE"
	PATCH   HttpMethod = "PATCH"
	OPTIONS HttpMethod = "OPTIONS"
)

const USER_PERM_PREFIX = "user-id-perm-"

func (receiver HttpMethod) Name() string {
	return string(receiver)
}

func BuildUploadForm(r *http.Request, filePrefix string) (templ.Component, error) {

	params, err := util.GetUploadFormParams(r, filePrefix)
	if err != nil {
		return nil, err
	}
	return UploadFormView(*params), nil
}

func ProcessBackup(ctx context.Context, bucket, key, etag *string,
	processJson func(*json.Decoder) (int64, error)) (int64, error) {
	s3Client, err := aws_h.GetS3Client(ctx)
	if err != nil {
		log.Printf("Error getting s3 client: %s", err)
		return 0, err
	}

	log.Printf("Getting object from bucket %s key %s", *bucket, *key)
	outPut, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket:       bucket,
		Key:          key,
		IfMatch:      etag,
		ChecksumMode: types.ChecksumModeEnabled,
	})

	if err != nil {
		log.Printf("Error getting object from bucket %s key %s: %s", *bucket, *key, err)
		return 0, err
	}

	deleteObj := func() {
		log.Printf("Deleting object from bucket %s key %s", *bucket, *key)
		_, err = s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: bucket,
			Key:    key,
		})

		if err != nil {
			log.Printf("Error deleting object from bucket %s key %s: %s", *bucket, *key, err)
		}
	}

	defer deleteObj()

	closeBody := func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Error closing: ", err)
			return
		}
	}

	defer closeBody(outPut.Body)

	gzipReader, err := gzip.NewReader(outPut.Body)
	if err != nil {
		log.Printf("Error creating gzip reader: %s", err)
		return 0, err
	}

	defer closeBody(gzipReader)

	decoder := json.NewDecoder(gzipReader)

	var inserted int64 = 0

	for decoder.More() {
		rowsAffected, err := processJson(decoder)
		if err != nil {
			return 0, err
		}
		inserted += rowsAffected
	}

	return inserted, nil
}

func ProcessUploadBackup(r *http.Request, redirecUrl string, processJson func(*json.Decoder) (int64, error)) (templ.Component, error) {
	key := r.FormValue("key")
	if key == "" {
		log.Printf("key is empty")
		return nil, errors.New("BAD REQUEST")
	}

	bucket, err := util.GetReceiptsBucket()
	if err != nil {
		log.Printf("Error getting bucket Name: %s", err)
		return nil, err
	}

	_, err = ProcessBackup(r.Context(), &bucket, &key, nil, processJson)
	if err != nil {
		return nil, err
	}

	return AnchorClickInitView(redirecUrl), nil
}

func NewRouterHolder(router *mux.Router) *RouterHolder {
	return &RouterHolder{router: router}
}

type RouterHolder struct {
	router *mux.Router
}

func (holder *RouterHolder) AddRoute(params RouteParams) {
	holder.router.HandleFunc(params.Path, params.routeHandler()).Methods(params.Method.Name())
}

func (holder *RouterHolder) GET(path string, handler func(http.ResponseWriter, *http.Request), perms ...PERM) {
	holder.AddRoute(RouteParams{
		Method:  GET,
		Path:    path,
		Handler: handler,
		Perms:   perms,
	})
}

func (holder *RouterHolder) POST(path string, handler func(http.ResponseWriter, *http.Request), recaptchaAction RecaptchaAction, perms ...PERM) {
	holder.AddRoute(RouteParams{
		Method:          POST,
		Path:            path,
		Handler:         handler,
		Perms:           perms,
		RecaptchaAction: recaptchaAction,
	})
}

func (holder *RouterHolder) PUT(path string, handler func(http.ResponseWriter, *http.Request), recaptchaAction RecaptchaAction, perms ...PERM) {
	holder.AddRoute(RouteParams{
		Method:          PUT,
		Path:            path,
		Handler:         handler,
		Perms:           perms,
		RecaptchaAction: recaptchaAction,
	})
}

func (holder *RouterHolder) DELETE(path string, handler func(http.ResponseWriter, *http.Request), recaptchaAction RecaptchaAction, perms ...PERM) {
	holder.AddRoute(RouteParams{
		Method:          DELETE,
		Path:            path,
		Handler:         handler,
		Perms:           perms,
		RecaptchaAction: recaptchaAction,
	})
}

type RouteParams struct {
	Method          HttpMethod
	Path            string
	Handler         func(http.ResponseWriter, *http.Request)
	Perms           []PERM
	RecaptchaAction RecaptchaAction
}

func (rec RouteParams) routeHandler() func(http.ResponseWriter,
	*http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		var wg sync.WaitGroup
		wg.Add(2)
		errorChan := make(chan error, 2)

		go func() {
			defer wg.Done()
			//timestamp := time.Now().UnixMilli()
			//defer func() { log.Printf("CheckPerms took %d ms", time.Now().UnixMilli()-timestamp) }()

			userPerms, err := checkPerms(r, rec.Perms)
			if err != nil {
				errorChan <- err
				return
			}

			newCtx := r.Context()
			for _, p := range userPerms {
				newCtx = context.WithValue(newCtx, USER_PERM_PREFIX+p, p)
			}
			r = r.WithContext(newCtx)
		}()

		go func() {

			defer wg.Done()

			if rec.RecaptchaAction == "" {
				return
			}

			enableCaptcha, _ := resource.Get("EnableCaptcha", "value")
			if enableCaptcha != nil && enableCaptcha.(string) == "false" {
				log.Println("Captcha is disabled, skipping recaptcha check")
				return
			}

			timestamp := time.Now().UnixMilli()
			defer func() { log.Printf("CheckRecaptcha took %d ms", time.Now().UnixMilli()-timestamp) }()

			err := rec.checkCaptcha(r)
			if err != nil {
				errorChan <- err
				return
			}
		}()

		//timestamp := time.Now().UnixMilli()
		wg.Wait()
		close(errorChan)

		err := util.HasErrors(errorChan)

		if err != nil {
			log.Printf("Error: %s", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		//log.Printf("Middleware took %d ms", time.Now().UnixMilli()-timestamp)
		rec.Handler(w, r)
	}
}

func getUserPermissions(ctx context.Context, userId string) ([]string, error) {
	permsMap := GetPermsMap()

	cachedPerms := permsMap.Get(userId)
	if cachedPerms != "" {
		return strings.Split(cachedPerms, ","), nil
	}

	stmt := Permissions.SELECT(Permissions.Name).
		FROM(
			Users.INNER_JOIN(UserRoles, Users.ID.EQ(UserRoles.UserID)).
				INNER_JOIN(Roles, UserRoles.RoleID.EQ(Roles.ID)).
				INNER_JOIN(RolePermissions, Roles.ID.EQ(RolePermissions.RoleID)).
				INNER_JOIN(Permissions, RolePermissions.PermissionID.EQ(Permissions.ID)),
		).
		WHERE(Users.ID.EQ(sqlite.String(userId)))

	var dest []model.Permissions

	err := stmt.QueryContext(ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	permissions := make([]string, len(dest))
	for i, p := range dest {
		permissions[i] = p.Name
	}

	if len(permissions) > 0 {
		permsMap.Put(userId, strings.Join(permissions, ","))
	}

	return permissions, nil

}

var permsMap *TTLMap
var permsMapOnce sync.Once

func GetPermsMap() *TTLMap {

	permsMapOnce.Do(func() {
		permsMap = New(5, time.Second*60)
	})
	return permsMap
}

func checkPerms(r *http.Request, perm []PERM) ([]string, error) {

	userId := r.Context().Value(util.USER_ID)
	if userId == nil || userId.(string) == "" {
		return nil, errors.New("invalid userId")
	}

	userPerms, err := getUserPermissions(r.Context(), userId.(string))
	if err != nil {
		return nil, fmt.Errorf("error getting user permissions: %s", err)
	}

	hasPerm := len(perm) <= 0
	for _, p := range perm {
		if slices.Contains(userPerms, p.Name()) {
			hasPerm = true
			break
		}
	}

	if !hasPerm {
		return nil, errors.New("perms check failed")
	}

	return userPerms, nil
}

func (rec RouteParams) checkCaptcha(r *http.Request) error {

	token := r.Header.Get("X-Recaptcha-Token")

	if token == "" {
		return errors.New("recaptcha token is empty")
	}

	secret, err := resource.Get("CaptchaSecretKey", "value")
	if err != nil {
		return fmt.Errorf("error getting secret from resource: %s", err)
	}

	err = CheckRecaptcha(rec.RecaptchaAction, secret.(string), token)
	if err != nil {
		return fmt.Errorf("error checking recaptcha: %s", err)
	}

	return nil
}

func HasPerms(ctx context.Context, perms ...PERM) bool {
	if len(perms) == 0 {
		return true
	}

	for _, p := range perms {
		userPerm := ctx.Value(USER_PERM_PREFIX + p.Name())
		if userPerm != nil && userPerm.(string) == p.Name() {
			return true
		}
	}

	return false
}

func RedirectToAuthServer(w http.ResponseWriter, r *http.Request) {
	url := os.Getenv("AUTH_SERVER_URL") + "/api/authorize"
	//url := fmt.Sprintf("%s://%s", r.URL.Scheme, r.URL.Host)
	//url :=  "/api/authorize"

	log.Printf("Redirecting to auth server: %s", url)
	w.Header().Add("HX-Redirect", url)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Unauthorized"))
}

func RemoveAuthCookies(w http.ResponseWriter, r *http.Request) {
	accessTokenCookie, err := r.Cookie("access_token")
	if err == nil && accessTokenCookie != nil {
		cookie := CookieToRemove("access_token")
		http.SetCookie(w, cookie)
	}

	refreshTokenCookie, err := r.Cookie("refresh_token")
	if err == nil && refreshTokenCookie != nil {
		cookie := CookieToRemove("refresh_token")
		http.SetCookie(w, cookie)
	}

}

func CookieToRemove(name string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}
}
