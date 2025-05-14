package debts

import (
	"context"
	"github.com/go-jet/jet/v2/sqlite"
	"github.com/yaz/kyo-repo/internal/db"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	. "github.com/yaz/kyo-repo/internal/db/gen/table"
	"github.com/yaz/kyo-repo/internal/util"
	"log"
)

type Repository struct {
	ctx context.Context
}

func NewRepository(ctx context.Context) Repository {
	return Repository{ctx: ctx}
}

func (repo Repository) InsertBulk(array []model.Debts) (int64, error) {
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

	res, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
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

func (repo Repository) SelectByBuildingReceipt(buildingId string, receiptId string) ([]model.Debts, error) {

	stmt := Debts.SELECT(Debts.AllColumns).
		WHERE(Debts.BuildingID.EQ(sqlite.String(buildingId)).
			AND(Debts.ReceiptID.EQ(sqlite.String(receiptId))))

	var dest []model.Debts
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}

func (repo Repository) SelectByReceipts(ids []string) ([]model.Debts, error) {
	receipts := make([]sqlite.Expression, len(ids))
	for i, p := range ids {
		receipts[i] = sqlite.String(p)
	}
	var dest []model.Debts
	err := Debts.SELECT(Debts.AllColumns).WHERE(Debts.ReceiptID.IN(receipts...)).QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}
	return dest, nil

}

func (repo Repository) DeleteByReceipt(buildingId string, receiptId string) (int64, error) {
	stmt := Debts.DELETE().WHERE(Debts.BuildingID.EQ(sqlite.String(buildingId)).AND(Debts.ReceiptID.EQ(sqlite.String(receiptId))))
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

func (repo Repository) update(debt model.Debts) (int64, error) {
	stmt := Debts.UPDATE(Debts.Receipts, Debts.Amount, Debts.Months, Debts.PreviousPaymentAmount, Debts.PreviousPaymentAmountCurrency).
		WHERE(Debts.BuildingID.EQ(sqlite.String(debt.BuildingID)).AND(Debts.ReceiptID.EQ(sqlite.String(debt.ReceiptID))).AND(Debts.AptNumber.EQ(sqlite.String(debt.AptNumber)))).
		SET(debt.Receipts, debt.Amount, debt.Months, debt.PreviousPaymentAmount, debt.PreviousPaymentAmountCurrency)

	res, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
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
