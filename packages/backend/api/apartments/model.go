package apartments

import "db/gen/model"

type Apt struct {
	Number string
	Name   string
}

type RequestQuery struct {
	lastBuildingId string
	lastNumber     string
	q              string
	buildings      []string
	Limit          int
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
	CardId string
	Key    string
	Item   model.Apartments
}

type Keys struct {
	BuildingId string
	Number     string
}

func keys(apartments model.Apartments) Keys {
	return Keys{
		BuildingId: apartments.BuildingID,
		Number:     apartments.Number,
	}
}
