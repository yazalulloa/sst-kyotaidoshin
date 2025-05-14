package isr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/yaz/kyo-repo/internal/db"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	. "github.com/yaz/kyo-repo/internal/db/gen/table"
	"log"
	"slices"
)

type Repository struct {
	ctx context.Context
}

func NewRepository(ctx context.Context) Repository {
	return Repository{ctx: ctx}
}

func (repo Repository) getDistinctCurrencies() ([]string, error) {
	stmt := Rates.SELECT(Rates.FromCurrency).DISTINCT().FROM(Rates)
	var dest []string
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}

func (repo Repository) buildingIds() ([]string, error) {
	stmt := Buildings.SELECT(Buildings.ID).FROM(Buildings)

	var buildings []model.Buildings
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &buildings)
	if err != nil {
		return nil, err
	}

	array := make([]string, len(buildings))
	for i, building := range buildings {
		array[i] = building.ID
	}

	return array, nil
}

func (repo Repository) apartmentBuildings() ([]string, error) {
	stmt := Apartments.SELECT(Apartments.BuildingID).DISTINCT().FROM(Apartments)
	var dest []string
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	og, err := repo.buildingIds()
	if err != nil {
		return nil, err
	}

	for _, id := range og {
		if !slices.Contains(dest, id) {
			dest = append(dest, id)
		}

	}

	log.Printf("Apartment Buildings: %v", dest)

	return dest, nil
}

func (repo Repository) receiptBuildings() ([]string, error) {
	stmt := Receipts.SELECT(Receipts.BuildingID).DISTINCT().FROM(Receipts)

	var dest []string
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	og, err := repo.buildingIds()
	if err != nil {
		return nil, err
	}

	for _, id := range og {
		if !slices.Contains(dest, id) {
			dest = append(dest, id)
		}
	}

	log.Printf("Receipt Buildings: %v", dest)

	return dest, nil

}

func (repo Repository) receiptYears() ([]int16, error) {
	stmt := Receipts.SELECT(Receipts.Year).DISTINCT().FROM(Receipts)

	var dest []int16
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}

type Apt struct {
	Number string `json:"number"`
	Name   string `json:"name"`
}

func (repo Repository) receiptApts() (*string, error) {
	stmt := Apartments.SELECT(Apartments.BuildingID, Apartments.Number, Apartments.Name).FROM(Apartments).
		ORDER_BY(Apartments.BuildingID.ASC(), Apartments.Number.ASC())

	var array []model.Apartments
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &array)
	if err != nil {
		return nil, err
	}

	apts := make(map[string][]Apt)

	for _, apt := range array {
		apts[apt.BuildingID] = append(apts[apt.BuildingID], Apt{
			Number: apt.Number,
			Name:   apt.Name,
		})
	}

	bytes, err := json.Marshal(apts)
	if err != nil {
		return nil, err
	}

	base64Str := base64.URLEncoding.EncodeToString(bytes)
	return &base64Str, nil

}
