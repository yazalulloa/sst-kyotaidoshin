package buildings

import (
	"db"
	"db/gen/model"
	. "db/gen/table"
	"github.com/go-jet/jet/v2/sqlite"
	"kyotaidoshin/util"
)

func getTotalCount() (int64, error) {
	var dest struct {
		Count int64
	}
	err := Buildings.SELECT(sqlite.COUNT(Buildings.ID).AS("Count")).FROM(Buildings).Query(db.GetDB().DB, &dest)
	if err != nil {
		return 0, err
	}
	return dest.Count, nil
}

func idExists(id string) (bool, error) {

	stmt := Buildings.SELECT(Buildings.ID).FROM(Buildings).WHERE(Buildings.ID.EQ(sqlite.String(id)))
	var dest []string
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return false, err
	}
	return len(dest) > 0, nil
}

func selectList(req RequestQuery) ([]model.Buildings, error) {
	condition := sqlite.Bool(true)

	if req.LastCreatedAt != nil {
		date := *req.LastCreatedAt
		dateTime := sqlite.DateTime(date.Year(), date.Month(), date.Day(), date.Hour(), date.Minute(), date.Second())
		condition = condition.AND(Buildings.CreatedAt.LT(dateTime))
	}

	stmt := Buildings.SELECT(Buildings.AllColumns).FROM(Buildings).WHERE(condition)
	if req.SortOrder == util.SortOrderTypeASC {
		stmt = stmt.ORDER_BY(Buildings.CreatedAt.ASC())
	} else {
		stmt = stmt.ORDER_BY(Buildings.CreatedAt.DESC())
	}

	if req.Limit > 0 {
		stmt = stmt.LIMIT(int64(req.Limit))
	}

	var list []model.Buildings
	err := stmt.Query(db.GetDB().DB, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func deleteById(id string) (int64, error) {
	stmt := Buildings.DELETE().WHERE(Buildings.ID.EQ(sqlite.String(id)))
	result, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, err
}

func insert(building model.Buildings) error {
	stmt := Buildings.INSERT(Buildings.ID, Buildings.Name, Buildings.Rif, Buildings.MainCurrency, Buildings.DebtCurrency, Buildings.CurrenciesToShowAmountToPay, Buildings.FixedPay, Buildings.FixedPayAmount, Buildings.RoundUpPayments, Buildings.EmailConfig).
		VALUES(building.ID, building.Name, building.Rif, building.MainCurrency, building.DebtCurrency, building.CurrenciesToShowAmountToPay, building.FixedPay, building.FixedPayAmount, building.RoundUpPayments, building.EmailConfig)

	_, err := stmt.Exec(db.GetDB().DB)
	return err
}

func insertBackup(buildings []model.Buildings) (int64, error) {
	stmt := Buildings.INSERT(Buildings.ID, Buildings.Name, Buildings.Rif, Buildings.MainCurrency, Buildings.DebtCurrency, Buildings.CurrenciesToShowAmountToPay, Buildings.FixedPay, Buildings.FixedPayAmount, Buildings.RoundUpPayments, Buildings.EmailConfig).
		ON_CONFLICT().DO_NOTHING()

	for _, building := range buildings {
		stmt = stmt.VALUES(building.ID, building.Name, building.Rif, building.MainCurrency, building.DebtCurrency, building.CurrenciesToShowAmountToPay, building.FixedPay, building.FixedPayAmount, building.RoundUpPayments, building.EmailConfig)
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

func update(building model.Buildings) error {

	stmt := Buildings.UPDATE(
		Buildings.Name, Buildings.Rif, Buildings.MainCurrency, Buildings.DebtCurrency, Buildings.CurrenciesToShowAmountToPay, Buildings.FixedPay, Buildings.FixedPayAmount, Buildings.RoundUpPayments, Buildings.EmailConfig).
		WHERE(Buildings.ID.EQ(sqlite.String(building.ID))).
		SET(building.Name, building.Rif, building.MainCurrency, building.DebtCurrency, building.CurrenciesToShowAmountToPay, building.FixedPay, building.FixedPayAmount, building.RoundUpPayments, building.EmailConfig)

	_, err := stmt.Exec(db.GetDB().DB)
	return err

}

func selectById(id string) (*model.Buildings, error) {
	stmt := Buildings.SELECT(Buildings.AllColumns).FROM(Buildings).WHERE(Buildings.ID.EQ(sqlite.String(id)))
	var dest model.Buildings
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}
	return &dest, nil
}
