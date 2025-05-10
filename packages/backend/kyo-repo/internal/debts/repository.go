package debts

import (
	"github.com/go-jet/jet/v2/sqlite"
	"kyo-repo/internal/db"
	"kyo-repo/internal/db/gen/model"
	. "kyo-repo/internal/db/gen/table"
	"kyo-repo/internal/util"
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

func InsertBulk(array []model.Debts) (int64, error) {
	if len(array) == 0 {
		return 0, nil
	}

	stmt := Debts.INSERT(Debts.BuildingID, Debts.ReceiptID, Debts.AptNumber, Debts.Receipts, Debts.Amount, Debts.Months, Debts.PreviousPaymentAmount, Debts.PreviousPaymentAmountCurrency)

	for _, debt := range array {
		if debt.PreviousPaymentAmountCurrency == "" {
			debt.PreviousPaymentAmountCurrency = util.VED.Name()
		}

		stmt = stmt.VALUES(debt.BuildingID, debt.ReceiptID, debt.AptNumber, debt.Receipts, debt.Amount, debt.Months, debt.PreviousPaymentAmount, debt.PreviousPaymentAmountCurrency)
	}

	res, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		log.Printf("Error inserting array: %v\n%s", err, stmt.DebugSql())
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func SelectByBuildingReceipt(buildingId string, receiptId string) ([]model.Debts, error) {

	stmt := Debts.SELECT(Debts.AllColumns).
		WHERE(Debts.BuildingID.EQ(sqlite.String(buildingId)).
			AND(Debts.ReceiptID.EQ(sqlite.String(receiptId))))

	var dest []model.Debts
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}

func SelectByReceipts(ids []string) ([]model.Debts, error) {
	receipts := make([]sqlite.Expression, len(ids))
	for i, p := range ids {
		receipts[i] = sqlite.String(p)
	}
	var dest []model.Debts
	err := Debts.SELECT(Debts.AllColumns).WHERE(Debts.ReceiptID.IN(receipts...)).Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}
	return dest, nil

}

func DeleteByReceipt(buildingId string, receiptId string) (int64, error) {
	stmt := Debts.DELETE().WHERE(Debts.BuildingID.EQ(sqlite.String(buildingId)).AND(Debts.ReceiptID.EQ(sqlite.String(receiptId))))
	res, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

func update(debt model.Debts) (int64, error) {
	stmt := Debts.UPDATE(Debts.Receipts, Debts.Amount, Debts.Months, Debts.PreviousPaymentAmount, Debts.PreviousPaymentAmountCurrency).
		WHERE(Debts.BuildingID.EQ(sqlite.String(debt.BuildingID)).AND(Debts.ReceiptID.EQ(sqlite.String(debt.ReceiptID))).AND(Debts.AptNumber.EQ(sqlite.String(debt.AptNumber)))).
		SET(debt.Receipts, debt.Amount, debt.Months, debt.PreviousPaymentAmount, debt.PreviousPaymentAmountCurrency)

	res, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		log.Printf("Error updating debt: %v\n", err)
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}
