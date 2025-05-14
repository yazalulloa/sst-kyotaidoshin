package permissions

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/yaz/kyo-repo/internal/api"
)

type Service struct {
	repo Repository
}

func NewService(ctx context.Context) Service {
	return Service{
		repo: NewRepository(ctx),
	}
}

func (service Service) insertAll() (int64, error) {

	perms := api.All()
	array := make([]string, len(perms))
	for i := range perms {
		array[i] = perms[i].Name()
	}

	return service.repo.insertBulk(array)
}

func (service Service) tableResponse() (*TableResponse, error) {

	perms, err := service.repo.selectAll()
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
