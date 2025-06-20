package start

import (
	"database/sql"
	"errors"
	"github.com/go-jet/jet/v2/qrm"
	"github.com/go-jet/jet/v2/sqlite"
	"github.com/gorilla/mux"
	"github.com/yaz/kyo-repo/internal/api"
	"github.com/yaz/kyo-repo/internal/db"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	. "github.com/yaz/kyo-repo/internal/db/gen/table"
	"github.com/yaz/kyo-repo/internal/users"
	"github.com/yaz/kyo-repo/internal/util"
	"log"
	"net/http"
	"slices"
	"sync"
)

func Routes(server *mux.Router) {

	server.HandleFunc("/api/init", getInit).Methods("GET")
}

func getInit(w http.ResponseWriter, r *http.Request) {

	userId, ok := r.Context().Value(util.USER_ID).(string)
	if !ok {
		log.Println("getInit: user id not found")
		http.Error(w, "Unauthorized", http.StatusNotFound)
		return
	}

	if userId == "" {
		log.Println("getInit: user id is empty")
		http.Error(w, "Unauthorized", http.StatusNotFound)
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)
	errorChan := make(chan error, 2)

	var dest []model.Permissions
	var user *model.Users

	go func() {
		defer wg.Done()

		stmt := Permissions.SELECT(Permissions.AllColumns).
			FROM(
				Users.INNER_JOIN(UserRoles, Users.ID.EQ(UserRoles.UserID)).
					INNER_JOIN(Roles, UserRoles.RoleID.EQ(Roles.ID)).
					INNER_JOIN(RolePermissions, Roles.ID.EQ(RolePermissions.RoleID)).
					INNER_JOIN(Permissions, RolePermissions.PermissionID.EQ(Permissions.ID)),
			).
			WHERE(Users.ID.EQ(sqlite.String(userId)))

		err := stmt.QueryContext(r.Context(), db.GetDB().DB, &dest)

		if err != nil {
			log.Printf("Error getting permissions: %v", err)
			errorChan <- err
			return
		}
	}()

	go func() {
		defer wg.Done()

		anUser, err := users.NewRepository(r.Context()).GetByID(userId)

		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, qrm.ErrNoRows) {
			log.Printf("user %s not found", userId)
			return
		}

		if err != nil {
			log.Printf("Error getting user: %v", err)
			errorChan <- err
			return
		}

		user = anUser
	}()

	wg.Wait()
	close(errorChan)

	err := util.HasErrors(errorChan)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if user == nil {
		api.RemoveAuthCookies(w, r)
		api.RedirectToAuthServer(w, r)
		return
	}

	pages := make([]Page, 0)

	array := make([]string, len(dest))

	for i, perm := range dest {
		array[i] = perm.Name

		switch perm.Name {
		case api.APARTMENTS_READ.Name():
			pages = append(pages, Page{
				Id:   "nav-apartments",
				Path: "/apartments",
			})
			break
		case api.BUILDINGS_READ.Name():
			subRoutes := make([]SubRoute, 0)

			hasWrite := slices.ContainsFunc(dest, func(p model.Permissions) bool {
				return p.Name == api.BUILDINGS_WRITE.Name()
			})

			if hasWrite {
				subRoutes = append(subRoutes, SubRoute{
					Id:   "nav-buildings-new",
					Path: "/buildings/new",
				})
				subRoutes = append(subRoutes, SubRoute{
					Id:   "nav-buildings-edit",
					Path: "/buildings/edit/:id",
				})
			}

			pages = append(pages, Page{
				Id:        "nav-buildings",
				Path:      "/buildings",
				SubRoutes: subRoutes,
			})
			break
		case api.RATES_READ.Name():
			pages = append(pages, Page{
				Id:   "nav-rates",
				Path: "/rates",
			})
			break
		case api.RECEIPTS_READ.Name():
			subRoutes := make([]SubRoute, 0)

			hasWrite := slices.ContainsFunc(dest, func(p model.Permissions) bool {
				return p.Name == api.RECEIPTS_WRITE.Name()
			})

			if hasWrite {
				subRoutes = append(subRoutes, SubRoute{
					Id:   "nav-receipts-edit",
					Path: "/receipts/edit/:id",
				})
			}

			subRoutes = append(subRoutes, SubRoute{
				Id:   "nav-receipts-view",
				Path: "/receipts/view/:id",
			})

			pages = append(pages, Page{
				Id:        "nav-receipts",
				Path:      "/receipts",
				SubRoutes: subRoutes,
			})
			break
		case api.BCV_FILES_READ.Name():
			pages = append(pages, Page{
				Id:   "nav-bcv-files",
				Path: "/bcv-files",
			})
			break
		case api.USERS_READ.Name():
			pages = append(pages, Page{
				Id:   "nav-users",
				Path: "/users",
			})
			break
		case api.PERMISSIONS_READ.Name():
			pages = append(pages, Page{
				Id:   "nav-permissions",
				Path: "/permissions",
			})
			break
		case api.ROLES_READ.Name():
			pages = append(pages, Page{
				Id:   "nav-roles",
				Path: "/roles",
			})
			break
		case api.ADMIN.Name():
			pages = append(pages, Page{
				Id:   "nav-admin",
				Path: "/admin",
			})
			break

		}
	}

	permStr, err := util.ObjToJsonBase64(array)

	if err != nil {
		log.Printf("Error marshalling perms: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pagesStr, err := util.ObjToJsonBase64(pages)

	if err != nil {
		log.Printf("Error marshalling perms: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = Init(pages, permStr, pagesStr, *user).Render(r.Context(), w)
	if err != nil {
		log.Printf("Error rendering init: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
