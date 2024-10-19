package smartmeter

import (
	"encoding/binary"
	"time"

	"github.com/rokoucha/akizuki-dg-route-b-exporter/echonetlite/property"
)

const (
	ClassGroupCode = 0x02
	ClassCode      = 0x88
)

type EnergyValuePair struct {
	Normal  uint32
	Reverse uint32
}

// 動作状態
const EPCOperationStatus property.EPC = 0x80

type OperationStatus struct {
	Enabled bool
}

func (o *OperationStatus) ToSettable() property.RawProperty {
	value := uint8(0x31)
	if o.Enabled {
		value = 0x30
	}

	return property.RawProperty{
		EPC: EPCOperationStatus,
		EDT: []uint8{value},
	}
}

func NewOperationStatus(p property.RawProperty) (*OperationStatus, error) {
	if p.EPC != EPCOperationStatus {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) != 1 {
		return nil, property.ErrInvalidPropertyData
	}

	return &OperationStatus{
		Enabled: p.EDT[0] == 0x30,
	}, nil
}

// B ルート識別番号
const EPCRouteBIdentificationNumber property.EPC = 0xC0

type RouteBIdentificationNumber struct {
	ManufacturerCode string
	FreeArea         string
}

func (r *RouteBIdentificationNumber) ToSettable() property.RawProperty {
	return property.RawProperty{
		EPC: EPCRouteBIdentificationNumber,
		EDT: []uint8{},
	}
}

func NewRouteBIdentificationNumber(p property.RawProperty) (*RouteBIdentificationNumber, error) {
	if p.EPC != EPCRouteBIdentificationNumber {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) != 16 {
		return nil, property.ErrInvalidPropertyData
	}

	return &RouteBIdentificationNumber{
		ManufacturerCode: string(p.EDT[1:3]),
		FreeArea:         string(p.EDT[4:]),
	}, nil
}

// 1分積算電力量計測値（正方向、逆方向計測値）
const EPCOneMinuteMeasuredCumulativeAmountsOfElectricEnergyMeasured property.EPC = 0xD0

type OneMinuteMeasuredCumulativeAmountsOfElectricEnergyMeasured struct {
	MeasuredAt time.Time
	Normal     uint32
	Reverse    uint32
}

func (o *OneMinuteMeasuredCumulativeAmountsOfElectricEnergyMeasured) ToSettable() property.RawProperty {
	return property.RawProperty{
		EPC: EPCOneMinuteMeasuredCumulativeAmountsOfElectricEnergyMeasured,
		EDT: []uint8{},
	}
}

func NewOneMinuteMeasuredCumulativeAmountsOfElectricEnergyMeasured(p property.RawProperty) (*OneMinuteMeasuredCumulativeAmountsOfElectricEnergyMeasured, error) {
	if p.EPC != EPCOneMinuteMeasuredCumulativeAmountsOfElectricEnergyMeasured {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) != 15 {
		return nil, property.ErrInvalidPropertyData
	}

	return &OneMinuteMeasuredCumulativeAmountsOfElectricEnergyMeasured{
		MeasuredAt: property.BytesToDate(p.EDT[0:7]),
		Normal:     binary.BigEndian.Uint32(p.EDT[7:11]),
		Reverse:    binary.BigEndian.Uint32(p.EDT[11:15]),
	}, nil
}

// 係数
const EPCCoefficient property.EPC = 0xD3

type Coefficient struct {
	Value uint32
}

func (c *Coefficient) ToSettable() property.RawProperty {
	return property.RawProperty{
		EPC: EPCCoefficient,
		EDT: []uint8{},
	}
}

func NewCoefficient(p property.RawProperty) (*Coefficient, error) {
	if p.EPC != EPCCoefficient {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) != 4 {
		return nil, property.ErrInvalidPropertyData
	}

	return &Coefficient{
		Value: binary.BigEndian.Uint32(p.EDT),
	}, nil
}

// 積積算電力量有効桁数
const EPCNumberOfEffectiveDigitsForCumulativeAmountOfElectricEnergy property.EPC = 0xD7

type NumberOfEffectiveDigitsForCumulativeAmountOfElectricEnergy struct {
	Value uint8
}

func (n *NumberOfEffectiveDigitsForCumulativeAmountOfElectricEnergy) ToSettable() property.RawProperty {
	return property.RawProperty{
		EPC: EPCNumberOfEffectiveDigitsForCumulativeAmountOfElectricEnergy,
		EDT: []uint8{},
	}
}

func NewNumberOfEffectiveDigitsForCumulativeAmountOfElectricEnergy(p property.RawProperty) (*NumberOfEffectiveDigitsForCumulativeAmountOfElectricEnergy, error) {
	if p.EPC != EPCNumberOfEffectiveDigitsForCumulativeAmountOfElectricEnergy {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) != 1 {
		return nil, property.ErrInvalidPropertyData
	}

	return &NumberOfEffectiveDigitsForCumulativeAmountOfElectricEnergy{
		Value: p.EDT[0],
	}, nil
}

// 積算電力量計測値(正方向計測値)
const EPCMeasuredCumulativeAmountOfElectricEnergyNormalDirection property.EPC = 0xE0

type MeasuredCumulativeAmountOfElectricEnergyNormalDirection struct {
	Value uint32
}

func (m *MeasuredCumulativeAmountOfElectricEnergyNormalDirection) ToSettable() property.RawProperty {
	return property.RawProperty{
		EPC: EPCMeasuredCumulativeAmountOfElectricEnergyNormalDirection,
		EDT: []uint8{},
	}
}

func NewMeasuredCumulativeAmountOfElectricEnergyNormalDirection(p property.RawProperty) (*MeasuredCumulativeAmountOfElectricEnergyNormalDirection, error) {
	if p.EPC != EPCMeasuredCumulativeAmountOfElectricEnergyNormalDirection {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) != 4 {
		return nil, property.ErrInvalidPropertyData
	}

	return &MeasuredCumulativeAmountOfElectricEnergyNormalDirection{
		Value: binary.BigEndian.Uint32(p.EDT),
	}, nil
}

// 積算電力量単位（正方向、逆方向計測値）
const EPCUnitForCumulativeAmountOfElectricEnergy property.EPC = 0xE1

type UnitForCumulativeAmountOfElectricEnergy struct {
	Value float32
}

func (u *UnitForCumulativeAmountOfElectricEnergy) ToSettable() property.RawProperty {
	return property.RawProperty{
		EPC: EPCUnitForCumulativeAmountOfElectricEnergy,
		EDT: []uint8{},
	}
}

func NewUnitForCumulativeAmountOfElectricEnergy(p property.RawProperty) (*UnitForCumulativeAmountOfElectricEnergy, error) {
	if p.EPC != EPCUnitForCumulativeAmountOfElectricEnergy {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) != 1 {
		return nil, property.ErrInvalidPropertyData
	}

	var value float32
	switch p.EDT[0] {
	case 0x00:
		value = 1
	case 0x01:
		value = 0.1
	case 0x02:
		value = 0.01
	case 0x03:
		value = 0.001
	case 0x04:
		value = 0.0001
	case 0x0A:
		value = 10
	case 0x0B:
		value = 100
	case 0x0C:
		value = 1000
	case 0x0D:
		value = 10000
	}

	return &UnitForCumulativeAmountOfElectricEnergy{
		Value: value,
	}, nil
}

// 積算電力量計測値履歴１(正方向計測値)
const EPCHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1NormalDirection property.EPC = 0xE2

type HistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1NormalDirection struct {
	CollectedAt uint16
	Values      []uint32
}

func (h *HistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1NormalDirection) ToSettable() property.RawProperty {
	return property.RawProperty{
		EPC: EPCHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1NormalDirection,
		EDT: []uint8{},
	}
}

func NewHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1NormalDirection(p property.RawProperty) (*HistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1NormalDirection, error) {
	if p.EPC != EPCHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1NormalDirection {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) != 194 {
		return nil, property.ErrInvalidPropertyData
	}

	values := make([]uint32, 48)
	for i := 2; i < len(p.EDT); i += 4 {
		values[(i-2)/4] = binary.BigEndian.Uint32(p.EDT[i : i+4])
	}

	return &HistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1NormalDirection{
		CollectedAt: binary.BigEndian.Uint16(p.EDT[0:2]),
		Values:      values,
	}, nil
}

// 積算電力量計測値(逆方向計測値)
const EPCMeasuredCumulativeAmountOfElectricEnergyReverseDirection property.EPC = 0xE3

type MeasuredCumulativeAmountOfElectricEnergyReverseDirection struct {
	Value uint32
}

func (m *MeasuredCumulativeAmountOfElectricEnergyReverseDirection) ToSettable() property.RawProperty {
	return property.RawProperty{
		EPC: EPCMeasuredCumulativeAmountOfElectricEnergyReverseDirection,
		EDT: []uint8{},
	}
}

func NewMeasuredCumulativeAmountOfElectricEnergyReverseDirection(p property.RawProperty) (*MeasuredCumulativeAmountOfElectricEnergyReverseDirection, error) {
	if p.EPC != EPCMeasuredCumulativeAmountOfElectricEnergyReverseDirection {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) != 4 {
		return nil, property.ErrInvalidPropertyData
	}

	return &MeasuredCumulativeAmountOfElectricEnergyReverseDirection{
		Value: binary.BigEndian.Uint32(p.EDT),
	}, nil
}

// 積算電力量計測値履歴１(逆方向計測値)
const EPCHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1ReverseDirection property.EPC = 0xE4

type HistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1ReverseDirection struct {
	CollectedAt uint16
	Values      []uint32
}

func (h *HistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1ReverseDirection) ToSettable() property.RawProperty {
	return property.RawProperty{
		EPC: EPCHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1ReverseDirection,
		EDT: []uint8{},
	}
}

func NewHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1ReverseDirection(p property.RawProperty) (*HistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1ReverseDirection, error) {
	if p.EPC != EPCHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1ReverseDirection {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) != 194 {
		return nil, property.ErrInvalidPropertyData
	}

	values := make([]uint32, 48)
	for i := 2; i < len(p.EDT); i += 4 {
		values[(i-2)/4] = binary.BigEndian.Uint32(p.EDT[i : i+4])
	}

	return &HistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1ReverseDirection{
		CollectedAt: binary.BigEndian.Uint16(p.EDT[0:2]),
		Values:      values,
	}, nil
}

// 積算履歴収集日１
const EPCDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved1 property.EPC = 0xE5

type DayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved1 struct {
	CollectedAt uint8
}

func (d *DayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved1) ToSettable() property.RawProperty {
	return property.RawProperty{
		EPC: EPCDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved1,
		EDT: []uint8{d.CollectedAt},
	}
}

func NewDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved1(p property.RawProperty) (*DayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved1, error) {
	if p.EPC != EPCDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved1 {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) != 1 {
		return nil, property.ErrInvalidPropertyData
	}

	return &DayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved1{
		CollectedAt: p.EDT[0],
	}, nil
}

// 瞬時電力計測値
const EPCMeasuredInstantaneousElectricPower property.EPC = 0xE7

type MeasuredInstantaneousElectricPower struct {
	Value int32
}

func (m *MeasuredInstantaneousElectricPower) ToSettable() property.RawProperty {
	return property.RawProperty{
		EPC: EPCMeasuredInstantaneousElectricPower,
		EDT: []uint8{},
	}
}

func NewMeasuredInstantaneousElectricPower(p property.RawProperty) (*MeasuredInstantaneousElectricPower, error) {
	if p.EPC != EPCMeasuredInstantaneousElectricPower {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) != 4 {
		return nil, property.ErrInvalidPropertyData
	}

	return &MeasuredInstantaneousElectricPower{
		Value: int32(binary.BigEndian.Uint32(p.EDT)),
	}, nil
}

// 瞬時電流計測値
const EPCMeasuredInstantaneousCurrents property.EPC = 0xE8

type MeasuredInstantaneousCurrents struct {
	R float32
	T float32
}

func (m *MeasuredInstantaneousCurrents) ToSettable() property.RawProperty {
	return property.RawProperty{
		EPC: EPCMeasuredInstantaneousCurrents,
		EDT: []uint8{},
	}
}

func NewMeasuredInstantaneousCurrents(p property.RawProperty) (*MeasuredInstantaneousCurrents, error) {
	if p.EPC != EPCMeasuredInstantaneousCurrents {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) != 4 {
		return nil, property.ErrInvalidPropertyData
	}

	return &MeasuredInstantaneousCurrents{
		R: float32(binary.BigEndian.Uint16(p.EDT[0:2])) / 10,
		T: float32(binary.BigEndian.Uint16(p.EDT[2:4])) / 10,
	}, nil
}

// 定時積算電力量計測値(正方向計測値)
const EPCCumulativeAmountOfElectricEnergyMeasuredAtFixedTimeNormalDirection property.EPC = 0xEA

type CumulativeAmountOfElectricEnergyMeasuredAtFixedTimeNormalDirection struct {
	MeasuredAt time.Time
	Value      uint32
}

func (c *CumulativeAmountOfElectricEnergyMeasuredAtFixedTimeNormalDirection) ToSettable() property.RawProperty {
	return property.RawProperty{
		EPC: EPCCumulativeAmountOfElectricEnergyMeasuredAtFixedTimeNormalDirection,
		EDT: []uint8{},
	}
}

func NewCumulativeAmountOfElectricEnergyMeasuredAtFixedTimeNormalDirection(p property.RawProperty) (*CumulativeAmountOfElectricEnergyMeasuredAtFixedTimeNormalDirection, error) {
	if p.EPC != EPCCumulativeAmountOfElectricEnergyMeasuredAtFixedTimeNormalDirection {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) != 11 {
		return nil, property.ErrInvalidPropertyData
	}

	return &CumulativeAmountOfElectricEnergyMeasuredAtFixedTimeNormalDirection{
		MeasuredAt: property.BytesToDate(p.EDT[0:7]),
		Value:      binary.BigEndian.Uint32(p.EDT[7:11]),
	}, nil
}

// 定時積算電力量計測値(逆方向計測値)
const EPCCumulativeAmountOfElectricEnergyMeasuredAtFixedTimeReverseDirection property.EPC = 0xEB

type CumulativeAmountOfElectricEnergyMeasuredAtFixedTimeReverseDirection struct {
	MeasuredAt time.Time
	Value      uint32
}

func (c *CumulativeAmountOfElectricEnergyMeasuredAtFixedTimeReverseDirection) ToSettable() property.RawProperty {
	return property.RawProperty{
		EPC: EPCCumulativeAmountOfElectricEnergyMeasuredAtFixedTimeReverseDirection,
		EDT: []uint8{},
	}
}

func NewCumulativeAmountOfElectricEnergyMeasuredAtFixedTimeReverseDirection(p property.RawProperty) (*CumulativeAmountOfElectricEnergyMeasuredAtFixedTimeReverseDirection, error) {
	if p.EPC != EPCCumulativeAmountOfElectricEnergyMeasuredAtFixedTimeReverseDirection {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) != 11 {
		return nil, property.ErrInvalidPropertyData
	}

	return &CumulativeAmountOfElectricEnergyMeasuredAtFixedTimeReverseDirection{
		MeasuredAt: property.BytesToDate(p.EDT[0:7]),
		Value:      binary.BigEndian.Uint32(p.EDT[7:11]),
	}, nil
}

// 積算電力量計測値履歴２（正方向、逆方向計測値）
const EPCHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy2 property.EPC = 0xEC

type HistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy2 struct {
	CollectedAt        time.Time
	CollectionSegments uint8
	Values             []*EnergyValuePair
}

func (h *HistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy2) ToSettable() property.RawProperty {
	return property.RawProperty{
		EPC: EPCHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy2,
		EDT: []uint8{},
	}
}

func NewHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy2(p property.RawProperty) (*HistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy2, error) {
	if p.EPC != EPCHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy2 {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) < 7 {
		return nil, property.ErrInvalidPropertyData
	}

	segments := int(p.EDT[6])

	if segments != len(p.EDT[7:])/4 {
		return nil, property.ErrInvalidPropertyData
	}

	values := make([]*EnergyValuePair, segments)
	for i := 7; i < len(p.EDT); i += 8 {
		values[(i-7)/8] = &EnergyValuePair{
			Normal:  binary.BigEndian.Uint32(p.EDT[i : i+4]),
			Reverse: binary.BigEndian.Uint32(p.EDT[i+4 : i+8]),
		}
	}

	return &HistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy2{
		CollectedAt:        property.BytesToDate(p.EDT[0:6]),
		CollectionSegments: uint8(segments),
		Values:             values,
	}, nil
}

// 積算履歴収集日２
const EPCDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved2 property.EPC = 0xED

type DayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved2 struct {
	CollectedAt        time.Time
	CollectionSegments uint8
}

func (d *DayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved2) ToSettable() property.RawProperty {
	edt := []uint8{}
	binary.BigEndian.AppendUint16(edt, uint16(d.CollectedAt.Year()))
	edt = append(edt, uint8(d.CollectedAt.Month()), uint8(d.CollectedAt.Day()), uint8(d.CollectedAt.Hour()), uint8(d.CollectedAt.Minute()))
	edt = append(edt, d.CollectionSegments)

	return property.RawProperty{
		EPC: EPCDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved2,
		EDT: edt,
	}
}

func NewDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved2(p property.RawProperty) (*DayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved2, error) {
	if p.EPC != EPCDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved2 {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) != 7 {
		return nil, property.ErrInvalidPropertyData
	}

	return &DayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved2{
		CollectedAt:        property.BytesToDate(p.EDT[0:6]),
		CollectionSegments: p.EDT[6],
	}, nil
}

// 積算電力量計測値履歴３（正方向、逆方向計測値）
const EPCHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy3 property.EPC = 0xEE

type HistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy3 struct {
	CollectedAt        time.Time
	CollectionSegments uint8
	Values             []*EnergyValuePair
}

func (h *HistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy3) ToSettable() property.RawProperty {
	return property.RawProperty{
		EPC: EPCHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy3,
		EDT: []uint8{},
	}
}

func NewHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy3(p property.RawProperty) (*HistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy3, error) {
	if p.EPC != EPCHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy3 {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) < 7 {
		return nil, property.ErrInvalidPropertyData
	}

	segments := int(p.EDT[6])

	if segments != len(p.EDT[7:])/4 {
		return nil, property.ErrInvalidPropertyData
	}

	values := make([]*EnergyValuePair, segments)
	for i := 7; i < len(p.EDT); i += 8 {
		values[(i-7)/8] = &EnergyValuePair{
			Normal:  binary.BigEndian.Uint32(p.EDT[i : i+4]),
			Reverse: binary.BigEndian.Uint32(p.EDT[i+4 : i+8]),
		}
	}

	return &HistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy3{
		CollectedAt:        property.BytesToDate(p.EDT[0:6]),
		CollectionSegments: uint8(segments),
		Values:             values,
	}, nil
}

// 積算履歴収集日3
const EPCDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved3 property.EPC = 0xEF

type DayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved3 struct {
	CollectedAt        time.Time
	CollectionSegments uint8
}

func (d *DayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved3) ToSettable() property.RawProperty {
	edt := []uint8{}
	binary.BigEndian.AppendUint16(edt, uint16(d.CollectedAt.Year()))
	edt = append(edt, uint8(d.CollectedAt.Month()), uint8(d.CollectedAt.Day()), uint8(d.CollectedAt.Hour()), uint8(d.CollectedAt.Minute()))
	edt = append(edt, d.CollectionSegments)

	return property.RawProperty{
		EPC: EPCDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved3,
		EDT: edt,
	}
}

func NewDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved3(p property.RawProperty) (*DayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved3, error) {
	if p.EPC != EPCDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved3 {
		return nil, property.ErrPropertyMismatch
	}

	if len(p.EDT) != 7 {
		return nil, property.ErrInvalidPropertyData
	}

	return &DayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved3{
		CollectedAt:        property.BytesToDate(p.EDT[0:6]),
		CollectionSegments: p.EDT[6],
	}, nil
}

func ParseProperty(p property.RawProperty) (property.Property, error) {
	switch p.EPC {
	case EPCOperationStatus:
		return NewOperationStatus(p)
	case EPCRouteBIdentificationNumber:
		return NewRouteBIdentificationNumber(p)
	case EPCOneMinuteMeasuredCumulativeAmountsOfElectricEnergyMeasured:
		return NewOneMinuteMeasuredCumulativeAmountsOfElectricEnergyMeasured(p)
	case EPCCoefficient:
		return NewCoefficient(p)
	case EPCNumberOfEffectiveDigitsForCumulativeAmountOfElectricEnergy:
		return NewNumberOfEffectiveDigitsForCumulativeAmountOfElectricEnergy(p)
	case EPCMeasuredCumulativeAmountOfElectricEnergyNormalDirection:
		return NewMeasuredCumulativeAmountOfElectricEnergyNormalDirection(p)
	case EPCUnitForCumulativeAmountOfElectricEnergy:
		return NewUnitForCumulativeAmountOfElectricEnergy(p)
	case EPCHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1NormalDirection:
		return NewHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1NormalDirection(p)
	case EPCMeasuredCumulativeAmountOfElectricEnergyReverseDirection:
		return NewMeasuredCumulativeAmountOfElectricEnergyReverseDirection(p)
	case EPCHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1ReverseDirection:
		return NewHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy1ReverseDirection(p)
	case EPCDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved1:
		return NewDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved1(p)
	case EPCMeasuredInstantaneousElectricPower:
		return NewMeasuredInstantaneousElectricPower(p)
	case EPCMeasuredInstantaneousCurrents:
		return NewMeasuredInstantaneousCurrents(p)
	case EPCCumulativeAmountOfElectricEnergyMeasuredAtFixedTimeNormalDirection:
		return NewCumulativeAmountOfElectricEnergyMeasuredAtFixedTimeNormalDirection(p)
	case EPCCumulativeAmountOfElectricEnergyMeasuredAtFixedTimeReverseDirection:
		return NewCumulativeAmountOfElectricEnergyMeasuredAtFixedTimeReverseDirection(p)
	case EPCHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy2:
		return NewHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy2(p)
	case EPCDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved2:
		return NewDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved2(p)
	case EPCHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy3:
		return NewHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergy3(p)
	case EPCDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved3:
		return NewDayForWhichTheHistoricalDataOfMeasuredCumulativeAmountOfElectricEnergyIsToBeRetrieved3(p)
	default:
		return nil, property.ErrUnknownProperty
	}
}
