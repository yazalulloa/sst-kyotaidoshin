package profile

import (
	"context"
	"log"
	"net/http"
	"slices"
	"strings"

	"github.com/yaz/kyo-repo/internal/api"
	"github.com/yaz/kyo-repo/internal/users"
	"github.com/yaz/kyo-repo/internal/util"
)

const _PATH = "/api/profile"

func Routes(holder *api.RouterHolder) {
	holder.PUT(_PATH+"/notifications/new_rate", changeNewRate)
}

func changeNewRate(w http.ResponseWriter, r *http.Request) {

	userId, ok := r.Context().Value(util.USER_ID).(string)
	if !ok {
		log.Println("changeNewRate: user id not found")
		http.Error(w, "Unauthorized", http.StatusNotFound)
		return
	}

	if userId == "" {
		log.Println("changeNewRate: user id is empty")
		http.Error(w, "Unauthorized", http.StatusNotFound)
		return
	}

	act := util.GetQueryParamAsBool(r, "active")

	go func() { changeEvent(context.Background(), userId, act) }()

	w.WriteHeader(http.StatusNoContent)
}

func changeEvent(ctx context.Context, userId string, active bool) {
	log.Printf("changeEvent: userId=%s, active=%t", userId, active)
	repo := users.NewRepository(ctx)

	event := users.NEW_RATE.Name()

	user, err := repo.GetByID(userId)
	if err != nil {
		log.Printf("changeEvent: failed to get user %s: %v", userId, err)
		return
	}

	events := make([]string, 0)

	if user.NotificationEvents != nil {
		events = strings.Split(*user.NotificationEvents, ",")
	}

	if active {
		if slices.Contains(events, event) {
			log.Printf("changeEvent: user already has event %s %s", event, events)
			return
		}
		events = append(events, event)
	} else {
		i := slices.Index(events, event)
		if i == -1 {
			log.Printf("changeEvent: user does not have event %s %s", event, events)
			return
		}
		events = slices.Delete(events, i, i+1)
	}

	rowsAffected, err := repo.UpdateNotificationEvents(userId, strings.Join(events, ","))
	if err != nil {
		log.Printf("changeEvent: failed to update user %s: %v", userId, err)
		return
	}

	log.Printf("changeEvent: updated user %s, rows affected: %d", userId, rowsAffected)
}
