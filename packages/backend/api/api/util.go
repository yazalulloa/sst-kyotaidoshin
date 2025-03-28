package api

import (
	"aws_h"
	"compress/gzip"
	"context"
	"db"
	"db/gen/model"
	. "db/gen/table"
	"encoding/json"
	"errors"
	"github.com/a-h/templ"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-jet/jet/v2/sqlite"
	"github.com/gorilla/mux"
	"io"
	"kyotaidoshin/util"
	"log"
	"net/http"
	"slices"
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

func (holder *RouterHolder) AddRoute(method HttpMethod, path string, handler func(http.ResponseWriter, *http.Request), perms ...PERM) {
	holder.router.HandleFunc(path, checkPerms(handler, perms)).Methods(method.Name())
}

func (holder *RouterHolder) GET(path string, handler func(http.ResponseWriter, *http.Request), perms ...PERM) {
	holder.AddRoute(GET, path, handler, perms...)
}

func (holder *RouterHolder) POST(path string, handler func(http.ResponseWriter, *http.Request), perms ...PERM) {
	holder.AddRoute(POST, path, handler, perms...)
}

func (holder *RouterHolder) PUT(path string, handler func(http.ResponseWriter, *http.Request), perms ...PERM) {
	holder.AddRoute(PUT, path, handler, perms...)
}

func (holder *RouterHolder) DELETE(path string, handler func(http.ResponseWriter, *http.Request), perms ...PERM) {
	holder.AddRoute(DELETE, path, handler, perms...)
}

func getUserPermissions(ctx context.Context, userId string) ([]string, error) {

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

	return permissions, nil

}

func userHasPermissions(ctx context.Context, userId string, permission []string) (bool, error) {
	perms := make([]sqlite.Expression, len(permission))
	for i, p := range permission {
		perms[i] = sqlite.String(p)
	}

	stmt := UserRoles.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).
		FROM(
			UserRoles.
				INNER_JOIN(RolePermissions, UserRoles.RoleID.EQ(RolePermissions.RoleID)).
				INNER_JOIN(Permissions, RolePermissions.PermissionID.EQ(Permissions.ID)),
		).WHERE(UserRoles.UserID.EQ(sqlite.String(userId)).AND(Permissions.Name.IN(perms...)))

	var dest struct {
		Count int64
	}

	err := stmt.QueryContext(ctx, db.GetDB().DB, &dest)
	if err != nil {
		return false, err
	}

	return dest.Count > 0, nil
}

func checkPerms(next func(http.ResponseWriter, *http.Request), perm []PERM) func(http.ResponseWriter,
	*http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if false {
			next(w, r)
			return
		}

		if len(perm) == 0 {
			next(w, r)
			return
		}

		userId := r.Context().Value(util.USER_ID)
		if userId == nil || userId.(string) == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userPerms, err := getUserPermissions(r.Context(), userId.(string))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		hasPerm := false
		for _, p := range perm {
			if slices.Contains(userPerms, p.Name()) {
				hasPerm = true
				break
			}
		}

		if !hasPerm {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		newCtx := r.Context()
		for _, p := range userPerms {
			newCtx = context.WithValue(newCtx, USER_PERM_PREFIX+p, p)
		}
		r = r.WithContext(newCtx)

		//permissions := make([]string, len(perm))
		//for i, p := range perm {
		//	permissions[i] = p.Name()
		//}
		//
		//
		//hasPerm, err := userHasPermissions(r.Context(), userId.(string), permissions)
		//if err != nil {
		//	http.Error(w, err.Error(), http.StatusInternalServerError)
		//	return
		//}

		next(w, r)
	}
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
