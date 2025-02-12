package bcv_bucket

import (
	"bcv/bcv"
	"context"
	"fmt"
	"github.com/a-h/templ"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
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
	server.HandleFunc(_PATH+"/process-all", processAll).Methods("GET")
	server.HandleFunc(_PATH+"/look-up", lookUp).Methods("GET")
}

func search(w http.ResponseWriter, r *http.Request) {
	bucketName, err := bcv.GetBcvBucket()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s3Client, err := bcv.GetS3Client(r.Context())
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
			processed := obj.Metadata[bcv.MetadataProcessedKey]
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

	s3Client, err := bcv.GetS3Client(r.Context())
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

	ProcessAll := true
	err = file.ParseFile(file.ParsingParams{
		Ctx:        r.Context(),
		Bucket:     bucketName,
		Key:        str,
		ProcessAll: &ProcessAll,
	})
	//err = invokeParsingFunction(r.Context(), bucketName, str)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)

}

func invokeParsingFunction(ctx context.Context, bucket string, key string) error {
	lambdaName, err := bcv.GetParsingLambda()
	if err != nil {
		return err
	}
	lambdaClient, err := bcv.GetLambdaClient(ctx)
	if err != nil {
		return err
	}

	invokeOutput, err := lambdaClient.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(lambdaName),
		Payload:        []byte(fmt.Sprintf(`{"bucket": "%s", "key": "%s"}`, bucket, key)),
		InvocationType: "RequestResponse",
	})

	if err != nil {
		return err
	}

	log.Printf("Invoked %d %s", invokeOutput.StatusCode, string(invokeOutput.Payload))

	return nil
}

func processAll(w http.ResponseWriter, r *http.Request) {
	bucketName, err := bcv.GetBcvBucket()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s3Client, err := bcv.GetS3Client(r.Context())
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

	for _, item := range s3List.Contents {

		//err := invokeParsingFunction(r.Context(), bucketName, *item.Key)
		//if err != nil {
		//	http.Error(w, err.Error(), http.StatusInternalServerError)
		//	return
		//}
		go func(item types.Object) {
			defer wg.Done()

			ProcessAll := true
			err := file.ParseFile(file.ParsingParams{
				Ctx:        r.Context(),
				Bucket:     bucketName,
				Key:        *item.Key,
				ProcessAll: &ProcessAll,
			})

			if err != nil {
				handleErr(err)
				return
			}

		}(item)
	}

	wg.Wait()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

func lookUp(w http.ResponseWriter, r *http.Request) {
	err := bcv.Check(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
