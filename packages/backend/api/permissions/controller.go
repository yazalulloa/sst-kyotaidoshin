package permissions

import (
	"fmt"
	"github.com/gorilla/mux"
	"kyotaidoshin/util"
	"net/http"
	"strings"
)

const _PATH = "/api/permissions"

func Routes(server *mux.Router) {

	server.HandleFunc(_PATH+"/all", permissionsAll).Methods("POST")
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

	//all, err := allItems()
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	return
	//}

	//_, err = util.WriteJSON(w, all)
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	return
	//}
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
