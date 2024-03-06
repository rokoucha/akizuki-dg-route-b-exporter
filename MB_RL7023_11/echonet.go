package MB_RL7023_11

import (
	"errors"
	"slices"
)

var (
	ErrInvalidECHONETService = errors.New("invalid ECHONET service")
	ErrInvalidMessage        = errors.New("invalid message")
	ErrInvalidPacket         = errors.New("invalid packet")
)

// ECHONET Lite ヘッダ１
type ECHONETLiteEHD1 uint8

const (
	// ECHONET Lite
	ECHONETLiteEHD1ECHONETLite ECHONETLiteEHD1 = 0x10
)

// ECHONET Lite ヘッダ２
type ECHONETLiteEHD2 uint8

const (
	// 形式1（規定電文形式）
	ECHONETLiteEHD2SpecifiedMessageFormat ECHONETLiteEHD2 = 0x81
	// 形式2（任意電文形式）
	ECHONETLiteEHD2ArbitraryMessageFormat ECHONETLiteEHD2 = 0x82
)

// ECHONET Lite サービス
type ECHONETLiteESV uint8

// 要求用 ESV コード
const (
	// プロパティ値書き込み要求（応答不要）
	ECHONETLiteESVSetI ECHONETLiteESV = 0x60
	// プロパティ値書き込み要求（応答要）
	ECHONETLiteESVSetC ECHONETLiteESV = 0x61
	// プロパティ値読み出し要求
	ECHONETLiteESVGet ECHONETLiteESV = 0x62
	// プロパティ値通知要求
	ECHONETLiteESVINF_REQ ECHONETLiteESV = 0x63
)

// 応答・通知用 ESV コード
const (
	// プロパティ値書き込み応答
	ECHONETLiteESVSet_Res ECHONETLiteESV = 0x71
	// プロパティ値読み出し応答
	ECHONETLiteESVGet_Res ECHONETLiteESV = 0x72
	// プロパティ値通知
	ECHONETLiteESVINF ECHONETLiteESV = 0x73
	// プロパティ値通知（応答要）
	ECHONETLiteESVINFC ECHONETLiteESV = 0x74
	// プロパティ値通知応答
	ECHONETLiteESVINFC_Res ECHONETLiteESV = 0x7a
	// プロパティ値書き込み・読み出し応答
	ECHONETLiteESVSetGet_Res ECHONETLiteESV = 0x7e
)

// 不可応答用 ESV コード
const (
	// プロパティ値書き込み要求不可応答（応答不要）
	ECHONETLiteESVSetI_SNA ECHONETLiteESV = 0x50
	// プロパティ値書き込み要求不可応答（応答要）
	ECHONETLiteESVSetC_SNA ECHONETLiteESV = 0x51
	// プロパティ値読み出し不可応答
	ECHONETLiteESVGet_SNA ECHONETLiteESV = 0x52
	// プロパティ値通知不可応答
	ECHONETLiteESVINF_SNA ECHONETLiteESV = 0x53
	// プロパティ値書き込み・読み出し不可応答
	ECHONETLiteESVSetGet_SNA ECHONETLiteESV = 0x5e
)

// ECHONET プロパティ
type ECHONETLitePropertyCode uint8

const (
	// ３．３．２５ 低圧スマート電力量メータクラス規定 瞬時電力計測値
	ECHONETLitePropertyCodeInstantaneousPowerMeasurementValue ECHONETLitePropertyCode = 0xE7
)

// ECHONET プロパティ
type ECHONETLiteProperty struct {
	EPC ECHONETLitePropertyCode
	EDT []uint8
}

// ECHONET Liteデータ
type ECHONETLiteData struct {
	SEOJ  [3]uint8
	DEOJ  [3]uint8
	ESV   ECHONETLiteESV
	Props []ECHONETLiteProperty
}

// ECHONET Lite フレーム
type ECHONETLiteFrame struct {
	EHD1  ECHONETLiteEHD1
	EHD2  ECHONETLiteEHD2
	TID   [2]uint8
	EDATA ECHONETLiteData
}

func NewECHONETLiteFrame(bytes []uint8) (*ECHONETLiteFrame, error) {
	if bytes[0] != uint8(ECHONETLiteEHD1ECHONETLite) || bytes[1] != uint8(ECHONETLiteEHD2SpecifiedMessageFormat) {
		return nil, ErrInvalidPacket
	}

	var props []ECHONETLiteProperty
	for i := 12; i < len(bytes); i += 2 {
		epc := bytes[i]
		pdc := bytes[i+1]
		edt := bytes[i+2 : i+2+int(pdc)]

		props = append(props, ECHONETLiteProperty{
			EPC: ECHONETLitePropertyCode(epc),
			EDT: edt,
		})

		i += int(pdc)
	}

	if len(props) != int(bytes[11]) {
		return nil, ErrInvalidPacket
	}

	e := &ECHONETLiteFrame{
		EHD1: ECHONETLiteEHD1(bytes[0]),
		EHD2: ECHONETLiteEHD2(bytes[1]),
		TID:  [2]uint8{bytes[2], bytes[3]},
		EDATA: ECHONETLiteData{
			SEOJ:  [3]uint8{bytes[4], bytes[5], bytes[6]},
			DEOJ:  [3]uint8{bytes[7], bytes[8], bytes[9]},
			ESV:   ECHONETLiteESV(bytes[10]),
			Props: props,
		},
	}

	return e, nil
}

func (e *ECHONETLiteFrame) Bytes() []uint8 {
	var data []uint8
	data = append(data, uint8(e.EHD1))
	data = append(data, uint8(e.EHD2))
	data = append(data, e.TID[:]...)
	data = append(data, e.EDATA.SEOJ[:]...)
	data = append(data, e.EDATA.DEOJ[:]...)
	data = append(data, uint8(e.EDATA.ESV))
	data = append(data, uint8(len(e.EDATA.Props)))

	for _, p := range e.EDATA.Props {
		data = append(data, uint8(p.EPC))
		data = append(data, uint8(len(p.EDT)))
		data = append(data, p.EDT...)
	}

	return data
}

func (e *ECHONETLiteFrame) IsPairFrame(f *ECHONETLiteFrame) bool {
	// transaction ID mismatch
	return slices.Equal(e.TID[:], f.TID[:])
}

func (e *ECHONETLiteFrame) InstantaneousPowerMeasurementValue() (int32, error) {
	if e.EDATA.ESV != ECHONETLiteESVGet_Res {
		return 0, ErrInvalidECHONETService
	}

	if len(e.EDATA.Props) != 1 {
		return 0, ErrInvalidMessage
	}

	if e.EDATA.Props[0].EPC != ECHONETLitePropertyCodeInstantaneousPowerMeasurementValue {
		return 0, ErrInvalidMessage
	}

	if len(e.EDATA.Props[0].EDT) != 4 {
		return 0, ErrInvalidMessage
	}

	var value int32
	for _, b := range e.EDATA.Props[0].EDT {
		value = (value << 8) | int32(b)
	}

	return value, nil
}
