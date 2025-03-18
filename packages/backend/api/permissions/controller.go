package permissions

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"kyotaidoshin/util"
	"net/http"
	"strings"
)

const _PATH = "/api/permissions"

func Routes(server *mux.Router) {

	server.HandleFunc(_PATH+"/all", permissionsAll).Methods("POST")
	server.HandleFunc(_PATH+"/all", getAllWithLabels).Methods("GET")
	server.HandleFunc(_PATH+"/search", search).Methods("GET")
	server.HandleFunc(_PATH+"/{id}", permissionsDelete).Methods("DELETE")
}

func permissionsAll(w http.ResponseWriter, r *http.Request) {

	all, err := insertAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write([]byte(fmt.Sprintf("Inserted: %v", all)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func search(w http.ResponseWriter, r *http.Request) {

	all, err := tableResponse()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = SearchView(*all).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func permissionsDelete(w http.ResponseWriter, r *http.Request) {

	key := strings.TrimSpace(mux.Vars(r)["id"])

	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	id := util.StringToInt32(key)

	_, err := deleteById(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getAllWithLabels(w http.ResponseWriter, r *http.Request) {

	dbPerms, err := selectAll()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	labels := WithLabels()
	array := make([]PermWithLabels, 0)

	for _, label := range labels {
		perms := make([]PermDto, 0)
		permWithLabel := PermWithLabels{
			Label: label.Label,
		}

		for _, perm := range label.Perms {
			for _, dbPerm := range dbPerms {
				if dbPerm.Name == perm.Name() {
					perms = append(perms, PermDto{
						ID:   *dbPerm.ID,
						Name: dbPerm.Name,
					})
				}

			}
		}

		if len(perms) > 0 {
			permWithLabel.Items = perms
			array = append(array, permWithLabel)
		}
	}

	byteArray, err := json.Marshal(array)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	base64Str := base64.URLEncoding.EncodeToString(byteArray)

	err = permsWithLabels(base64Str).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
