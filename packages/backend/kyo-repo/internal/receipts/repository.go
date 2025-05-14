package receipts

import (
	"context"
	"github.com/go-jet/jet/v2/sqlite"
	"github.com/yaz/kyo-repo/internal/db"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	. "github.com/yaz/kyo-repo/internal/db/gen/table"
	"github.com/yaz/kyo-repo/internal/util"
	"strings"
)

type Repository struct {
	ctx context.Context
}

func NewRepository(ctx context.Context) Repository {
	return Repository{ctx: ctx}
}

func (repo Repository) getTotalCount() (int64, error) {
	var dest struct {
		Count int64
	}
	err := Receipts.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).FROM(Receipts).QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return 0, err
	}
	return dest.Count, nil
}

func (repo Repository) InsertBulk(receipt []model.Receipts) (int64, error) {
	stmt := Receipts.INSERT(Receipts.ID, Receipts.BuildingID, Receipts.Year, Receipts.Month, Receipts.Date, Receipts.RateID, Receipts.Sent, Receipts.LastSent, Receipts.CreatedAt)

	for _, r := range receipt {
		stmt = stmt.VALUES(r.ID, r.BuildingID, r.Year, r.Month, r.Date, r.RateID, r.Sent, r.LastSent, r.CreatedAt)
	}

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

func (repo Repository) getQueryCount(requestQuery RequestQuery) (*int64, error) {
	condition := queryCondition(requestQuery)
	if condition == nil {
		return nil, nil
	}

	stmt := Receipts.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).FROM(Receipts).WHERE(*condition)

	//log.Printf("CountQuery : %v\n", stmt.DebugSql())
	var dest struct {
		Count int64
	}

	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return &dest.Count, nil
}

func queryCondition(requestQuery RequestQuery) *sqlite.BoolExpression {
	condition := sqlite.Bool(true)
	isThereAnyCondition := false

	if len(requestQuery.Buildings) > 0 {
		//log.Printf("Buildings : %v\n", requestQuery.buildings)
		var buildingIds []sqlite.Expression
		for _, buildingId := range requestQuery.Buildings {
			buildingId = strings.TrimSpace(buildingId)
			if buildingId == "" {
				continue
			}

			buildingIds = append(buildingIds, sqlite.String(buildingId))
		}

		if len(buildingIds) > 0 {
			condition = condition.AND(Receipts.BuildingID.IN(buildingIds...))
			isThereAnyCondition = true
		}
	}

	if len(requestQuery.Months) > 0 {
		var months []sqlite.Expression
		for _, month := range requestQuery.Months {
			months = append(months, sqlite.Int16(month))
		}
		condition = condition.AND(Receipts.Month.IN(months...))
		isThereAnyCondition = true
	}

	if len(requestQuery.Years) > 0 {
		var years []sqlite.Expression
		for _, year := range requestQuery.Years {
			years = append(years, sqlite.Int16(year))
		}
		condition = condition.AND(Receipts.Year.IN(years...))
		isThereAnyCondition = true
	}

	if !isThereAnyCondition {
		return nil
	}

	return &condition
}

func (repo Repository) selectList(requestQuery RequestQuery) ([]model.Receipts, error) {
	condition := sqlite.Bool(true)

	sortOrder := util.SortOrderTypeDESC

	if requestQuery.SortOrder == util.SortOrderTypeASC {
		sortOrder = util.SortOrderTypeASC
	}

	if requestQuery.LastId != "" {
		if sortOrder == util.SortOrderTypeDESC {
			condition = condition.AND(Receipts.ID.LT(sqlite.String(requestQuery.LastId)))
		} else {
			condition = condition.AND(Receipts.ID.GT(sqlite.String(requestQuery.LastId)))
		}

	}

	commonQueryCondition := queryCondition(requestQuery)
	if commonQueryCondition != nil {
		condition = condition.AND(*commonQueryCondition)
	}

	stmt := Receipts.SELECT(Receipts.AllColumns).FROM(Receipts).WHERE(condition).
		LIMIT(requestQuery.Limit)

	if sortOrder == util.SortOrderTypeDESC {
		stmt = stmt.ORDER_BY(Receipts.ID.DESC())
	} else {
		stmt = stmt.ORDER_BY(Receipts.ID.ASC())
	}

	var list []model.Receipts

	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (repo Repository) selectById(id string) (*model.Receipts, error) {
	stmt := Receipts.SELECT(Receipts.AllColumns).FROM(Receipts).WHERE(Receipts.ID.EQ(sqlite.String(id)))

	var dest model.Receipts
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return &dest, nil
}

func (repo Repository) selectByIdWithRate(id string) (*struct {
	model.Receipts
	model.Rates
}, error) {

	var dest struct {
		model.Receipts
		model.Rates
	}

	stmt := Receipts.SELECT(Receipts.AllColumns, Rates.AllColumns).
		FROM(Receipts.LEFT_JOIN(Rates, Receipts.RateID.EQ(Rates.ID))).WHERE(Receipts.ID.EQ(sqlite.String(id)))

	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return &dest, nil
}

func (repo Repository) update(receipt model.Receipts) (int64, error) {
	stmt := Receipts.UPDATE(Receipts.Year, Receipts.Month, Receipts.Date, Receipts.RateID).
		WHERE(Receipts.ID.EQ(sqlite.String(receipt.ID))).
		SET(receipt.Year, receipt.Month, receipt.Date, receipt.RateID)

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

func (repo Repository) UpdateLastSent(id string) (int64, error) {
	stmt := Receipts.UPDATE(Receipts.Sent, Receipts.LastSent).WHERE(Receipts.ID.EQ(sqlite.String(id))).
		SET(sqlite.Bool(true), sqlite.DATETIME("now"))

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

func (repo Repository) deleteById(id string) (int64, error) {
	stmt := Receipts.DELETE().WHERE(Receipts.ID.EQ(sqlite.String(id)))

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

func (repo Repository) insert(receipt model.Receipts) (int64, error) {
	stmt := Receipts.INSERT(Receipts.ID, Receipts.BuildingID, Receipts.Year, Receipts.Month, Receipts.Date, Receipts.RateID, Receipts.Sent).
		VALUES(receipt.ID, receipt.BuildingID, receipt.Year, receipt.Month, receipt.Date, receipt.RateID, receipt.Sent)

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

//
//func selectIds(lastId string, limit int64) ([]string, error) {
//	condition := sqlite.Bool(true)
//
//	if lastId != "" {
//		condition = condition.AND(Receipts.ID.LT(sqlite.String(lastId)))
//	}
//
//	stmt := Receipts.SELECT(Receipts.ID).FROM(Receipts).WHERE(condition).ORDER_BY(Receipts.ID.ASC()).LIMIT(limit)
//
//	var dest []string
//	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
//	if err != nil {
//		return nil, err
//	}
//	return dest, nil
//}
//
//func selectRecords(lastId string, limit int64) ([]struct {
//	model.Receipts
//	Expenses     []model.Expenses
//	ExtraCharges []model.ExtraCharges
//	Debts        []model.Debts
//}, error) {
//
//	condition := sqlite.Bool(true)
//
//	if lastId != "" {
//		condition = condition.AND(Receipts.ID.LT(sqlite.String(lastId)))
//	}
//
//	var dest []struct {
//		model.Receipts
//		Expenses     []model.Expenses
//		ExtraCharges []model.ExtraCharges
//		Debts        []model.Debts
//	}
//
//	paginatedReceipts := sqlite.CTE("paginated_receipts")
//	idColumn := sqlite.StringColumn("id").From(paginatedReceipts)
//
//	stmt := sqlite.WITH(paginatedReceipts.AS(
//		Receipts.SELECT(sqlite.StringColumn("id")).FROM(Receipts).
//			WHERE(condition).
//			ORDER_BY(Receipts.ID.ASC()).
//			LIMIT(limit),
//	))(
//		paginatedReceipts.SELECT(Receipts.AllColumns, Expenses.AllColumns, ExtraCharges.AllColumns, Debts.AllColumns).
//			FROM(paginatedReceipts.
//				INNER_JOIN(Receipts, idColumn.EQ(Receipts.ID)).
//				LEFT_JOIN(Expenses, Expenses.BuildingID.EQ(Receipts.BuildingID).
//					AND(Expenses.ReceiptID.EQ(Receipts.ID))).
//				LEFT_JOIN(ExtraCharges, ExtraCharges.BuildingID.EQ(Receipts.BuildingID).
//					AND(ExtraCharges.ParentReference.EQ(Receipts.ID)).
//					AND(ExtraCharges.Type.EQ(sqlite.String(extraCharges.TypeReceipt)))).
//				LEFT_JOIN(Debts, Debts.BuildingID.EQ(Receipts.BuildingID).
//					AND(Debts.ReceiptID.EQ(Receipts.ID))),
//			).
//			ORDER_BY(Receipts.ID.ASC()),
//	)
//
//	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
//	if err != nil {
//		return nil, err
//	}
//
//	return dest, nil
//}
