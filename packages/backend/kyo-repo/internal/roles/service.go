package roles

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/util"
	"log"
	"sync"
)

func getTableResponse(ctx context.Context, requestQuery RequestQuery) (*TableResponse, error) {
	var tableResponse TableResponse

	var wg sync.WaitGroup
	wg.Add(3)
	errorChan := make(chan error, 3)

	go func() {
		defer wg.Done()
		array, err := selectList(ctx, requestQuery)
		if err != nil {
			errorChan <- err
			return
		}

		items := make([]Item, len(array))
		for i, item := range array {

			obj, err := toItem(RoleWithPermissions{
				Role:        item.Roles,
				Permissions: item.Permissions,
			}, nil)
			if err != nil {
				errorChan <- err
				return
			}

			items[i] = *obj

		}
		tableResponse.Results = items
	}()

	go func() {
		defer wg.Done()
		totalCount, err := getTotalCount(ctx)
		if err != nil {
			errorChan <- err
			return
		}
		tableResponse.Counters.TotalCount = totalCount
	}()

	go func() {
		defer wg.Done()
		queryCount, err := getQueryCount(ctx, requestQuery)
		if err != nil {
			errorChan <- err
			return
		}
		if queryCount != nil {
			tableResponse.Counters.QueryCount = queryCount
		}
	}()

	wg.Wait()
	close(errorChan)

	err := util.HasErrors(errorChan)
	if err != nil {
		return nil, err
	}

	return &tableResponse, nil
}

func toItem(item RoleWithPermissions, oldCardId *string) (*Item, error) {

	var cardIdStr string
	if oldCardId != nil {
		cardIdStr = *oldCardId
	} else {
		cardIdStr = cardId()
	}

	keys := keys(item.Role, cardIdStr)
	key := *util.Encode(keys)

	perms := make([]int32, len(item.Permissions))

	for i, perm := range item.Permissions {
		perms[i] = *perm.ID
	}

	params := UpdateParams{
		Key:   key,
		Name:  item.Role.Name,
		Perms: perms,
	}

	byteArray, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	updateParams := base64.StdEncoding.EncodeToString(byteArray)

	return &Item{
		CardId:       cardIdStr,
		Key:          key,
		Item:         item,
		UpdateParams: &updateParams,
	}, nil
}

func insertRole(ctx context.Context, name string, perms []int32) (int32, error) {
	role := model.Roles{
		Name: name,
	}
	roleId, err := insert(ctx, role)
	if err != nil {
		return 0, err
	}

	_, err = insertPerms(ctx, int32(roleId), perms)

	return int32(roleId), nil
}

func updateRole(ctx context.Context, role model.Roles, perms []int32) (int64, error) {

	var wg sync.WaitGroup
	wg.Add(3)
	errorChan := make(chan error, 3)

	go func() {
		defer wg.Done()
		_, err := update(ctx, role)
		if err != nil {
			errorChan <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		_, err := insertPerms(ctx, *role.ID, perms)
		if err != nil {
			errorChan <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		rows, err := deleteOnUpdate(ctx, *role.ID, perms)
		if err != nil {
			errorChan <- err
			return
		}

		log.Printf("Deleted %d rows", rows)
	}()

	wg.Wait()
	close(errorChan)

	err := util.HasErrors(errorChan)
	if err != nil {
		return 0, err
	}

	return 0, nil
}

func getItem(ctx context.Context, id int32, oldCardId *string) (*Item, error) {
	role, err := selectById(ctx, id)
	if err != nil {
		return nil, err
	}

	return toItem(RoleWithPermissions{
		Role:        role.Roles,
		Permissions: role.Permissions,
	}, oldCardId)

}

func deleteAndReturnCounters(ctx context.Context, keys Keys) (*Counters, error) {

	_, err := deleteById(ctx, keys.ID)
	if err != nil {
		return nil, err
	}

	totalCount, err := getTotalCount(ctx)
	if err != nil {
		return nil, err
	}

	return &Counters{
		TotalCount: totalCount,
	}, nil
}
