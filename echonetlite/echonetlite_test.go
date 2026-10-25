package echonetlite

import (
	"testing"

	"github.com/rokoucha/akizuki-dg-route-b-exporter/echonetlite/property"
	"github.com/rokoucha/akizuki-dg-route-b-exporter/echonetlite/property/smartmeter"
)

func TestNewFrameMultipleProperties(t *testing.T) {
	// Get_Res: E7=1234W, E8=R12.3A/T-4.5A, E1=0.1kWh
	b := []uint8{
		0x10, 0x81, 0x00, 0x2A, 0x02, 0x88, 0x01, 0x05, 0xFF, 0x01, 0x72, 0x03,
		0xE7, 0x04, 0x00, 0x00, 0x04, 0xD2,
		0xE8, 0x04, 0x00, 0x7B, 0xFF, 0xD3,
		0xE1, 0x01, 0x01,
	}
	f, err := NewFrame(b)
	if err != nil {
		t.Fatal(err)
	}
	if f.TID != [2]uint8{0x00, 0x2A} || f.EDATA.ESV != ESVGet_Res || len(f.EDATA.Properties) != 3 {
		t.Fatalf("unexpected frame: %+v", f)
	}
	if v := f.EDATA.Properties[0].(*smartmeter.MeasuredInstantaneousElectricPower).Value; v != 1234 {
		t.Errorf("power = %d", v)
	}
	c := f.EDATA.Properties[1].(*smartmeter.MeasuredInstantaneousCurrents)
	if c.R == nil || *c.R != 12.3 || c.T == nil || *c.T != -4.5 {
		t.Errorf("currents = %+v", c)
	}
	if v := f.EDATA.Properties[2].(*smartmeter.UnitForCumulativeAmountOfElectricEnergy).Value; v != 0.1 {
		t.Errorf("unit = %v", v)
	}
}

func TestNewFrameGetSNA(t *testing.T) {
	// Get_SNA: D3 は未対応 (PDC=0)、E1=1kWh
	b := []uint8{
		0x10, 0x81, 0x00, 0x01, 0x02, 0x88, 0x01, 0x05, 0xFF, 0x01, 0x52, 0x02,
		0xD3, 0x00,
		0xE1, 0x01, 0x00,
	}
	f, err := NewFrame(b)
	if err != nil {
		t.Fatal(err)
	}
	u, ok := f.EDATA.Properties[0].(*property.UnknownProperty)
	if !ok || u.EPC != smartmeter.EPCCoefficient {
		t.Errorf("properties[0] = %#v", f.EDATA.Properties[0])
	}
	if v := f.EDATA.Properties[1].(*smartmeter.UnitForCumulativeAmountOfElectricEnergy).Value; v != 1 {
		t.Errorf("unit = %v", v)
	}
}

func TestNewFrameTruncated(t *testing.T) {
	for _, b := range [][]uint8{
		{0x10, 0x81},
		{0x10, 0x81, 0x00, 0x01, 0x02, 0x88, 0x01, 0x05, 0xFF, 0x01, 0x72, 0x01, 0xE7, 0x04, 0x00},
		{0x10, 0x81, 0x00, 0x01, 0x02, 0x88, 0x01, 0x05, 0xFF, 0x01, 0x72, 0x02, 0xE7, 0x04, 0x00, 0x00, 0x00, 0x01},
	} {
		if _, err := NewFrame(b); err == nil {
			t.Errorf("NewFrame(% X) succeeded", b)
		}
	}
}

func TestFrameBytesRoundTrip(t *testing.T) {
	f := &Frame{
		EHD1: EHD1ECHONETLite,
		EHD2: EHD2SpecifiedMessageFormat,
		TID:  [2]uint8{0x12, 0x34},
		EDATA: Data{
			SEOJ:       [3]uint8{0x05, 0xFF, 0x01},
			DEOJ:       [3]uint8{0x02, 0x88, 0x01},
			ESV:        ESVGet,
			Properties: []property.Property{&smartmeter.MeasuredInstantaneousElectricPower{}, &smartmeter.Coefficient{}},
		},
	}
	want := []uint8{0x10, 0x81, 0x12, 0x34, 0x05, 0xFF, 0x01, 0x02, 0x88, 0x01, 0x62, 0x02, 0xE7, 0x00, 0xD3, 0x00}
	if got := f.Bytes(); string(got) != string(want) {
		t.Errorf("Bytes() = % X, want % X", got, want)
	}
}
