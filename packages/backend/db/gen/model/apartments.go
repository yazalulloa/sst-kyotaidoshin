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

type Apartments struct {
	BuildingID string `sql:"primary_key"`
	Number     string `sql:"primary_key"`
	Name       string
	IDDoc      *string
	Aliquot    float64
	Emails     *string
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
}
