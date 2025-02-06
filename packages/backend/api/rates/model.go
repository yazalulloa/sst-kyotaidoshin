package rates

import (
	"db/gen/model"
	"time"
)

type RateQuery struct {
	LastId     int32
	Limit      int
	DateOfRate *time.Time
	Currencies []string
	SortOrder  string
}

type RateTableResponse struct {
	Counters    Counters
	NextPageUrl string
	Results     []Item
}

type Counters struct {
	TotalCount int
	QueryCount *int
}

type Item struct {
	CardId     string
	Key        string
	Item       model.Rates
	DateOfRate string
	DateOfFile int64
	CreatedAt  int64
}

type Pagination struct {
}
