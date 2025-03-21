package isr

import (
	"db"
	"db/gen/model"
	. "db/gen/table"
	"log"
	"slices"
)

func getDistinctCurrencies() ([]string, error) {
	stmt := Rates.SELECT(Rates.FromCurrency).DISTINCT().FROM(Rates)
	var dest []string
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}

func buildingIds() ([]string, error) {
	stmt := Buildings.SELECT(Buildings.ID).FROM(Buildings)

	var buildings []model.Buildings
	err := stmt.Query(db.GetDB().DB, &buildings)
	if err != nil {
		return nil, err
	}

	array := make([]string, len(buildings))
	for i, building := range buildings {
		array[i] = building.ID
	}

	return array, nil
}

func apartmentBuildings() ([]string, error) {
	stmt := Apartments.SELECT(Apartments.BuildingID).DISTINCT().FROM(Apartments)
	var dest []string
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	og, err := buildingIds()
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

func receiptBuildings() ([]string, error) {
	stmt := Receipts.SELECT(Receipts.BuildingID).DISTINCT().FROM(Receipts)

	var dest []string
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	og, err := buildingIds()
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

func receiptYears() ([]int16, error) {
	stmt := Receipts.SELECT(Receipts.Year).DISTINCT().FROM(Receipts)

	var dest []int16
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}
