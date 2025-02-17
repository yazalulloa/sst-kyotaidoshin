package apartments

import (
	"aws_h"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sst/sst/v3/sdk/golang/resource"
	"io"
	"kyotaidoshin/api"
	"log"
	"net/http"
	"strings"
)

const _PATH = "/api/apartments"
const _SEARCH = _PATH + "/search"
const _UPLOAD_BACKUP_FORM = _PATH + "/uploadBackupForm"
const _UPLOAD_BACKUP = _PATH + "/upload/backup"

func Routes(server *mux.Router) {

	server.HandleFunc(_SEARCH, search).Methods("GET")
	server.HandleFunc(_PATH+"/{key}", aptDelete).Methods("DELETE")
	server.HandleFunc(_UPLOAD_BACKUP_FORM, getUploadBackupForm).Methods("GET")
	server.HandleFunc(_UPLOAD_BACKUP, uploadBackup).Methods("GET")
	//server.HandleFunc(_PATH+"/upload/backup", uploadBackupUrl).Methods("GET")
	//server.HandleFunc(_PATH, aptPut).Methods("PUT")
	//server.HandleFunc(_PATH+"/formData", formData).Methods("GET")
}

func search(w http.ResponseWriter, r *http.Request) {
	nextPage := api.GetQueryParamAsString(r, "next_page")
	var keys Keys
	if nextPage != "" {
		err := api.Decode(nextPage, &keys)

		if err != nil {
			log.Printf("failed to decode nextPage: %v", err)
			http.Error(w, "Bad Request nextPage", http.StatusBadRequest)
			return
		}
	}
	query := r.URL.Query()
	buildings := query["building_input"]
	requestQuery := RequestQuery{
		lastBuildingId: keys.BuildingId,
		lastNumber:     keys.Number,
		q:              api.GetQueryParamAsString(r, "apt_search_input"),
		buildings:      buildings,
		Limit:          31,
	}

	response, err := getTableResponse(requestQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	results := response.Results
	var nextPageUrl string
	if len(results) == requestQuery.Limit {
		results = results[:len(results)-1]
		last := results[len(results)-1]
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf(_SEARCH+"?next_page=%s", last.Key))

		if requestQuery.q != "" {
			sb.WriteString(fmt.Sprintf("&apt_search_input=%s", requestQuery.q))
		}

		if len(requestQuery.buildings) > 0 {
			for _, building := range requestQuery.buildings {
				sb.WriteString(fmt.Sprintf("&building_input=%s", building))
			}
		}

		nextPageUrl = sb.String()
	}

	response.NextPageUrl = nextPageUrl
	response.Results = results

	err = Search(response).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func aptDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyStr := vars["key"]
	if keyStr == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var keys Keys
	err := api.Decode(keyStr, &keys)
	if err != nil {
		log.Printf("failed to decode nextPage: %v", err)
		http.Error(w, "Bad Request nextPage", http.StatusBadRequest)
		return
	}

	counters, err := deleteAndReturnCounters(keys)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = CountersView(*counters).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func getUploadBackupForm(w http.ResponseWriter, r *http.Request) {

	bucketName, err := resource.Get("UploadBackup", "name")
	if err != nil {
		log.Printf("Error getting bucket name: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	functionUrl, err := resource.Get("ApiFunction", "url")
	if err != nil {
		log.Printf("Error getting function url: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	redirectUrl := fmt.Sprintf("%s%s", functionUrl.(string), _UPLOAD_BACKUP[1:])
	metaUuid := uuid.New().String()

	conditions := []interface{}{
		map[string]string{"success_action_redirect": redirectUrl},
		//[]interface{}{"starts-with", "$Content-Type", "application/gzip"},
		map[string]string{"x-amz-meta-uuid": metaUuid},
		//[]interface{}{"starts-with", "$x-amz-meta-tag", ""},
		[]interface{}{"content-length-range", 1, 2048576},
	}

	optionFn := func(options *s3.PresignPostOptions) {
		//options.Expires = time.Hour
		options.Conditions = conditions
	}

	presignedPostRequest, err := aws_h.PresignPostObject(r.Context(), bucketName.(string), fmt.Sprintf("apartments_%s", uuid.New().String()), optionFn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	presignedPostRequest.Values["success_action_redirect"] = redirectUrl
	presignedPostRequest.Values["x-amz-meta-uuid"] = metaUuid

	err = uploadBackupForm(presignedPostRequest.URL, presignedPostRequest.Values).Render(r.Context(), w)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func uploadBackup(w http.ResponseWriter, r *http.Request) {

	bucket := api.GetQueryParamAsString(r, "bucket")
	key := api.GetQueryParamAsString(r, "key")
	etag := api.GetQueryParamAsString(r, "etag")

	if bucket == "" || key == "" || etag == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	s3Client, err := aws_h.GetS3Client(r.Context())
	if err != nil {
		log.Printf("Error getting s3 client: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Getting object from bucket %s key %s", bucket, key)
	outPut, err := s3Client.GetObject(r.Context(), &s3.GetObjectInput{
		Bucket:       &bucket,
		Key:          &key,
		IfMatch:      &etag,
		ChecksumMode: types.ChecksumModeEnabled,
	})

	if err != nil {
		log.Printf("Error getting object from bucket %s key %s: %s", bucket, key, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer closeBody(gzipReader)

	decoder := json.NewDecoder(gzipReader)

	var inserted int64 = 0
	//apts := make([]ApartmentDto, 50)

	for decoder.More() { // Loop through JSON objects in the stream
		var dto []ApartmentDto
		err := decoder.Decode(&dto)
		if err != nil {
			log.Printf("Error decoding json: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rowsAffected, err := insertDtos(dto)
		if err != nil {
			log.Printf("Error inserting dtos: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		inserted += rowsAffected
	}

	_, err = s3Client.DeleteObject(r.Context(), &s3.DeleteObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})

	if err != nil {
		log.Printf("Error deleting object from bucket %s key %s: %s", bucket, key, err)
	}

	err = uploadBackupResponse(inserted).Render(r.Context(), w)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
