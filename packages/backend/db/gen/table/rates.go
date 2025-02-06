//
// Code generated by go-jet DO NOT EDIT.
//
// WARNING: Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated
//

package table

import (
	"github.com/go-jet/jet/v2/sqlite"
)

var Rates = newRatesTable("", "rates", "")

type ratesTable struct {
	sqlite.Table

	// Columns
	ID           sqlite.ColumnInteger
	FromCurrency sqlite.ColumnString
	ToCurrency   sqlite.ColumnString
	Rate         sqlite.ColumnFloat
	DateOfRate   sqlite.ColumnDate
	Source       sqlite.ColumnString
	DateOfFile   sqlite.ColumnTimestamp
	CreatedAt    sqlite.ColumnTimestamp
	Hash         sqlite.ColumnInteger
	Etag         sqlite.ColumnString
	LastModified sqlite.ColumnString

	AllColumns     sqlite.ColumnList
	MutableColumns sqlite.ColumnList
}

type RatesTable struct {
	ratesTable

	EXCLUDED ratesTable
}

// AS creates new RatesTable with assigned alias
func (a RatesTable) AS(alias string) *RatesTable {
	return newRatesTable(a.SchemaName(), a.TableName(), alias)
}

// Schema creates new RatesTable with assigned schema name
func (a RatesTable) FromSchema(schemaName string) *RatesTable {
	return newRatesTable(schemaName, a.TableName(), a.Alias())
}

// WithPrefix creates new RatesTable with assigned table prefix
func (a RatesTable) WithPrefix(prefix string) *RatesTable {
	return newRatesTable(a.SchemaName(), prefix+a.TableName(), a.TableName())
}

// WithSuffix creates new RatesTable with assigned table suffix
func (a RatesTable) WithSuffix(suffix string) *RatesTable {
	return newRatesTable(a.SchemaName(), a.TableName()+suffix, a.TableName())
}

func newRatesTable(schemaName, tableName, alias string) *RatesTable {
	return &RatesTable{
		ratesTable: newRatesTableImpl(schemaName, tableName, alias),
		EXCLUDED:   newRatesTableImpl("", "excluded", ""),
	}
}

func newRatesTableImpl(schemaName, tableName, alias string) ratesTable {
	var (
		IDColumn           = sqlite.IntegerColumn("id")
		FromCurrencyColumn = sqlite.StringColumn("from_currency")
		ToCurrencyColumn   = sqlite.StringColumn("to_currency")
		RateColumn         = sqlite.FloatColumn("rate")
		DateOfRateColumn   = sqlite.DateColumn("date_of_rate")
		SourceColumn       = sqlite.StringColumn("source")
		DateOfFileColumn   = sqlite.TimestampColumn("date_of_file")
		CreatedAtColumn    = sqlite.TimestampColumn("created_at")
		HashColumn         = sqlite.IntegerColumn("hash")
		EtagColumn         = sqlite.StringColumn("etag")
		LastModifiedColumn = sqlite.StringColumn("last_modified")
		allColumns         = sqlite.ColumnList{IDColumn, FromCurrencyColumn, ToCurrencyColumn, RateColumn, DateOfRateColumn, SourceColumn, DateOfFileColumn, CreatedAtColumn, HashColumn, EtagColumn, LastModifiedColumn}
		mutableColumns     = sqlite.ColumnList{FromCurrencyColumn, ToCurrencyColumn, RateColumn, DateOfRateColumn, SourceColumn, DateOfFileColumn, CreatedAtColumn, HashColumn, EtagColumn, LastModifiedColumn}
	)

	return ratesTable{
		Table: sqlite.NewTable(schemaName, tableName, alias, allColumns...),

		//Columns
		ID:           IDColumn,
		FromCurrency: FromCurrencyColumn,
		ToCurrency:   ToCurrencyColumn,
		Rate:         RateColumn,
		DateOfRate:   DateOfRateColumn,
		Source:       SourceColumn,
		DateOfFile:   DateOfFileColumn,
		CreatedAt:    CreatedAtColumn,
		Hash:         HashColumn,
		Etag:         EtagColumn,
		LastModified: LastModifiedColumn,

		AllColumns:     allColumns,
		MutableColumns: mutableColumns,
	}
}
