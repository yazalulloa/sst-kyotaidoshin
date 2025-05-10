package rates

import (
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	"github.com/yaz/kyo-repo/internal/util"
	"time"
)

type RequestQuery struct {
	LastId     int64
	Limit      int
	DateOfRate *time.Time
	Currencies []string
	SortOrder  util.SortOrderType
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

type Item struct {
	CardId     string
	Key        string
	Item       model.Rates
	DateOfRate string
	DateOfFile int64
	CreatedAt  int64
}

type Option struct {
	Key        string  `json:"key"`
	DateOfRate string  `json:"dateOfRate"`
	Rate       float64 `json:"rate"`
}
