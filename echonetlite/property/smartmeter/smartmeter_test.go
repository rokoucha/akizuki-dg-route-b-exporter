package smartmeter

import (
	"testing"
	"time"

	"github.com/rokoucha/akizuki-dg-route-b-exporter/echonetlite/property"
)

func TestHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy2(t *testing.T) {
	edt := []uint8{0x07, 0xEA, 0x0A, 0x18, 0x0C, 0x00, 0x02,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x02,
		0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x04,
	}
	h, err := NewHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy2(property.RawProperty{EPC: EPCHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy2, EDT: edt})
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Values) != 2 || h.Values[1].Normal != 3 || h.Values[1].Reverse != 4 {
		t.Errorf("values = %+v", h.Values)
	}
}

func TestDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved2ToSettable(t *testing.T) {
	d := &DayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved2{
		CollectedAt:        time.Date(2026, 10, 24, 12, 30, 0, 0, time.Local),
		CollectionSegments: 6,
	}
	want := []uint8{0x07, 0xEA, 0x0A, 0x18, 0x0C, 0x1E, 0x06}
	if got := d.ToSettable().EDT; string(got) != string(want) {
		t.Errorf("EDT = % X, want % X", got, want)
	}
}
