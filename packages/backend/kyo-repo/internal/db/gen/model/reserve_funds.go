//
// Code generated by go-jet DO NOT EDIT.
//
// WARNING: Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated
//

package model

import (
	"time"
)

type ReserveFunds struct {
	ID            *int32 `sql:"primary_key"`
	BuildingID    string
	Name          string
	Fund          float64
	Expense       float64
	Pay           float64
	Active        bool
	Type          string
	ExpenseType   string
	AddToExpenses bool
	CreatedAt     *time.Time
	UpdatedAt     *time.Time
}
