package users

import (
	"context"
	"db/gen/model"
	"encoding/base64"
	"encoding/json"
	"kyotaidoshin/util"
	"strings"
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

	events := make([]string, 0)

	if user.NotificationEvents != nil {
		events = strings.Split(*user.NotificationEvents, ",")
	}

	params := UpdateParams{
		Key:                key,
		RoleId:             roleId,
		NotificationEvents: events,

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
	wg.Add(2)
	errorChan := make(chan error, 2)

	go func() {
		defer wg.Done()
		totalCount, err := service.repo.getTotalCount()
		if err != nil {
			errorChan <- err
			return
		}
		counters.TotalCount = totalCount
	}()

	go func() {
		defer wg.Done()
		queryCount, err := service.repo.getQueryCount(requestQuery)
		if err != nil {
			errorChan <- err
			return
		}
		counters.QueryCount = queryCount
	}()

	wg.Wait()
	close(errorChan)

	err = util.HasErrors(errorChan)
	if err != nil {
		return nil, err
	}

	return &counters, nil

}

func (service Service) updateUser(id string, roleId *int32, notificationEvents string) (int64, error) {

	justDeleteRole := roleId == nil
	numberOfWorkers := 2
	if !justDeleteRole {
		numberOfWorkers = 3
	}

	var wg sync.WaitGroup
	wg.Add(numberOfWorkers)
	rowsChan := make(chan int64, numberOfWorkers)
	errorChan := make(chan error, numberOfWorkers)

	if justDeleteRole {
		go func() {
			defer wg.Done()
			rowsAffected, err := service.repo.deleteUserRole(id, nil)
			if err != nil {
				errorChan <- err
				return
			}

			rowsChan <- rowsAffected
		}()

	} else {

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
	}

	go func() {
		defer wg.Done()
		rowsAffected, err := service.repo.updateNotificationEvents(id, notificationEvents)
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

	return toItem(user.Users, user.Role, user.Chat, &keys.CardId)
}
