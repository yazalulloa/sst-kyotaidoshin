package users

import (
	"context"
	"db/gen/model"
	"encoding/base64"
	"encoding/json"
	"kyotaidoshin/util"
	"log"
	"sync"
)

type Service struct {
	repo Repository
}

func NewService(ctx context.Context) Service {
	return Service{
		repo: NewRepository(ctx),
	}
}

func (service Service) getTableResponse(requestQuery RequestQuery) (*TableResponse, error) {
	var tableResponse TableResponse
	var wg sync.WaitGroup
	wg.Add(3)
	errorChan := make(chan error, 3)

	go func() {
		defer wg.Done()
		array, err := service.repo.selectList(requestQuery)
		if err != nil {
			errorChan <- err
			return
		}

		results := make([]Item, len(array))
		for i, item := range array {
			obj, err := toItem(item.Users, item.Role, item.Chat, nil)
			if err != nil {
				errorChan <- err
				return
			}

			results[i] = *obj
		}

		tableResponse.Results = results
	}()

	go func() {
		defer wg.Done()
		totalCount, err := service.repo.getTotalCount()

		if err != nil {
			errorChan <- err
			return
		}

		tableResponse.Counters.TotalCount = totalCount
	}()

	go func() {
		defer wg.Done()
		queryCount, err := service.repo.getQueryCount(requestQuery)
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

func toItem(user model.Users, role *model.Roles, chat *model.TelegramChats, oldCardId *string) (*Item, error) {

	var cardIdStr string
	if oldCardId != nil {
		cardIdStr = *oldCardId
	} else {
		cardIdStr = cardId()
	}

	keys := keys(user, cardIdStr)
	key := *util.Encode(keys)

	var roleId *int32
	if role != nil {
		roleId = role.ID
	}

	params := UpdateParams{
		Key:    key,
		RoleId: roleId,

		Provider: user.Provider,
		Email:    user.Email,
		Username: user.Username,
		Picture:  user.Picture,
	}

	byteArray, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	updateParams := base64.StdEncoding.EncodeToString(byteArray)

	var lastLoginAt int64

	if user.LastLoginAt != nil {
		lastLoginAt = user.LastLoginAt.UnixMilli()
	}

	return &Item{
		CardId:       cardIdStr,
		Key:          key,
		Item:         user,
		Role:         role,
		Chat:         chat,
		UpdateParams: &updateParams,
		CreatedAt:    user.CreatedAt.UnixMilli(),
		LastLoginAt:  lastLoginAt,
	}, nil
}

func (service Service) deleteRateReturnCounters(id string, requestQuery RequestQuery) (*Counters, error) {

	_, err := service.repo.deleteById(id)
	if err != nil {
		return nil, err
	}

	var counters Counters
	var wg sync.WaitGroup
	var once sync.Once
	var oErr error
	handleErr := func(e error) {
		if e != nil {
			once.Do(func() {
				oErr = e
			})
		}
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		totalCount, err := service.repo.getTotalCount()
		if err != nil {
			handleErr(err)
			return
		}
		counters.TotalCount = totalCount
	}()

	go func() {
		defer wg.Done()
		queryCount, err := service.repo.getQueryCount(requestQuery)
		if err != nil {
			handleErr(err)
			return
		}
		counters.QueryCount = queryCount
	}()

	wg.Wait()

	if oErr != nil {
		return nil, oErr
	}

	return &counters, nil

}

func (service Service) updateRole(id string, roleId *int32) (int64, error) {

	if roleId == nil {
		log.Printf("roleId is nil, deleting user role")
		return service.repo.deleteUserRole(id, nil)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	rowsChan := make(chan int64, 2)
	errorChan := make(chan error, 2)

	go func() {
		defer wg.Done()
		rowsAffected, err := service.repo.insertUserRole(id, *roleId)
		if err != nil {
			errorChan <- err
			return
		}

		rowsChan <- rowsAffected
	}()

	go func() {
		defer wg.Done()
		rowsAffected, err := service.repo.deleteUserRole(id, roleId)
		if err != nil {
			errorChan <- err
			return
		}

		rowsChan <- rowsAffected
	}()

	wg.Wait()
	close(errorChan)
	close(rowsChan)

	err := util.HasErrors(errorChan)
	if err != nil {
		return 0, err
	}

	sum := int64(0)
	for rows := range rowsChan {
		sum += rows
	}

	return sum, nil
}

func (service Service) getItemWitRole(keys Keys) (*Item, error) {

	user, err := service.repo.getWitRole(keys.ID)

	if err != nil {
		return nil, err
	}

	return toItem(user.Users, user.Role, nil, &keys.CardId)
}
