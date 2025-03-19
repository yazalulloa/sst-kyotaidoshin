package permissions

import (
	"fmt"
	"github.com/google/uuid"
	"kyotaidoshin/api"
)

func insertAll() (int64, error) {

	perms := api.All()
	array := make([]string, len(perms))
	for i := range perms {
		array[i] = perms[i].Name()
	}

	return insertBulk(array)
}

func tableResponse() (*TableResponse, error) {

	perms, err := selectAll()
	if err != nil {
		return nil, err
	}

	items := make([]Item, len(perms))
	for i := range perms {
		items[i] = Item{
			CardId: "permissions-" + uuid.NewString(),
			Key:    fmt.Sprint(*perms[i].ID),
			Item:   perms[i],
		}
	}

	return &TableResponse{
		Counters: Counters{
			TotalCount: int64(len(perms)),
		},
		Results: items,
	}, nil
}
