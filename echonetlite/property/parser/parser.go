package parser

import (
	"github.com/rokoucha/akizuki-dg-route-b-exporter/echonetlite/property"
	"github.com/rokoucha/akizuki-dg-route-b-exporter/echonetlite/property/smartmeter"
)

func ParseProperty(object [3]uint8, epc uint8, edt []uint8) (property.Property, error) {
	r := property.RawProperty{
		EPC: property.EPC(epc),
		EDT: edt,
	}

	switch [2]uint8{object[0], object[1]} {
	case [2]uint8{smartmeter.ClassGroupCode, smartmeter.ClassCode}:
		return smartmeter.ParseProperty(r)
	}

	return property.NewUnknownProperty(r), nil
}
