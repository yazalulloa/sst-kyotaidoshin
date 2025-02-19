package api

import (
	"github.com/gorilla/mux"
	"net/http"
)

func Routes(server *mux.Router) {

	server.HandleFunc("/api/init", getInit).Methods("GET")
}

func getInit(w http.ResponseWriter, r *http.Request) {

}
