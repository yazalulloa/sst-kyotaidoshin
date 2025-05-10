package buildings

import (
	"github.com/go-jet/jet/v2/sqlite"
	"github.com/yaz/kyo-repo/internal/db"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	. "github.com/yaz/kyo-repo/internal/db/gen/table"
	"github.com/yaz/kyo-repo/internal/extraCharges"
	"github.com/yaz/kyo-repo/internal/util"
)

func getTotalCount() (int64, error) {
	var dest struct {
		Count int64
	}
	err := Buildings.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).FROM(Buildings).Query(db.GetDB().DB, &dest)
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

func selectList(req RequestQuery) ([]struct {
	model.Buildings
	AptCount int64
}, error) {
	condition := sqlite.Bool(true)

	if req.LastCreatedAt != nil {
		date := *req.LastCreatedAt
		dateTime := sqlite.DateTime(date.Year(), date.Month(), date.Day(), date.Hour(), date.Minute(), date.Second())
		condition = condition.AND(Buildings.CreatedAt.LT(dateTime))
	}

	var dest []struct {
		model.Buildings
		AptCount int64
	}

	stmt := Buildings.SELECT(Buildings.AllColumns, sqlite.COUNT(sqlite.STAR).AS("apt_count")).
		FROM(Buildings.LEFT_JOIN(Apartments, Apartments.BuildingID.EQ(Buildings.ID))).
		WHERE(condition).GROUP_BY(Buildings.ID)

	if req.SortOrder == util.SortOrderTypeASC {
		stmt = stmt.ORDER_BY(Buildings.CreatedAt.ASC())
	} else {
		stmt = stmt.ORDER_BY(Buildings.CreatedAt.DESC())
	}

	if req.Limit > 0 {
		stmt = stmt.LIMIT(int64(req.Limit))
	}

	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}

//type ExtraChargesArray []model.ExtraCharges
//type ReserveFundsArray []model.ReserveFunds

func selectRecords(lastId string, limit int64) ([]struct {
	model.Buildings
	ExtraCharges []model.ExtraCharges
	ReserveFunds []model.ReserveFunds
}, error) {
	condition := sqlite.Bool(true)

	if lastId != "" {
		condition = condition.AND(Buildings.ID.GT(sqlite.String(lastId)))
	}

	var dest []struct {
		model.Buildings
		ExtraCharges []model.ExtraCharges
		ReserveFunds []model.ReserveFunds
	}

	paginatedBuildings := sqlite.CTE("paginated_buildings")
	idColumn := sqlite.StringColumn("id").From(paginatedBuildings)

	stmt := sqlite.WITH(paginatedBuildings.AS(
		Buildings.SELECT(sqlite.StringColumn("id")).FROM(Buildings).
			WHERE(condition).
			ORDER_BY(Buildings.ID.ASC()).
			LIMIT(limit),
	))(
		paginatedBuildings.SELECT(Buildings.AllColumns, ExtraCharges.AllColumns, ReserveFunds.AllColumns).
			FROM(paginatedBuildings.
				INNER_JOIN(Buildings, idColumn.EQ(Buildings.ID)).
				LEFT_JOIN(ExtraCharges, ExtraCharges.BuildingID.EQ(Buildings.ID).
					AND(ExtraCharges.ParentReference.EQ(Buildings.ID)).
					AND(ExtraCharges.Type.EQ(sqlite.String(extraCharges.TypeBuilding)))).
				LEFT_JOIN(ReserveFunds, ReserveFunds.BuildingID.EQ(Buildings.ID)),
			).
			ORDER_BY(Buildings.ID.ASC()),
	)

	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
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
	stmt := Buildings.INSERT(Buildings.ID, Buildings.Name, Buildings.Rif, Buildings.MainCurrency, Buildings.DebtCurrency, Buildings.CurrenciesToShowAmountToPay, Buildings.DebtsCurrenciesToShow, Buildings.FixedPay, Buildings.FixedPayAmount, Buildings.RoundUpPayments, Buildings.EmailConfig).
		VALUES(building.ID, building.Name, building.Rif, building.MainCurrency, building.DebtCurrency, building.CurrenciesToShowAmountToPay, building.DebtsCurrenciesToShow, building.FixedPay, building.FixedPayAmount, building.RoundUpPayments, building.EmailConfig)

	_, err := stmt.Exec(db.GetDB().DB)
	return err
}

func insertBackup(buildings []model.Buildings) (int64, error) {
	stmt := Buildings.INSERT(Buildings.ID, Buildings.Name, Buildings.Rif, Buildings.MainCurrency, Buildings.DebtCurrency, Buildings.CurrenciesToShowAmountToPay, Buildings.DebtsCurrenciesToShow, Buildings.FixedPay, Buildings.FixedPayAmount, Buildings.RoundUpPayments, Buildings.EmailConfig).
		ON_CONFLICT().DO_NOTHING()

	for _, building := range buildings {
		stmt = stmt.VALUES(building.ID, building.Name, building.Rif, building.MainCurrency, building.DebtCurrency, building.CurrenciesToShowAmountToPay, building.DebtsCurrenciesToShow, building.FixedPay, building.FixedPayAmount, building.RoundUpPayments, building.EmailConfig)
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
		Buildings.Name, Buildings.Rif, Buildings.MainCurrency, Buildings.DebtCurrency, Buildings.CurrenciesToShowAmountToPay, Buildings.DebtsCurrenciesToShow, Buildings.FixedPay, Buildings.FixedPayAmount, Buildings.RoundUpPayments, Buildings.EmailConfig).
		WHERE(Buildings.ID.EQ(sqlite.String(building.ID))).
		SET(building.Name, building.Rif, building.MainCurrency, building.DebtCurrency, building.CurrenciesToShowAmountToPay, building.DebtsCurrenciesToShow, building.FixedPay, building.FixedPayAmount, building.RoundUpPayments, building.EmailConfig)

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

func SelectIds() ([]string, error) {
	stmt := Buildings.SELECT(Buildings.ID).FROM(Buildings).ORDER_BY(Buildings.ID.ASC())
	var dest []string
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}
	return dest, nil
}

func SelectById(id string) (*model.Buildings, error) {

	stmt := Buildings.SELECT(Buildings.AllColumns).FROM(Buildings).WHERE(Buildings.ID.EQ(sqlite.String(id)))

	var dest model.Buildings
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return &dest, nil
}
