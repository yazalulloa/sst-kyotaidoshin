package expenses

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
	err := Expenses.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).FROM(Expenses).Query(db.GetDB().DB, &dest)
	if err != nil {
		return 0, err
	}
	return dest.Count, nil
}

func InsertBackup(array []model.Expenses) (int64, error) {
	if len(array) == 0 {
		return 0, nil
	}

	stmt := Expenses.INSERT(Expenses.BuildingID, Expenses.ReceiptID, Expenses.Description, Expenses.Amount, Expenses.Currency, Expenses.Type)

	for _, expense := range array {
		stmt = stmt.VALUES(expense.BuildingID, expense.ReceiptID, expense.Description, expense.Amount, expense.Currency, expense.Type)
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

func SelectByReceipt(receiptID int32) ([]model.Expenses, error) {
	var dest []model.Expenses
	err := Expenses.SELECT(Expenses.AllColumns).WHERE(Expenses.ReceiptID.EQ(sqlite.Int32(receiptID))).Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}
	return dest, nil
}

func deleteById(id int32) (int64, error) {
	stmt := Expenses.DELETE().WHERE(Expenses.ID.EQ(sqlite.Int32(id)))
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

func countByReceipt(receiptID int32) (int64, error) {
	stmt := Expenses.SELECT(sqlite.COUNT(Expenses.ID).AS("Count")).FROM(Expenses).
		WHERE(Expenses.ReceiptID.EQ(sqlite.Int32(receiptID)))

	var dest struct {
		Count int64
	}
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return 0, err
	}

	return dest.Count, nil
}

func update(expense model.Expenses) (int64, error) {
	stmt := Expenses.UPDATE(Expenses.Description, Expenses.Amount, Expenses.Currency, Expenses.Type).
		WHERE(Expenses.ID.EQ(sqlite.Int32(*expense.ID))).
		SET(expense.Description, expense.Amount, expense.Currency, expense.Type)

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

func insert(expense model.Expenses) (int64, error) {
	stmt := Expenses.INSERT(Expenses.BuildingID, Expenses.ReceiptID, Expenses.Description, Expenses.Amount, Expenses.Currency, Expenses.Type).
		VALUES(expense.BuildingID, expense.ReceiptID, expense.Description, expense.Amount, expense.Currency, expense.Type)

	result, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	lastInsertId, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastInsertId, nil
}
