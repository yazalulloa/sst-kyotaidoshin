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

var Debts = newDebtsTable("", "debts", "")

type debtsTable struct {
	sqlite.Table

	// Columns
	BuildingID                    sqlite.ColumnString
	ReceiptID                     sqlite.ColumnString
	AptNumber                     sqlite.ColumnString
	Receipts                      sqlite.ColumnInteger
	Amount                        sqlite.ColumnFloat
	Months                        sqlite.ColumnString
	PreviousPaymentAmount         sqlite.ColumnFloat
	PreviousPaymentAmountCurrency sqlite.ColumnString

	AllColumns     sqlite.ColumnList
	MutableColumns sqlite.ColumnList
}

type DebtsTable struct {
	debtsTable

	EXCLUDED debtsTable
}

// AS creates new DebtsTable with assigned alias
func (a DebtsTable) AS(alias string) *DebtsTable {
	return newDebtsTable(a.SchemaName(), a.TableName(), alias)
}

// Schema creates new DebtsTable with assigned schema name
func (a DebtsTable) FromSchema(schemaName string) *DebtsTable {
	return newDebtsTable(schemaName, a.TableName(), a.Alias())
}

// WithPrefix creates new DebtsTable with assigned table prefix
func (a DebtsTable) WithPrefix(prefix string) *DebtsTable {
	return newDebtsTable(a.SchemaName(), prefix+a.TableName(), a.TableName())
}

// WithSuffix creates new DebtsTable with assigned table suffix
func (a DebtsTable) WithSuffix(suffix string) *DebtsTable {
	return newDebtsTable(a.SchemaName(), a.TableName()+suffix, a.TableName())
}

func newDebtsTable(schemaName, tableName, alias string) *DebtsTable {
	return &DebtsTable{
		debtsTable: newDebtsTableImpl(schemaName, tableName, alias),
		EXCLUDED:   newDebtsTableImpl("", "excluded", ""),
	}
}

func newDebtsTableImpl(schemaName, tableName, alias string) debtsTable {
	var (
		BuildingIDColumn                    = sqlite.StringColumn("building_id")
		ReceiptIDColumn                     = sqlite.StringColumn("receipt_id")
		AptNumberColumn                     = sqlite.StringColumn("apt_number")
		ReceiptsColumn                      = sqlite.IntegerColumn("receipts")
		AmountColumn                        = sqlite.FloatColumn("amount")
		MonthsColumn                        = sqlite.StringColumn("months")
		PreviousPaymentAmountColumn         = sqlite.FloatColumn("previous_payment_amount")
		PreviousPaymentAmountCurrencyColumn = sqlite.StringColumn("previous_payment_amount_currency")
		allColumns                          = sqlite.ColumnList{BuildingIDColumn, ReceiptIDColumn, AptNumberColumn, ReceiptsColumn, AmountColumn, MonthsColumn, PreviousPaymentAmountColumn, PreviousPaymentAmountCurrencyColumn}
		mutableColumns                      = sqlite.ColumnList{ReceiptsColumn, AmountColumn, MonthsColumn, PreviousPaymentAmountColumn, PreviousPaymentAmountCurrencyColumn}
	)

	return debtsTable{
		Table: sqlite.NewTable(schemaName, tableName, alias, allColumns...),

		//Columns
		BuildingID:                    BuildingIDColumn,
		ReceiptID:                     ReceiptIDColumn,
		AptNumber:                     AptNumberColumn,
		Receipts:                      ReceiptsColumn,
		Amount:                        AmountColumn,
		Months:                        MonthsColumn,
		PreviousPaymentAmount:         PreviousPaymentAmountColumn,
		PreviousPaymentAmountCurrency: PreviousPaymentAmountCurrencyColumn,

		AllColumns:     allColumns,
		MutableColumns: mutableColumns,
	}
}
