package bcv_bucket

import (
	"bcv/bcv"
	"github.com/a-h/templ"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"kyotaidoshin/api"
	"kyotaidoshin/util"
	"log"
	"net/http"
	"process-bcv-file/file"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const _PATH = "/api/bcv-bucket"
const _SEARCH = _PATH + "/search"

func Routes(server *mux.Router) {
	server.Handle(_PATH, templ.Handler(Init())).Methods("GET")

	server.HandleFunc(_SEARCH, search).Methods("GET")
	server.HandleFunc(_PATH+"/{id}", bcvBucketDelete).Methods("DELETE")
	server.HandleFunc(_PATH+"/process/{id}", process).Methods("POST")
}

func search(w http.ResponseWriter, r *http.Request) {
	bucketName, err := bcv.GetBcvBucket()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s3Client, err := bcv.GetS3Client()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s3List, err := s3Client.ListObjectsV2(r.Context(), &s3.ListObjectsV2Input{Bucket: aws.String(bucketName)})
	if err != nil {
		log.Printf("Error getting objects from bucket %s: %s", bucketName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var wg sync.WaitGroup
	var once sync.Once
	handleErr := func(e error) {
		if e != nil {
			once.Do(func() {
				err = e
			})
		}
	}

	wg.Add(len(s3List.Contents))

	results := make([]Item, len(s3List.Contents))
	for i, item := range s3List.Contents {

		go func(item types.Object) {
			defer wg.Done()
			obj, err := s3Client.HeadObject(r.Context(), &s3.HeadObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(*item.Key),
			})

			if err != nil {
				handleErr(err)
				return
			}

			url := obj.Metadata["url"]
			processed := obj.Metadata["processed"]
			processedBool := false
			if processed != "" {
				processedBool, _ = strconv.ParseBool(processed)
			}

			results[i] = Item{
				Item: S3File{
					Name:          *item.Key,
					Size:          *item.Size,
					SizeFormatted: util.FormatBytes(*item.Size),
					Etag:          *item.ETag,
					LastModified:  (*item.LastModified).UnixMilli(),
					Url:           url,
					Processed:     processedBool,
				},
				Key:    *api.Base64Encode(*item.Key),
				CardId: "bcv-buckets-" + uuid.NewString(),
			}
		}(item)

	}
	wg.Wait()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sort.Slice(results, func(i, j int) bool {
		lhs, _ := strconv.Atoi(strings.Split(results[i].Item.Name, "=")[1])
		rhs, _ := strconv.Atoi(strings.Split(results[j].Item.Name, "=")[1])
		return lhs > rhs
	})

	response := TableResponse{
		TotalCount: len(s3List.Contents),
		Results:    results,
	}

	err = Search(response).Render(r.Context(), w)
	if err != nil {
		log.Printf("Error rendering table view: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func bcvBucketDelete(w http.ResponseWriter, r *http.Request) {
	bucketName, err := bcv.GetBcvBucket()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]
	var str string
	err = api.Decode(id, &str)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s3Client, err := bcv.GetS3Client()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = s3Client.DeleteObject(r.Context(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(str),
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s3List, err := s3Client.ListObjectsV2(r.Context(), &s3.ListObjectsV2Input{Bucket: aws.String(bucketName)})
	if err != nil {
		log.Printf("Error getting objects from bucket %s: %s", bucketName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = CountersView(len(s3List.Contents)).Render(r.Context(), w)
	if err != nil {
		log.Printf("Error rendering view: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func process(w http.ResponseWriter, r *http.Request) {
	bucketName, err := bcv.GetBcvBucket()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	var str string
	err = api.Decode(id, &str)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Handling PROCESS %s", str)

	err = file.ParseFile(r.Context(), bucketName, str)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Error(w, "", http.StatusNoContent)

}

type S3File struct {
	Name          string
	Size          int64
	SizeFormatted string
	Etag          string
	LastModified  int64
	Url           string
	Processed     bool
}

type Item struct {
	Item   S3File
	CardId string
	Key    string
}

type TableResponse struct {
	TotalCount int
	Results    []Item
}
