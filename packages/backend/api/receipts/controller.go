package receipts

import (
	"aws_h"
	"bytes"
	"compress/flate"
	"db/gen/model"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-playground/form"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/sync/syncmap"
	"io"
	"kyotaidoshin/api"
	"kyotaidoshin/buildings"
	"kyotaidoshin/debts"
	"kyotaidoshin/expenses"
	"kyotaidoshin/extraCharges"
	"kyotaidoshin/rates"
	"kyotaidoshin/receiptPdf"
	"kyotaidoshin/util"
	"log"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

const _PATH = "/api/receipts"
const _SEARCH = _PATH + "/search"
const _UPLOAD_BACKUP_FORM = _PATH + "/uploadBackupForm"
const _UPLOAD_BACKUP = _PATH + "/upload/backup"
const _DOWNLOAD_ZIP_FILE = _PATH + "/download/zip"
const _DOWNLOAD_PDF_FILE = _PATH + "/download/pdf"
const _SEND_PDFS = _PATH + "/send_pdfs"
const _SEND_PDFS_PROGRESS = _SEND_PDFS + "/progress"
const _DUPLICATE = _PATH + "/duplicate"

func Routes(server *mux.Router) {

	server.HandleFunc(_SEARCH, search).Methods("GET")
	server.HandleFunc(_PATH, receiptPost).Methods("POST")
	server.HandleFunc(_PATH, receiptPut).Methods("PUT")
	server.HandleFunc(_PATH+"/clear_pdfs", clearPdfs).Methods("DELETE")
	server.HandleFunc(_PATH+"/{key}", receiptDelete).Methods("DELETE")
	server.HandleFunc(_PATH+"/buildingsIds", getBuildingIds).Methods("GET")
	//server.HandleFunc(_PATH+"/init", getInit).Methods("GET")
	server.HandleFunc(_UPLOAD_BACKUP_FORM, getUploadBackupForm).Methods("GET")
	server.HandleFunc(_UPLOAD_BACKUP, uploadBackup).Methods("POST")
	server.HandleFunc(_PATH+"/years", getYears).Methods("GET")
	//server.HandleFunc(_PATH+"/buildingsIds", getBuildingIds).Methods("GET")
	server.HandleFunc(_PATH+"/formData/{key}", formData).Methods("GET")
	server.HandleFunc(_PATH+"/view/{key}", getReceiptView).Methods("GET")
	server.HandleFunc(_DOWNLOAD_ZIP_FILE+"/{key}", getZip).Methods("GET")
	server.HandleFunc(_DOWNLOAD_PDF_FILE+"/{key}", getPdf).Methods("GET")
	server.HandleFunc(_SEND_PDFS+"/{key}", sendPdfs).Methods("GET")
	server.HandleFunc(_SEND_PDFS_PROGRESS+"/{key}", sendPdfsProgress).Methods("GET")
	server.HandleFunc(_PATH+"/upload_form", getUploadForm).Methods("GET")
	server.HandleFunc(_PATH+"/new_from_file", newFromFile).Methods("POST")
	server.HandleFunc(_DUPLICATE+"/{key}", duplicateReceipt).Methods("POST")
	server.HandleFunc(_PATH+"/apts", getSendApts).Methods("GET")
}

func getBuildingIds(w http.ResponseWriter, r *http.Request) {
	buildingIds, err := buildings.SelectIds()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	str := util.StringArrayToString(buildingIds)

	err = BuildingIdsView(str).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getUploadBackupForm(w http.ResponseWriter, r *http.Request) {

	component, err := api.BuildUploadForm(r, "BACKUPS/RECEIPTS/")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = component.Render(r.Context(), w)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func uploadBackup(w http.ResponseWriter, r *http.Request) {

	component, err := api.ProcessUploadBackup(r, "/receipts", ProcessDecoder)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = component.Render(r.Context(), w)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ProcessDecoder(decoder *json.Decoder) (int64, error) {
	var records []ReceiptRecord
	err := decoder.Decode(&records)
	if err != nil {
		log.Printf("Error decoding json: %s", err)
		return 0, err
	}

	slices.SortFunc(records, func(a, b ReceiptRecord) int {

		lhs, err := time.Parse(time.DateOnly, a.Receipt.Date)
		if err != nil {
			//panic(err)
			log.Printf("Error parsing date: %s %v", a.Receipt.Date, err)
			return 0
		}

		rhs, err := time.Parse(time.DateOnly, b.Receipt.Date)
		if err != nil {
			//panic(err)
			log.Printf("Error parsing date: %s %v", b.Receipt.Date, err)
			return 0
		}

		return lhs.Compare(rhs)
	})

	array := util.SplitArray(records, 15)

	var total int64
	ratesHolder := RatesHolder{Rates: syncmap.Map{}}
	for _, chunk := range array {
		rowsAffected, err := insertRecord(chunk, &ratesHolder)
		if err != nil {
			return 0, err
		}
		total += rowsAffected
	}

	return total, nil
}

func search(w http.ResponseWriter, r *http.Request) {
	nextPage := util.GetQueryParamAsString(r, "next_page")
	var keys Keys
	if nextPage != "" {
		err := util.Decode(nextPage, &keys)

		if err != nil {
			log.Printf("failed to decode nextPage: %v", err)
			http.Error(w, "Bad Request nextPage", http.StatusBadRequest)
			return
		}
	}
	query := r.URL.Query()
	buildingIds := query["building_input"]
	monthArray := query["month_input"]
	months := make([]int16, 0)
	years := make([]int16, 0)

	for _, month := range monthArray {
		v := util.StringToInt16(month)
		if v >= 1 && v <= 12 {
			months = append(months, v)
			continue
		}
	}

	yearArray := query["year_input"]
	for _, value := range yearArray {
		v := util.StringToInt16(value)
		if v >= 2020 && v <= 2100 {
			years = append(years, v)
			continue
		}
	}

	requestQuery := RequestQuery{
		LastId:    keys.Id,
		Buildings: buildingIds,
		Months:    months,
		Years:     years,
		Limit:     31,
	}

	response, err := getTableResponse(requestQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	results := response.Results
	var nextPageUrl string
	if int64(len(results)) == requestQuery.Limit {
		results = results[:len(results)-1]
		last := results[len(results)-1]
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf(_SEARCH+"?next_page=%s", last.Key))

		if len(requestQuery.Buildings) > 0 {
			for _, building := range requestQuery.Buildings {
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

func getYears(w http.ResponseWriter, r *http.Request) {
	years, err := selectYears()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var builder strings.Builder
	builder.WriteString("years = [")
	for i, year := range years {
		builder.WriteString(fmt.Sprint(year))
		if i < len(years)-1 {
			builder.WriteString(",")
		}
	}

	builder.WriteString("]")

	err = YearsView(builder.String()).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func formData(w http.ResponseWriter, r *http.Request) {
	keyStr := mux.Vars(r)["key"]
	if keyStr == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var keys Keys
	err := util.Decode(keyStr, &keys)
	if err != nil {
		log.Printf("failed to decode key: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	formDto, oErr := getFormDto(keys)

	if oErr != nil {
		http.Error(w, oErr.Error(), http.StatusInternalServerError)
		return
	}

	err = FormView(*formDto).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func receiptPost(w http.ResponseWriter, r *http.Request) {

	createReceipt := func() FormResponse {

		response := FormResponse{}

		err := r.ParseForm()
		if err != nil {
			log.Printf("Error parsing form: %v", err)
			response.errorStr = err.Error()
			return response
		}

		decoder := form.NewDecoder()
		var request ReceiptNewFormRequest
		err = decoder.Decode(&request, r.Form)

		validate, err := util.GetValidator()
		if err != nil {
			log.Printf("Error getting validator: %v", err)
			response.errorStr = err.Error()
			return response
		}

		err = validate.Struct(request)
		if err != nil {
			// Validation failed, handle the error
			errors := err.(validator.ValidationErrors)
			for _, valErr := range errors {
				log.Printf("Validation error: %v", valErr)
			}
			response.errorStr = fmt.Sprintf("Validation error: %s", errors)
			return response
		}

		date, err := time.Parse(time.DateOnly, request.Date)
		if err != nil {
			log.Printf("Error parsing date: %v", err)
			response.errorStr = err.Error()
			return response
		}

		decoded, err := base64.URLEncoding.DecodeString(request.Data)
		if err != nil {
			log.Printf("Error decoding data: %v", err)
			response.errorStr = err.Error()
			return response
		}

		reader := bytes.NewReader(decoded)
		compressionReader := flate.NewReader(reader)
		defer func(compressionReader io.ReadCloser) {
			_ = compressionReader.Close()
		}(compressionReader)

		got, err := io.ReadAll(compressionReader)

		if err != nil {
			log.Printf("Error reading all: %v", err)
			response.errorStr = err.Error()
			return response
		}

		var parsedReceipt ParsedReceipt
		err = json.Unmarshal(got, &parsedReceipt)
		if err != nil {
			log.Printf("Error unmarshalling: %v", err)
			response.errorStr = err.Error()
			return response
		}

		if len(parsedReceipt.Debts) == 0 {
			log.Printf("No debts found")
			response.errorStr = "Invalid Data"
			return response
		}

		if len(parsedReceipt.Expenses) == 0 {
			log.Printf("No expenses found")
			response.errorStr = "Invalid Data"
			return response
		}

		var rateId *int64
		err = util.Decode(request.Rate, &rateId)
		if err != nil {
			log.Printf("Error decoding rateId: %v", err)
			response.errorStr = err.Error()
			return response
		}

		exist, err := rates.CheckRateExist(*rateId)
		if err != nil {
			log.Printf("Error checking rate: %v", err)
			response.errorStr = err.Error()
			return response
		}

		if !exist {
			response.errorStr = "Rate does not exist"
			return response
		}

		receiptId := util.UuidV7()

		receipt := model.Receipts{
			ID:         receiptId,
			BuildingID: request.Building,
			Year:       request.Year,
			Month:      request.Month,
			Date:       date,
			RateID:     *rateId,
		}

		for i := range parsedReceipt.Debts {
			parsedReceipt.Debts[i].BuildingID = receipt.BuildingID
			parsedReceipt.Debts[i].ReceiptID = receipt.ID
		}

		for i := range parsedReceipt.Expenses {
			parsedReceipt.Expenses[i].BuildingID = receipt.BuildingID
			parsedReceipt.Expenses[i].ReceiptID = receipt.ID
		}

		for i := range parsedReceipt.ExtraCharges {
			parsedReceipt.ExtraCharges[i].BuildingID = receipt.BuildingID
			parsedReceipt.ExtraCharges[i].ParentReference = receipt.ID
		}

		var wg sync.WaitGroup
		wg.Add(4)
		errorChan := make(chan error, 4)

		go func() {
			defer wg.Done()
			_, err = insert(receipt)
			if err != nil {
				errorChan <- err
				return
			}
		}()

		go func() {
			defer wg.Done()
			_, err = debts.InsertBulk(parsedReceipt.Debts)
			if err != nil {
				errorChan <- err
				return
			}
		}()

		go func() {
			defer wg.Done()
			_, err = expenses.InsertBulk(parsedReceipt.Expenses)
			if err != nil {
				errorChan <- err
				return
			}
		}()

		go func() {
			defer wg.Done()
			_, err = extraCharges.InsertBulk(parsedReceipt.ExtraCharges)
			if err != nil {
				errorChan <- err
				return
			}
		}()

		wg.Wait()
		close(errorChan)

		err = util.HasErrors(errorChan)
		if err != nil {
			response.errorStr = err.Error()
			return response
		}

		keys := keys(receipt, "")
		response.Key = util.Encode(keys)

		return response
	}

	response := createReceipt()

	if response.errorStr == "" {

		//w.Header().Add("HX-Location", fmt.Sprintf("/receipts/edit/%s", *response.Key))
		//w.WriteHeader(http.StatusOK)

		err := api.AnchorClickInitView(fmt.Sprintf("/receipts/edit/%s", *response.Key)).Render(r.Context(), w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		return
	}

	err := FormResponseView(response).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func receiptPut(w http.ResponseWriter, r *http.Request) {

	updateReceipt := func() FormResponse {

		response := FormResponse{}

		err := r.ParseForm()
		if err != nil {
			log.Printf("Error parsing form: %v", err)
			response.errorStr = err.Error()
			return response
		}

		decoder := form.NewDecoder()
		var request FormRequest
		err = decoder.Decode(&request, r.Form)

		if err != nil {
			log.Printf("Error decoding form: %v", err)
			response.errorStr = err.Error()
			return response
		}

		var keys Keys
		err = util.Decode(request.Key, &keys)
		if err != nil {
			log.Printf("Error decoding key: %v", err)
			response.errorStr = err.Error()
			return response
		}

		validate, err := util.GetValidator()
		if err != nil {
			log.Printf("Error getting validator: %v", err)
			response.errorStr = err.Error()
			return response
		}

		err = validate.Struct(request)
		if err != nil {
			// Validation failed, handle the error
			errors := err.(validator.ValidationErrors)
			for _, valErr := range errors {
				log.Printf("Validation error: %v", valErr)
			}
			response.errorStr = fmt.Sprintf("Validation error: %s", errors)
			return response
		}

		date, err := time.Parse(time.DateOnly, request.Date)
		if err != nil {
			log.Printf("Error parsing date: %v", err)
			response.errorStr = err.Error()
			return response
		}

		var rateId *int64
		err = util.Decode(request.RateKey, &rateId)
		if err != nil {
			log.Printf("Error decoding rateId: %v", err)
			response.errorStr = err.Error()
			return response
		}

		exist, err := rates.CheckRateExist(*rateId)
		if err != nil {
			log.Printf("Error checking rate: %v", err)
			response.errorStr = err.Error()
			return response
		}

		if !exist {
			response.errorStr = "Rate does not exist"
			return response
		}

		receipt := model.Receipts{
			ID:         keys.Id,
			BuildingID: keys.BuildingId,
			Year:       request.Year,
			Month:      request.Month,
			Date:       date,
			RateID:     *rateId,
		}

		_, err = update(receipt)
		if err != nil {
			log.Printf("Error updating receipt: %v", err)
			response.errorStr = err.Error()
			return response
		}

		defer receiptPdf.PublishReceipt(r.Context(), keys.BuildingId, keys.Id)
		if err != nil {
			log.Printf("Error deleting pdf: %v", err)
			response.errorStr = err.Error()
			return response
		}

		return response
	}

	response := updateReceipt()

	err := FormResponseView(response).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func receiptDelete(w http.ResponseWriter, r *http.Request) {
	keyStr := mux.Vars(r)["key"]
	if keyStr == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var keys Keys
	err := util.Decode(keyStr, &keys)
	if err != nil {
		log.Printf("failed to decode key: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	_, err = deleteReceipt(keys)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer receiptPdf.PublishReceipt(r.Context(), keys.BuildingId, keys.Id)

	w.WriteHeader(http.StatusOK)
}

func getReceiptView(w http.ResponseWriter, r *http.Request) {

	keyStr := mux.Vars(r)["key"]
	if keyStr == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var keys Keys
	err := util.Decode(keyStr, &keys)
	if err != nil {
		log.Printf("failed to decode key: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	receipt, err := CalculateReceipt(keys.BuildingId, keys.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	buildingDownloadKeys := util.Encode(DownloadKeys{
		BuildingId: receipt.Building.ID,
		Id:         receipt.Receipt.ID,
		Part:       receipt.Building.ID,
		IsApt:      false,
	})

	receipt.BuildingDownloadKeys = *buildingDownloadKeys

	idMap := make(map[string]string, len(receipt.Apartments)+1)
	tabs := make([]TabId, len(receipt.Apartments)+1)
	idMap[receipt.Building.ID] = "building-" + uuid.NewString()
	tabs[0] = TabId{ID: idMap[receipt.Building.ID], Name: receipt.Building.ID}

	for i := range receipt.Apartments {
		apt := &receipt.Apartments[i]
		idMap[apt.Apartment.Number] = "apartment-" + uuid.NewString()
		tabs[i+1] = TabId{ID: idMap[apt.Apartment.Number], Name: apt.Apartment.Number}

		downloadKeys := util.Encode(DownloadKeys{
			BuildingId: receipt.Building.ID,
			Id:         receipt.Receipt.ID,
			Part:       apt.Apartment.Number,
			IsApt:      true,
		})

		apt.DownloadKeys = *downloadKeys
	}

	byteArray, err := json.Marshal(tabs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	base64Str := base64.URLEncoding.EncodeToString(byteArray)

	err = Views(keyStr, *receipt, idMap, base64Str).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getZip(w http.ResponseWriter, r *http.Request) {
	keyStr := mux.Vars(r)["key"]
	if keyStr == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var keys Keys
	err := util.Decode(keyStr, &keys)
	if err != nil {
		log.Printf("failed to decode key: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	bucketName, err := util.GetReceiptsBucket()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	receipt, err := CalculateReceipt(keys.BuildingId, keys.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	date := receipt.Receipt.Date.Format(time.DateOnly)
	objectKey := fmt.Sprintf("RECEIPS/%s/%s/%s_%s_%s.zip", receipt.Building.ID, receipt.Receipt.ID,
		receipt.Building.ID, strings.ToUpper(receipt.MonthStr), date)

	exists, err := aws_h.FileExistsS3(r.Context(), bucketName, objectKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !exists {
		buf, err := toZip(receipt, r.Context())

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s3Client, err := aws_h.GetS3Client(r.Context())

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		contentLength := int64(buf.Len())
		_, err = s3Client.PutObject(r.Context(), &s3.PutObjectInput{
			Bucket:            aws.String(bucketName),
			Key:               aws.String(objectKey),
			Body:              buf,
			ChecksumAlgorithm: types.ChecksumAlgorithmCrc64nvme,
			//ChecksumCRC32:             nil,
			//ChecksumCRC32C:            nil,
			//ChecksumSHA1:              nil,
			//ChecksumSHA256:            nil,
			ContentLength: &contentLength,
			ContentType:   aws.String("application/zip"),
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	presignClient, err := aws_h.GetPresignClient(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	presignedHTTPRequest, err := presignClient.PresignGetObject(r.Context(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}, func(options *s3.PresignOptions) {
		options.Expires = time.Minute
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("HX-Redirect", presignedHTTPRequest.URL)
	w.WriteHeader(http.StatusOK)
}

func getPdf(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyStr := vars["key"]
	if keyStr == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var keys DownloadKeys
	err := util.Decode(keyStr, &keys)
	if err != nil {
		log.Printf("failed to decode key: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	bucketName, err := util.GetReceiptsBucket()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	receipt, err := CalculateReceipt(keys.BuildingId, keys.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	parts, err := GetParts(receipt, r.Context(), &keys)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	presignClient, err := aws_h.GetPresignClient(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	presignedHTTPRequest, err := presignClient.PresignGetObject(r.Context(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(parts[0].ObjectKey),
	}, func(options *s3.PresignOptions) {
		options.Expires = time.Minute
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("HX-Redirect", presignedHTTPRequest.URL)
	w.WriteHeader(http.StatusNoContent)
}

func clearPdfs(w http.ResponseWriter, r *http.Request) {

	err := receiptPdf.DeleteObjects(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func sendPdfs(w http.ResponseWriter, r *http.Request) {
	keyStr := mux.Vars(r)["key"]
	if keyStr == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var keys Keys
	err := util.Decode(keyStr, &keys)
	if err != nil {
		log.Printf("failed to decode key: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	clientId, err := receiptPdf.PublishSendPdfs(r.Context(), keys.BuildingId, keys.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	progressId := util.Encode(clientId)

	err = SendPdfsView(*progressId).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func sendPdfsProgress(w http.ResponseWriter, r *http.Request) {
	keyStr := mux.Vars(r)["key"]
	if keyStr == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var progressId string
	err := util.Decode(keyStr, &progressId)
	if err != nil {
		log.Printf("failed to decode key: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	progress, err := receiptPdf.GetProgress(r.Context(), progressId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	byteArray, err := json.Marshal(progress)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	encoded := base64.URLEncoding.EncodeToString(byteArray)

	err = SendPdfsProgressView(encoded).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func getUploadForm(w http.ResponseWriter, r *http.Request) {

	component, err := api.BuildUploadForm(r, "NEW_RECEIPTS/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func newFromFile(w http.ResponseWriter, r *http.Request) {

	key := r.FormValue("key")
	if key == "" {
		log.Printf("key is empty")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	dto, err := parseNewReceipt(r.Context(), key)
	if err != nil {
		log.Printf("Error parsing new receipt: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	byteArray, err := json.Marshal(*dto)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	encoded := base64.URLEncoding.EncodeToString(byteArray)

	err = ShowNewReceiptsDialog(encoded).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func duplicateReceipt(w http.ResponseWriter, r *http.Request) {
	keyStr := mux.Vars(r)["key"]
	if keyStr == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var keys Keys
	err := util.Decode(keyStr, &keys)
	if err != nil {
		log.Printf("failed to decode key: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	str, err := duplicate(keys)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = api.AnchorClickInitView(fmt.Sprintf("/receipts/edit/%s", *str)).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getSendApts(w http.ResponseWriter, r *http.Request) {

	str, err := getApts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = SendAptsView(*str).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
