package start

import (
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/telegram"
)

type Page struct {
	Id        string     `json:"Id"`
	Path      string     `json:"Path"`
	SubRoutes []SubRoute `json:"SubRoutes"`
}

type SubRoute struct {
	Id   string `json:"Id"`
	Path string `json:"Path"`
}

type TelegramChat struct {
	Chat     model.TelegramChats
	Pictures []telegram.ProfilePicture
	Pic      *telegram.ProfilePicture
}
