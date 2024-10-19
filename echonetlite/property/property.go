package property

import (
	"encoding/binary"
	"errors"
	"time"
)

var (
	ErrInvalidPropertyData = errors.New("invalid property data")
	ErrPropertyMismatch    = errors.New("property mismatch")
	ErrUnknownProperty     = errors.New("unknown property")
)

// ECHONET Liteプロパティ
type EPC uint8

// ECHONET Liteプロパティ
type RawProperty struct {
	// ECHONET Liteプロパティ
	EPC EPC
	// ECHONET Liteプロパティ値データ
	EDT []uint8
}

func BytesToDate(b []uint8) time.Time {
	if len(b) != 6 && len(b) != 7 {
		panic("invalid date data")
	}

	var s int
	if len(b) == 7 {
		s = int(b[6])
	}

	return time.Date(
		int(binary.BigEndian.Uint16(b[0:2])),
		time.Month(b[2]),
		int(b[3]),
		int(b[4]),
		int(b[5]),
		s,
		0,
		time.Local,
	)
}

type Property interface {
	ToSettable() RawProperty
}

type UnknownProperty struct {
	RawProperty
}

func (u *UnknownProperty) ToSettable() RawProperty {
	return u.RawProperty
}

func NewUnknownProperty(property RawProperty) *UnknownProperty {
	return &UnknownProperty{
		RawProperty: property,
	}
}
