package telegram_api

import (
	"encoding/json"
	"kyotaidoshin/api"
	"kyotaidoshin/util"
	"net/http"
	"telegram-webhook/telegram"
)

const _PATH = "/api/telegram"
const _WEBHOOK_PATH = _PATH + "/webhook"

func Routes(holder *api.RouterHolder) {

	holder.POST(_WEBHOOK_PATH, postWebhook, api.TelegramSetWebhookRecaptchaAction)
	holder.DELETE(_WEBHOOK_PATH, deleteWebhook, api.TelegramDeleteWebhookRecaptchaAction)
	holder.GET(_WEBHOOK_PATH, getWebhook)
	holder.GET(_PATH+"/start", getStartUrl)
	holder.GET(_PATH+"/info", getInfo)
}

func postWebhook(w http.ResponseWriter, r *http.Request) {

	err := telegram.NewService(r.Context()).SetWebhook()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func deleteWebhook(w http.ResponseWriter, r *http.Request) {

	err := telegram.NewService(r.Context()).DeleteWebhook()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getWebhook(w http.ResponseWriter, r *http.Request) {

	info, err := telegram.NewService(r.Context()).GetWebhook()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(info)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getStartUrl(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value(util.USER_ID)
	if userId == nil || userId.(string) == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	url, err := telegram.NewService(r.Context()).StartUrl(userId.(string))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = startUrl(url).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func getInfo(w http.ResponseWriter, r *http.Request) {

	info, err := telegram.NewService(r.Context()).Info()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = infoView(info).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
