package start

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"kyotaidoshin/users"
	"kyotaidoshin/util"
	"log"
	"net/http"
)

func Routes(server *mux.Router) {

	server.HandleFunc("/api/init", getInit).Methods("GET")
}

func getInit(w http.ResponseWriter, r *http.Request) {

	pages := []Page{
		{
			Id:   "nav-apartments",
			Path: "/apartments",
		},
		{
			Id:   "nav-buildings",
			Path: "/buildings",
			SubRoutes: []SubRoute{
				{
					Id:   "nav-buildings-new",
					Path: "/buildings/new",
				},
				{
					Id:   "nav-buildings-edit",
					Path: "/buildings/edit/:id",
				},
			},
		},
		{
			Id:   "nav-rates",
			Path: "/rates",
		},
		{
			Id:   "nav-receipts",
			Path: "/receipts",
			SubRoutes: []SubRoute{
				{
					Id:   "nav-receipts-edit",
					Path: "/receipts/edit/:id",
				},
				{
					Id:   "nav-receipts-view",
					Path: "/receipts/view/:id",
				},
			},
		},
		{
			Id:   "nav-bcv-files",
			Path: "/bcv-files",
		},
		{
			Id:   "nav-users",
			Path: "/users",
		},
	}

	bytes, err := json.Marshal(pages)

	if err != nil {
		log.Printf("Error marshalling pages: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userId, ok := r.Context().Value(util.USER_ID).(string)
	if !ok {
		http.Error(w, "userId not found", http.StatusNotFound)
		return
	}

	user, err := users.GetByID(userId)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//var userPayload util.UserPayload
	//err = json.Unmarshal([]byte(payload), &userPayload)
	//if err != nil {
	//	log.Printf("Error unmarshalling payload: %v", err)
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	return
	//}

	err = Init(pages, string(bytes), *user).Render(r.Context(), w)
	if err != nil {
		log.Printf("Error rendering init: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
