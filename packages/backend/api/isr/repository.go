package isr

import (
	"db"
	. "db/gen/table"
)

func apartmentBuildings() ([]string, error) {
	stmt := Apartments.SELECT(Apartments.BuildingID).DISTINCT().FROM(Apartments)
	var dest []string
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}

func getDistinctCurrencies() ([]string, error) {
	stmt := Rates.SELECT(Rates.FromCurrency).DISTINCT().FROM(Rates)
	var dest []string
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}

func receiptBuildings() ([]string, error) {
	stmt := Receipts.SELECT(Receipts.BuildingID).DISTINCT().FROM(Receipts)

	var buildingIds []string
	err := stmt.Query(db.GetDB().DB, &buildingIds)
	if err != nil {
		return nil, err
	}

	return buildingIds, nil

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
