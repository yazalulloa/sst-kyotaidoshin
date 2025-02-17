package apartments

import (
	"db/gen/model"
)

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
	Emails []string
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

type ApartmentDto struct {
	BuildingID string   `json:"building_id"`
	Number     string   `json:"number"`
	Name       string   `json:"name"`
	Aliquot    float64  `json:"aliquot"`
	Emails     []string `json:"emails"`
}
