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

type Rates struct {
	ID           *int64 `sql:"primary_key"`
	FromCurrency string
	ToCurrency   string
	Rate         float64
	DateOfRate   time.Time
	Source       string
	DateOfFile   time.Time
	CreatedAt    *time.Time
	Hash         *int64
	Etag         *string
	LastModified *string
}
