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

type Receipts struct {
	ID         string
	BuildingID string
	Year       int16
	Month      int16
	Date       time.Time
	RateID     int64
	Sent       bool
	LastSent   *time.Time
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
}
