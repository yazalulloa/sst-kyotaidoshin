package expenses

import (
	"context"
	"github.com/go-jet/jet/v2/sqlite"
	"github.com/yaz/kyo-repo/internal/db"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	. "github.com/yaz/kyo-repo/internal/db/gen/table"
	"log"
)

type Repository struct {
	ctx context.Context
}

func NewRepository(ctx context.Context) Repository {
	return Repository{ctx: ctx}
}

func (repo Repository) InsertBulk(array []model.Expenses) (int64, error) {

	if len(array) == 0 {
		return 0, nil
	}

	stmt := Expenses.INSERT(Expenses.BuildingID, Expenses.ReceiptID, Expenses.Description, Expenses.Amount, Expenses.Currency, Expenses.Type)

	for _, expense := range array {
		stmt = stmt.VALUES(expense.BuildingID, expense.ReceiptID, expense.Description, expense.Amount, expense.Currency, expense.Type)
	}

	res, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
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

func (repo Repository) SelectByReceipt(receiptID string) ([]model.Expenses, error) {
	var dest []model.Expenses
	err := Expenses.SELECT(Expenses.AllColumns).WHERE(Expenses.ReceiptID.EQ(sqlite.String(receiptID))).QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}
	return dest, nil
}

func (repo Repository) SelectByReceipts(ids []string) ([]model.Expenses, error) {
	receipts := make([]sqlite.Expression, len(ids))
	for i, p := range ids {
		receipts[i] = sqlite.String(p)
	}
	var dest []model.Expenses
	err := Expenses.SELECT(Expenses.AllColumns).WHERE(Expenses.ReceiptID.IN(receipts...)).QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}
	return dest, nil
}

func (repo Repository) DeleteByReceipt(receiptID string) (int64, error) {
	stmt := Expenses.DELETE().WHERE(Expenses.ReceiptID.EQ(sqlite.String(receiptID)))
	res, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

func (repo Repository) deleteById(id int32) (int64, error) {
	stmt := Expenses.DELETE().WHERE(Expenses.ID.EQ(sqlite.Int32(id)))
	res, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func (repo Repository) countByReceipt(receiptID string) (int64, error) {
	stmt := Expenses.SELECT(
		sqlite.COUNT(sqlite.STAR).
			//Expenses.ID).
			AS("Count")).FROM(Expenses).
		WHERE(Expenses.ReceiptID.EQ(sqlite.String(receiptID)))

	var dest struct {
		Count int64
	}
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return 0, err
	}

	return dest.Count, nil
}

func (repo Repository) update(expense model.Expenses) (int64, error) {
	stmt := Expenses.UPDATE(Expenses.Description, Expenses.Amount, Expenses.Currency, Expenses.Type).
		WHERE(Expenses.ID.EQ(sqlite.Int32(*expense.ID))).
		SET(expense.Description, expense.Amount, expense.Currency, expense.Type)

	res, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func (repo Repository) insert(expense model.Expenses) (int64, error) {
	stmt := Expenses.INSERT(Expenses.BuildingID, Expenses.ReceiptID, Expenses.Description, Expenses.Amount, Expenses.Currency, Expenses.Type).
		VALUES(expense.BuildingID, expense.ReceiptID, expense.Description, expense.Amount, expense.Currency, expense.Type)

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
