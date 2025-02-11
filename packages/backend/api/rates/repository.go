package rates

import (
	"db"
	"db/gen/model"
	. "db/gen/table"
	"github.com/go-jet/jet/v2/sqlite"
	"kyotaidoshin/api"
	"time"
)

func queryCondition(rateQuery *RateQuery) (sqlite.BoolExpression, bool) {
	condition := sqlite.Bool(true)
	justTrue := true

	if rateQuery.DateOfRate != nil {
		condition = condition.AND(Rates.DateOfRate.LT_EQ(sqlite.Date(rateQuery.DateOfRate.Date())))
		justTrue = false
	}

	if len(rateQuery.Currencies) > 0 {
		var sqlIDs []sqlite.Expression
		for _, str := range rateQuery.Currencies {
			sqlIDs = append(sqlIDs, sqlite.String(str))
		}

		condition = condition.AND(Rates.FromCurrency.IN(sqlIDs...))
		justTrue = false
	}

	return condition, justTrue
}

func GetRates(rateQuery RateQuery) ([]model.Rates, error) {
	condition, _ := queryCondition(&rateQuery)

	if rateQuery.LastId > 0 {
		condition = condition.AND(Rates.ID.LT(sqlite.Int64(rateQuery.LastId)))
	}

	stmt := Rates.SELECT(Rates.AllColumns).FROM(Rates).WHERE(condition).LIMIT(int64(rateQuery.Limit))

	if rateQuery.SortOrder == "ASC" {
		stmt = stmt.ORDER_BY(Rates.ID.ASC())
	} else {
		stmt = stmt.ORDER_BY(Rates.ID.DESC())
	}

	var rates []model.Rates
	err := stmt.Query(db.GetDB().DB, &rates)
	if err != nil {
		return nil, err
	}

	return rates, nil
}

func GetTotalCount() (int64, error) {
	var dest struct {
		Count int64
	}
	err := Rates.SELECT(sqlite.COUNT(Rates.ID).AS("Count")).FROM(Rates).Query(db.GetDB().DB, &dest)
	if err != nil {
		return 0, err
	}
	return dest.Count, nil
}

func getQueryCount(rateQuery RateQuery) (*int64, error) {
	condition, justTrue := queryCondition(&rateQuery)
	if justTrue {
		return nil, nil
	}

	stmt := Rates.SELECT(sqlite.COUNT(Rates.ID).AS("Count")).FROM(Rates).WHERE(condition)
	var dest struct {
		Count int64
	}
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}
	return &dest.Count, nil
}

func checkRateExist(rate model.Rates) (bool, error) {
	if true {
		stmt := Rates.SELECT(Rates.ID.AS("ID")).FROM(Rates).
			WHERE(Rates.ID.EQ(sqlite.Int64(*rate.ID)))

		var dest []struct {
			ID *int32
		}

		err := stmt.Query(db.GetDB().DB, &dest)
		if err != nil {

			if api.ErrNoRows.Error() == err.Error() {
				return false, nil
			}

			return false, err
		}

		return len(dest) > 0, nil
	}

	stmt := Rates.SELECT(Rates.ID.AS("ID")).FROM(Rates).
		WHERE(Rates.FromCurrency.EQ(sqlite.String(rate.FromCurrency)).
			AND(Rates.ToCurrency.EQ(sqlite.String(rate.ToCurrency))).
			AND(Rates.Rate.EQ(sqlite.Float(rate.Rate))).
			AND(Rates.DateOfRate.EQ(sqlite.Date(rate.DateOfRate.Date()))))

	//log.Printf("Query: %s", stmt.DebugSql())

	var dest []struct {
		ID *int32
	}

	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {

		if api.ErrNoRows.Error() == err.Error() {
			return false, nil
		}

		return false, err
	}

	return len(dest) > 0, nil
}

func Insert(rates []model.Rates) (int64, error) {

	stmt := Rates.INSERT(Rates.ID, Rates.FromCurrency, Rates.ToCurrency, Rates.Rate, Rates.DateOfRate, Rates.Source,
		Rates.DateOfFile, Rates.Etag, Rates.LastModified).
		ON_CONFLICT().DO_NOTHING()
	//.MODELS(rates)
	for _, rate := range rates {
		stmt = stmt.VALUES(rate.ID, rate.FromCurrency, rate.ToCurrency, rate.Rate, rate.DateOfRate.Format(time.DateOnly), rate.Source, rate.DateOfFile, rate.Etag, rate.LastModified)
	}

	res, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	//id, err := res.LastInsertId()
	//if err != nil {
	//	return 0, err
	//}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func getCurrencies() ([]string, error) {
	stmt := Rates.SELECT(Rates.FromCurrency).DISTINCT().FROM(Rates)
	var dest []string
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}

func deleteRateById(id int64) (int64, error) {
	stmt := Rates.DELETE().WHERE(Rates.ID.EQ(sqlite.Int64(id)))
	res, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, nil
	}

	rowsAffected, err := res.RowsAffected()

	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}
