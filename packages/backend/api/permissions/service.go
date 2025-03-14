package permissions

import (
	"fmt"
	"github.com/google/uuid"
)

func insertAll() (int64, error) {

	perms := All()
	array := make([]string, len(perms))
	for i := range perms {
		array[i] = perms[i].Name()
	}

	return insertBulk(array)
}

func allItems() ([]Item, error) {

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

	return items, nil
}
