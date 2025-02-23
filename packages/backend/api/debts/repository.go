package debts

import (
	"db"
	"db/gen/model"
	. "db/gen/table"
	"github.com/go-jet/jet/v2/sqlite"
	"log"
)

func count() (int64, error) {
	var dest struct {
		Count int64
	}
	err := Debts.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).FROM(Debts).Query(db.GetDB().DB, &dest)
	if err != nil {
		return 0, err
	}
	return dest.Count, nil
}

func InsertBackup(array []model.Debts) (int64, error) {
	if len(array) == 0 {
		return 0, nil
	}

	stmt := Debts.INSERT(Debts.BuildingID, Debts.ReceiptID, Debts.AptNumber, Debts.Receipts, Debts.Amount, Debts.Months, Debts.PreviousPaymentAmount, Debts.PreviousPaymentAmountCurrency)

	for _, debt := range array {
		if debt.PreviousPaymentAmountCurrency == "" {
			debt.PreviousPaymentAmountCurrency = "VED"
		}

		stmt = stmt.VALUES(debt.BuildingID, debt.ReceiptID, debt.AptNumber, debt.Receipts, debt.Amount, debt.Months, debt.PreviousPaymentAmount, debt.PreviousPaymentAmountCurrency)
	}

	res, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		log.Printf("Error inserting array: %v\n", err)
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func SelectByBuildingReceipt(buildingId string, receiptId int32) ([]model.Debts, error) {

	stmt := Debts.SELECT(Debts.AllColumns).
		WHERE(Debts.BuildingID.EQ(sqlite.String(buildingId)).
			AND(Debts.ReceiptID.EQ(sqlite.Int32(receiptId))))

	var dest []model.Debts
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}
