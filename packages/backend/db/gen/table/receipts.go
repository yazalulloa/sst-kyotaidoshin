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

var Receipts = newReceiptsTable("", "receipts", "")

type receiptsTable struct {
	sqlite.Table

	// Columns
	ID         sqlite.ColumnString
	BuildingID sqlite.ColumnString
	Year       sqlite.ColumnInteger
	Month      sqlite.ColumnInteger
	Date       sqlite.ColumnDate
	RateID     sqlite.ColumnInteger
	Sent       sqlite.ColumnBool
	LastSent   sqlite.ColumnTimestamp
	CreatedAt  sqlite.ColumnTimestamp
	UpdatedAt  sqlite.ColumnTimestamp

	AllColumns     sqlite.ColumnList
	MutableColumns sqlite.ColumnList
}

type ReceiptsTable struct {
	receiptsTable

	EXCLUDED receiptsTable
}

// AS creates new ReceiptsTable with assigned alias
func (a ReceiptsTable) AS(alias string) *ReceiptsTable {
	return newReceiptsTable(a.SchemaName(), a.TableName(), alias)
}

// Schema creates new ReceiptsTable with assigned schema name
func (a ReceiptsTable) FromSchema(schemaName string) *ReceiptsTable {
	return newReceiptsTable(schemaName, a.TableName(), a.Alias())
}

// WithPrefix creates new ReceiptsTable with assigned table prefix
func (a ReceiptsTable) WithPrefix(prefix string) *ReceiptsTable {
	return newReceiptsTable(a.SchemaName(), prefix+a.TableName(), a.TableName())
}

// WithSuffix creates new ReceiptsTable with assigned table suffix
func (a ReceiptsTable) WithSuffix(suffix string) *ReceiptsTable {
	return newReceiptsTable(a.SchemaName(), a.TableName()+suffix, a.TableName())
}

func newReceiptsTable(schemaName, tableName, alias string) *ReceiptsTable {
	return &ReceiptsTable{
		receiptsTable: newReceiptsTableImpl(schemaName, tableName, alias),
		EXCLUDED:      newReceiptsTableImpl("", "excluded", ""),
	}
}

func newReceiptsTableImpl(schemaName, tableName, alias string) receiptsTable {
	var (
		IDColumn         = sqlite.StringColumn("id")
		BuildingIDColumn = sqlite.StringColumn("building_id")
		YearColumn       = sqlite.IntegerColumn("year")
		MonthColumn      = sqlite.IntegerColumn("month")
		DateColumn       = sqlite.DateColumn("date")
		RateIDColumn     = sqlite.IntegerColumn("rate_id")
		SentColumn       = sqlite.BoolColumn("sent")
		LastSentColumn   = sqlite.TimestampColumn("last_sent")
		CreatedAtColumn  = sqlite.TimestampColumn("created_at")
		UpdatedAtColumn  = sqlite.TimestampColumn("updated_at")
		allColumns       = sqlite.ColumnList{IDColumn, BuildingIDColumn, YearColumn, MonthColumn, DateColumn, RateIDColumn, SentColumn, LastSentColumn, CreatedAtColumn, UpdatedAtColumn}
		mutableColumns   = sqlite.ColumnList{IDColumn, BuildingIDColumn, YearColumn, MonthColumn, DateColumn, RateIDColumn, SentColumn, LastSentColumn, CreatedAtColumn, UpdatedAtColumn}
	)

	return receiptsTable{
		Table: sqlite.NewTable(schemaName, tableName, alias, allColumns...),

		//Columns
		ID:         IDColumn,
		BuildingID: BuildingIDColumn,
		Year:       YearColumn,
		Month:      MonthColumn,
		Date:       DateColumn,
		RateID:     RateIDColumn,
		Sent:       SentColumn,
		LastSent:   LastSentColumn,
		CreatedAt:  CreatedAtColumn,
		UpdatedAt:  UpdatedAtColumn,

		AllColumns:     allColumns,
		MutableColumns: mutableColumns,
	}
}
