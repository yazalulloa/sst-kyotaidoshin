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

var ExtraCharges = newExtraChargesTable("", "extra_charges", "")

type extraChargesTable struct {
	sqlite.Table

	// Columns
	ID              sqlite.ColumnInteger
	BuildingID      sqlite.ColumnString
	ParentReference sqlite.ColumnString
	Type            sqlite.ColumnString
	Description     sqlite.ColumnString
	Amount          sqlite.ColumnFloat
	Currency        sqlite.ColumnString
	Active          sqlite.ColumnBool
	CreatedAt       sqlite.ColumnTimestamp
	UpdatedAt       sqlite.ColumnTimestamp

	AllColumns     sqlite.ColumnList
	MutableColumns sqlite.ColumnList
}

type ExtraChargesTable struct {
	extraChargesTable

	EXCLUDED extraChargesTable
}

// AS creates new ExtraChargesTable with assigned alias
func (a ExtraChargesTable) AS(alias string) *ExtraChargesTable {
	return newExtraChargesTable(a.SchemaName(), a.TableName(), alias)
}

// Schema creates new ExtraChargesTable with assigned schema name
func (a ExtraChargesTable) FromSchema(schemaName string) *ExtraChargesTable {
	return newExtraChargesTable(schemaName, a.TableName(), a.Alias())
}

// WithPrefix creates new ExtraChargesTable with assigned table prefix
func (a ExtraChargesTable) WithPrefix(prefix string) *ExtraChargesTable {
	return newExtraChargesTable(a.SchemaName(), prefix+a.TableName(), a.TableName())
}

// WithSuffix creates new ExtraChargesTable with assigned table suffix
func (a ExtraChargesTable) WithSuffix(suffix string) *ExtraChargesTable {
	return newExtraChargesTable(a.SchemaName(), a.TableName()+suffix, a.TableName())
}

func newExtraChargesTable(schemaName, tableName, alias string) *ExtraChargesTable {
	return &ExtraChargesTable{
		extraChargesTable: newExtraChargesTableImpl(schemaName, tableName, alias),
		EXCLUDED:          newExtraChargesTableImpl("", "excluded", ""),
	}
}

func newExtraChargesTableImpl(schemaName, tableName, alias string) extraChargesTable {
	var (
		IDColumn              = sqlite.IntegerColumn("id")
		BuildingIDColumn      = sqlite.StringColumn("building_id")
		ParentReferenceColumn = sqlite.StringColumn("parent_reference")
		TypeColumn            = sqlite.StringColumn("type")
		DescriptionColumn     = sqlite.StringColumn("description")
		AmountColumn          = sqlite.FloatColumn("amount")
		CurrencyColumn        = sqlite.StringColumn("currency")
		ActiveColumn          = sqlite.BoolColumn("active")
		CreatedAtColumn       = sqlite.TimestampColumn("created_at")
		UpdatedAtColumn       = sqlite.TimestampColumn("updated_at")
		allColumns            = sqlite.ColumnList{IDColumn, BuildingIDColumn, ParentReferenceColumn, TypeColumn, DescriptionColumn, AmountColumn, CurrencyColumn, ActiveColumn, CreatedAtColumn, UpdatedAtColumn}
		mutableColumns        = sqlite.ColumnList{BuildingIDColumn, ParentReferenceColumn, TypeColumn, DescriptionColumn, AmountColumn, CurrencyColumn, ActiveColumn, CreatedAtColumn, UpdatedAtColumn}
	)

	return extraChargesTable{
		Table: sqlite.NewTable(schemaName, tableName, alias, allColumns...),

		//Columns
		ID:              IDColumn,
		BuildingID:      BuildingIDColumn,
		ParentReference: ParentReferenceColumn,
		Type:            TypeColumn,
		Description:     DescriptionColumn,
		Amount:          AmountColumn,
		Currency:        CurrencyColumn,
		Active:          ActiveColumn,
		CreatedAt:       CreatedAtColumn,
		UpdatedAt:       UpdatedAtColumn,

		AllColumns:     allColumns,
		MutableColumns: mutableColumns,
	}
}
