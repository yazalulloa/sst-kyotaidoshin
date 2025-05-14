package reserveFunds

import (
	"context"
	"github.com/go-jet/jet/v2/sqlite"
	"github.com/yaz/kyo-repo/internal/db"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	. "github.com/yaz/kyo-repo/internal/db/gen/table"
)

type Repository struct {
	ctx context.Context
}

func NewRepository(ctx context.Context) Repository {
	return Repository{ctx: ctx}
}

func (repo Repository) InsertBackup(array []model.ReserveFunds) (int64, error) {
	stmt := ReserveFunds.INSERT(ReserveFunds.BuildingID, ReserveFunds.Name, ReserveFunds.Fund, ReserveFunds.Expense, ReserveFunds.Pay, ReserveFunds.Active, ReserveFunds.Type, ReserveFunds.ExpenseType, ReserveFunds.AddToExpenses).
		ON_CONFLICT().DO_NOTHING()

	for _, reserveFund := range array {
		stmt = stmt.VALUES(reserveFund.BuildingID, reserveFund.Name, reserveFund.Fund, reserveFund.Expense, reserveFund.Pay, reserveFund.Active, reserveFund.Type, reserveFund.ExpenseType, reserveFund.AddToExpenses)
	}

	result, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil

}

func (repo Repository) selectById(id int32) (*model.ReserveFunds, error) {
	stmt := ReserveFunds.SELECT(ReserveFunds.AllColumns).FROM(ReserveFunds).WHERE(ReserveFunds.ID.EQ(sqlite.Int32(id)))
	var dest model.ReserveFunds
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return &dest, nil
}

func (repo Repository) SelectByBuilding(buildingId string) ([]model.ReserveFunds, error) {
	stmt := ReserveFunds.SELECT(ReserveFunds.AllColumns).FROM(ReserveFunds).WHERE(ReserveFunds.BuildingID.EQ(sqlite.String(buildingId)))
	var dest []model.ReserveFunds
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}

func (repo Repository) CountByBuilding(buildingId string) (int64, error) {
	stmt := ReserveFunds.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).FROM(ReserveFunds).WHERE(ReserveFunds.BuildingID.EQ(sqlite.String(buildingId)))
	var dest struct {
		Count int64
	}
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return 0, err
	}

	return dest.Count, nil
}

func (repo Repository) DeleteByBuilding(buildingId string) (int64, error) {
	stmt := ReserveFunds.DELETE().WHERE(ReserveFunds.BuildingID.EQ(sqlite.String(buildingId)))
	result, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func (repo Repository) deleteById(id int32) (int64, error) {
	stmt := ReserveFunds.DELETE().WHERE(ReserveFunds.ID.EQ(sqlite.Int32(id)))
	result, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func (repo Repository) update(reserveFund model.ReserveFunds) (int64, error) {
	stmt := ReserveFunds.UPDATE(ReserveFunds.Name, ReserveFunds.Fund, ReserveFunds.Expense, ReserveFunds.Pay, ReserveFunds.Active, ReserveFunds.Type, ReserveFunds.ExpenseType, ReserveFunds.AddToExpenses).
		WHERE(ReserveFunds.ID.EQ(sqlite.Int32(*reserveFund.ID))).
		SET(reserveFund.Name, reserveFund.Fund, reserveFund.Expense, reserveFund.Pay, reserveFund.Active, reserveFund.Type, reserveFund.ExpenseType, reserveFund.AddToExpenses)
	result, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func (repo Repository) insert(reserveFund model.ReserveFunds) (int64, error) {
	stmt := ReserveFunds.INSERT(ReserveFunds.BuildingID, ReserveFunds.Name, ReserveFunds.Fund, ReserveFunds.Expense, ReserveFunds.Pay, ReserveFunds.Active, ReserveFunds.Type, ReserveFunds.ExpenseType, ReserveFunds.AddToExpenses).
		VALUES(reserveFund.BuildingID, reserveFund.Name, reserveFund.Fund, reserveFund.Expense, reserveFund.Pay, reserveFund.Active, reserveFund.Type, reserveFund.ExpenseType, reserveFund.AddToExpenses)

	result, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	lastInsertId, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastInsertId, nil
}
