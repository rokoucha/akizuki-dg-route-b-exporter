package echonetlite

import (
	"errors"
	"slices"

	"github.com/rokoucha/akizuki-dg-route-b-exporter/echonetlite/property"
	"github.com/rokoucha/akizuki-dg-route-b-exporter/echonetlite/property/parser"
)

var (
	ErrInvalidPacket = errors.New("invalid packet")
)

// ECHONET Lite ヘッダ１
type EHD1 uint8

const (
	// ECHONET Lite
	EHD1ECHONETLite EHD1 = 0x10
)

// ECHONET Lite ヘッダ２
type EHD2 uint8

const (
	// 形式1（規定電文形式）
	EHD2SpecifiedMessageFormat EHD2 = 0x81
	// 形式2（任意電文形式）
	EHD2ArbitraryMessageFormat EHD2 = 0x82
)

// ECHONET Lite サービス
type ESV uint8

// 要求用 ESV コード
const (
	// プロパティ値書き込み要求（応答不要）
	ESVSetI ESV = 0x60
	// プロパティ値書き込み要求（応答要）
	ESVSetC ESV = 0x61
	// プロパティ値読み出し要求
	ESVGet ESV = 0x62
	// プロパティ値通知要求
	ESVINF_REQ ESV = 0x63
)

// 応答・通知用 ESV コード
const (
	// プロパティ値書き込み応答
	ESVSet_Res ESV = 0x71
	// プロパティ値読み出し応答
	ESVGet_Res ESV = 0x72
	// プロパティ値通知
	ESVINF ESV = 0x73
	// プロパティ値通知（応答要）
	ESVINFC ESV = 0x74
	// プロパティ値通知応答
	ESVINFC_Res ESV = 0x7a
	// プロパティ値書き込み・読み出し応答
	ESVSetGet_Res ESV = 0x7e
)

// 不可応答用 ESV コード
const (
	// プロパティ値書き込み要求不可応答（応答不要）
	ESVSetI_SNA ESV = 0x50
	// プロパティ値書き込み要求不可応答（応答要）
	ESVSetC_SNA ESV = 0x51
	// プロパティ値読み出し不可応答
	ESVGet_SNA ESV = 0x52
	// プロパティ値通知不可応答
	ESVINF_SNA ESV = 0x53
	// プロパティ値書き込み・読み出し不可応答
	ESVSetGet_SNA ESV = 0x5e
)

// ECHONET Liteデータ
type Data struct {
	// 送信元ECHONET Liteオブジェクト指定
	SEOJ [3]uint8
	// 相手先ECHONET Liteオブジェクト指定
	DEOJ [3]uint8
	// ECHONET Lite サービス
	ESV ESV
	// ECHONET プロパティ
	Properties []property.Property
}

// ECHONET Lite フレーム
type Frame struct {
	// ECHONET Lite ヘッダ１
	EHD1 EHD1
	// ECHONET Lite ヘッダ２
	EHD2 EHD2
	// Transaction ID
	TID [2]uint8
	// ECHONET Lite データ
	EDATA Data
}

func NewFrame(bytes []uint8) (*Frame, error) {
	if bytes[0] != uint8(EHD1ECHONETLite) || bytes[1] != uint8(EHD2SpecifiedMessageFormat) {
		return nil, ErrInvalidPacket
	}

	e := &Frame{
		EHD1: EHD1(bytes[0]),
		EHD2: EHD2(bytes[1]),
		TID:  [2]uint8{bytes[2], bytes[3]},
		EDATA: Data{
			SEOJ: [3]uint8{bytes[4], bytes[5], bytes[6]},
			DEOJ: [3]uint8{bytes[7], bytes[8], bytes[9]},
			ESV:  ESV(bytes[10]),
		},
	}

	opc := bytes[11]

	props := make([]property.Property, opc)
	for i := 12; i < len(bytes); i += 2 {
		epc := bytes[i]
		pdc := bytes[i+1]
		edt := bytes[i+2 : i+2+int(pdc)]

		parsed, err := parser.ParseProperty(e.EDATA.SEOJ, epc, edt)
		if err != nil {
			return nil, err
		}

		props[(i-12)/2] = parsed

		i += int(pdc)
	}

	e.EDATA.Properties = props

	return e, nil
}

func (e *Frame) Bytes() []uint8 {
	var data []uint8
	data = append(data, uint8(e.EHD1))
	data = append(data, uint8(e.EHD2))
	data = append(data, e.TID[:]...)
	data = append(data, e.EDATA.SEOJ[:]...)
	data = append(data, e.EDATA.DEOJ[:]...)
	data = append(data, uint8(e.EDATA.ESV))
	data = append(data, uint8(len(e.EDATA.Properties)))

	for _, p := range e.EDATA.Properties {
		set := p.ToSettable()

		data = append(data, uint8(set.EPC))
		data = append(data, uint8(len(set.EDT)))
		data = append(data, set.EDT...)
	}

	return data
}

func (e *Frame) IsPairFrame(f *Frame) bool {
	// transaction ID mismatch
	return slices.Equal(e.TID[:], f.TID[:])
}
