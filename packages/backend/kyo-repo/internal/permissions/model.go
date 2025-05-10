package permissions

import "kyo-repo/internal/db/gen/model"

type Item struct {
	CardId string
	Key    string
	Item   model.Permissions
}

type TableResponse struct {
	Counters    Counters
	NextPageUrl string
	Results     []Item
}

type Counters struct {
	TotalCount int64
	QueryCount *int64
}

type PermDto struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

type PermWithLabels struct {
	Label string    `json:"label"`
	Items []PermDto `json:"items"`
}
