package bcv_files

import (
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/yaz/kyo-repo/internal/api"
	"github.com/yaz/kyo-repo/internal/bcv"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/util"
)

const _PATH = "/api/bcv-files"

func Routes(holder *api.RouterHolder) {

	holder.GET(_PATH, display, api.BCV_FILES_READ)
	//holder.DELETE(_PATH+"/{id}", bcvFileDelete, api.BcvBucketDeleteRecaptchaAction, api.BCV_FILES_WRITE)
	holder.POST(_PATH+"/process/{id}", process, api.BcvBucketProcessRecaptchaAction, api.BCV_FILES_WRITE)
	holder.GET(_PATH+"/process-all", processAll, api.BCV_FILES_WRITE)
	//holder.GET(_PATH+"/look-up", lookUp, api.BCV_FILES_WRITE)
}

func display(w http.ResponseWriter, r *http.Request) {

	repo := NewRepository(r.Context())

	files, err := repo.selectAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	results := make([]Item, len(files))
	for i, file := range files {
		results[i] = Item{
			Item:          file,
			SizeFormatted: util.FormatBytes(file.FileSize),
			FileDate:      file.FileDate.UnixMilli(),
			ProcessedAt:   file.ProcessedAt.UnixMilli(),
			CreatedAt:     file.CreatedAt.UnixMilli(),
			CardId:        "bcv-files-" + uuid.NewString(),
			Key:           *util.Encode(*file.Link),
		}
	}

	response := TableResponse{
		TotalCount: len(results),
		Results:    results,
	}

	err = Display(response).Render(r.Context(), w)
	if err != nil {
		log.Printf("Error rendering table view: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type Item struct {
	Item          model.BcvFiles
	SizeFormatted string
	FileDate      int64
	ProcessedAt   int64
	CreatedAt     int64
	CardId        string
	Key           string
}

type TableResponse struct {
	TotalCount int
	Results    []Item
}

func processAll(w http.ResponseWriter, r *http.Request) {

	err := processAllFiles(r.Context())
	if err != nil {
		log.Printf("Error processing all files: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)

}

func process(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var str string
	err := util.Decode(id, &str)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	service, err := bcv.NewService(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = processBcvFile(r.Context(), service, str, true)
	if err != nil {
		log.Printf("Error processing file %s: %v", str, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
