package reserveFunds

import (
	"db"
	"db/gen/model"
	. "db/gen/table"
	"github.com/go-jet/jet/v2/sqlite"
)

func InsertBackup(array []model.ReserveFunds) (int64, error) {
	stmt := ReserveFunds.INSERT(ReserveFunds.BuildingID, ReserveFunds.Name, ReserveFunds.Fund, ReserveFunds.Expense, ReserveFunds.Pay, ReserveFunds.Active, ReserveFunds.Type, ReserveFunds.ExpenseType, ReserveFunds.AddToExpenses).
		ON_CONFLICT().DO_NOTHING()

	for _, reserveFund := range array {
		stmt = stmt.VALUES(reserveFund.BuildingID, reserveFund.Name, reserveFund.Fund, reserveFund.Expense, reserveFund.Pay, reserveFund.Active, reserveFund.Type, reserveFund.ExpenseType, reserveFund.AddToExpenses)
	}

	result, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil

}

func SelectByBuilding(buildingId string) ([]model.ReserveFunds, error) {
	stmt := ReserveFunds.SELECT(ReserveFunds.AllColumns).FROM(ReserveFunds).WHERE(ReserveFunds.BuildingID.EQ(sqlite.String(buildingId)))
	var dest []model.ReserveFunds
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}

func DeleteByBuilding(buildingId string) (int64, error) {
	stmt := ReserveFunds.DELETE().WHERE(ReserveFunds.BuildingID.EQ(sqlite.String(buildingId)))
	result, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func deleteById(id int32) (int64, error) {
	stmt := ReserveFunds.DELETE().WHERE(ReserveFunds.ID.EQ(sqlite.Int32(id)))
	result, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}
