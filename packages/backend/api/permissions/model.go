package permissions

import "db/gen/model"

type Item struct {
	CardId string
	Key    string
	Item   model.Permissions
}
