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

type ExtraCharges struct {
	ID              *int32 `sql:"primary_key"`
	BuildingID      string
	ParentReference string
	Type            string
	Description     string
	Amount          float64
	Currency        string
	Active          bool
	Apartments      string
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
}
