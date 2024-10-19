package MB_RL7023_11

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

var (
	ErrInvalidEventID     = errors.New("invalid event id")
	ErrInvalidEventFormat = errors.New("invalid event format")
)

const (
	// 自端末宛ての UDP（マルチキャスト含む）を受信すると通知されます。
	ERXUDP_ID = "ERXUDP"
	// TCP でデータを受信すると通知されます。
	ERXTCP_ID = "ERXDATA"
	// ICMP Echo reply を受信すると通知されます。
	EPONG_ID = "EPONG"
	// TCP の接続、切断処理が発生すると通知されます。
	ETCP_ID = "ETCP"
	// 自端末で利用可能な IPv6 アドレス一覧を通知します。
	EADDR_ID = "EADDR"
	// 自端末のネイバーキャッシュ内の IPv6 エントリー一覧を通知します。
	ENEIGHBOR_ID = "ENEIGHBOR"
	// アクティブスキャンを実行して発見した PAN を通知します。
	EPANDESC_ID = "EPANDESC"
	// ED スキャンの実行結果を、RSSI 値で一覧表示します。
	EEDSCAN_ID = "EEDSCAN"
	// UDP または TCP の待ち受けポート設定状態を一覧表示します。
	EPORT_ID = "EPORT"
	// TCP ハンドルの現在の状態を一覧表示します。
	EHANDLE_ID = "EHANDLE"
	// 汎用イベント
	EVENT_ID = "EVENT"

	// SKSREG
	ESREG_ID = "ESREG"
	// SKINFO
	EINFO_ID = "EINFO"
	// SKVER
	EVER_ID = "EVER"
	// SKAPPVER
	EAPPVER_ID = "EAPPVER"
)

var EVENT_IDs = []string{
	ERXUDP_ID,
	ERXTCP_ID,
	EPONG_ID,
	ETCP_ID,
	EADDR_ID,
	ENEIGHBOR_ID,
	EPANDESC_ID,
	EEDSCAN_ID,
	EPORT_ID,
	EHANDLE_ID,
	EVENT_ID,

	ESREG_ID,
	EINFO_ID,
	EVER_ID,
	EAPPVER_ID,
}

// 自端末宛ての UDP（マルチキャスト含む）を受信すると通知されます。
type ERXUDP struct {
	Sender    string
	Dest      string
	Rport     uint16
	Lport     uint16
	SenderLLA string
	Secured   bool
	Reserved  uint8
	Data      []uint8
}

func NewERXUDP(line string) (*ERXUDP, error) {
	if !strings.HasPrefix(line, ERXUDP_ID) {
		return nil, ErrInvalidEventID
	}

	fields := strings.Fields(line[len(ERXUDP_ID+" "):])
	if len(fields) != 9 {
		return nil, ErrInvalidEventFormat
	}

	rport, err := strconv.ParseUint(fields[2], 16, 16)
	if err != nil {
		return nil, err
	}
	lport, err := strconv.ParseUint(fields[3], 16, 16)
	if err != nil {
		return nil, err
	}
	secured, err := strconv.ParseBool(fields[5])
	if err != nil {
		return nil, err
	}
	reserved, err := strconv.ParseUint(fields[6], 16, 8)
	if err != nil {
		return nil, err
	}
	data := make([]uint8, len(fields[8])/2)
	for i := 0; i < len(fields[8]); i += 2 {
		b, err := strconv.ParseUint(fields[8][i:i+2], 16, 8)
		if err != nil {
			return nil, err
		}
		data[i/2] = uint8(b)
	}

	return &ERXUDP{
		Sender:    fields[0],
		Dest:      fields[1],
		Rport:     uint16(rport),
		Lport:     uint16(lport),
		SenderLLA: fields[4],
		Secured:   secured,
		Reserved:  uint8(reserved),
		Data:      data,
	}, nil
}

// TCP でデータを受信すると通知されます。
type ERXTCP struct {
	Sender string
	Rport  uint16
	Lport  uint16
	Data   string
}

func NewERXTCP(line string) (*ERXTCP, error) {
	if !strings.HasPrefix(line, ERXTCP_ID) {
		return nil, ErrInvalidEventID
	}

	fields := strings.Fields(line[len(ERXTCP_ID+" "):])
	if len(fields) != 4 {
		return nil, ErrInvalidEventFormat
	}

	rport, err := strconv.ParseUint(fields[1], 16, 16)
	if err != nil {
		return nil, err
	}
	lport, err := strconv.ParseUint(fields[2], 16, 16)
	if err != nil {
		return nil, err
	}

	return &ERXTCP{
		Sender: fields[0],
		Rport:  uint16(rport),
		Lport:  uint16(lport),
		Data:   fields[3],
	}, nil
}

// ICMP Echo reply を受信すると通知されます。
type EPONG struct {
	Sender string
}

func NewEPONG(line string) (*EPONG, error) {
	if !strings.HasPrefix(line, EPONG_ID) {
		return nil, ErrInvalidEventID
	}

	return &EPONG{
		Sender: line[len(EPONG_ID+" "):],
	}, nil
}

// TCP の接続、切断処理が発生すると通知されます。
type ETCP struct {
	Status ETCPStatus
	Handle uint8
	IPAddr string
	Rport  uint16
	Lport  uint16
}

type ETCPStatus uint8

const (
	ETCPStatusConnected             ETCPStatus = 0x01
	ETCPStatusClosed                ETCPStatus = 0x03
	ETCPStatusSourcePortAlreadyUsed ETCPStatus = 0x04
	ETCPStatusSent                  ETCPStatus = 0x05
)

func NewETCP(line string) (*ETCP, error) {
	if !strings.HasPrefix(line, ETCP_ID) {
		return nil, ErrInvalidEventID
	}

	fields := strings.Fields(line[len(ETCP_ID+" "):])
	if len(fields) != 5 {
		return nil, ErrInvalidEventFormat
	}

	status, err := strconv.ParseUint(fields[0], 16, 8)
	if err != nil {
		return nil, err
	}

	var handle uint64
	var ipaddr string
	var rport uint64
	var lport uint64

	if ETCPStatus(status) == ETCPStatusConnected {
		handle, err = strconv.ParseUint(fields[1], 16, 8)
		if err != nil {
			return nil, err
		}
		ipaddr = fields[2]
		rport, err = strconv.ParseUint(fields[3], 16, 16)
		if err != nil {
			return nil, err
		}
		lport, err = strconv.ParseUint(fields[4], 16, 16)
		if err != nil {
			return nil, err
		}
	}

	return &ETCP{
		Status: ETCPStatus(status),
		Handle: uint8(handle),
		IPAddr: ipaddr,
		Rport:  uint16(rport),
		Lport:  uint16(lport),
	}, nil
}

// 自端末で利用可能な IPv6 アドレス一覧を通知します。
type EADDR struct {
	IPAddr []string
}

func NewEADDR(lines []string) (*EADDR, error) {
	if !strings.HasPrefix(lines[0], EADDR_ID) {
		return nil, ErrInvalidEventID
	}

	addresses := make([]string, len(lines)-1)
	for i, line := range lines[1:] {
		addresses[i] = strings.TrimSpace(line)
	}

	return &EADDR{
		IPAddr: addresses,
	}, nil
}

// 自端末のネイバーキャッシュ内の IPv6 エントリー一覧を通知します。
type ENEIGHBOR struct {
	Neighbor []ENEIGHBORNeighbor
}

type ENEIGHBORNeighbor struct {
	IPAddr string
	Addr64 string
	Addr16 uint16
}

func NewENEIGHBOR(lines []string) (*ENEIGHBOR, error) {
	if !strings.HasPrefix(lines[0], ENEIGHBOR_ID) {
		return nil, ErrInvalidEventID
	}

	var neighbors []ENEIGHBORNeighbor
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, ErrInvalidEventFormat
		}

		addr16, err := strconv.ParseUint(fields[2], 16, 16)
		if err != nil {
			return nil, err
		}

		neighbors = append(neighbors, ENEIGHBORNeighbor{
			IPAddr: fields[0],
			Addr64: fields[1],
			Addr16: uint16(addr16),
		})
	}

	return &ENEIGHBOR{
		Neighbor: neighbors,
	}, nil
}

// アクティブスキャンを実行して発見した PAN を通知します。
type EPANDESC struct {
	Channel     uint8
	ChannelPage uint8
	PanID       uint16
	Addr        string
	LQI         uint8
	Side        uint8
	PairID      string
}

func NewEPANDESC(lines []string) (*EPANDESC, error) {
	if !strings.HasPrefix(lines[0], EPANDESC_ID) {
		return nil, ErrInvalidEventID
	}

	if len(lines) != 8 {
		return nil, ErrInvalidEventFormat
	}

	channel, err := strconv.ParseUint(lines[1][len("  Channel:"):], 16, 8)
	if err != nil {
		return nil, err
	}
	channelPage, err := strconv.ParseUint(lines[2][len("  Channel Page:"):], 16, 8)
	if err != nil {
		return nil, err
	}
	panID, err := strconv.ParseUint(lines[3][len("  PAN ID:"):], 16, 16)
	if err != nil {
		return nil, err
	}
	lqi, err := strconv.ParseUint(lines[5][len("  LQI:"):], 16, 8)
	if err != nil {
		return nil, err
	}
	side, err := strconv.ParseUint(lines[6][len("  Side:"):], 16, 8)
	if err != nil {
		return nil, err
	}

	return &EPANDESC{
		Channel:     uint8(channel),
		ChannelPage: uint8(channelPage),
		PanID:       uint16(panID),
		Addr:        lines[4][len("  Addr:"):],
		LQI:         uint8(lqi),
		Side:        uint8(side),
		PairID:      lines[7][len("  PairID:"):],
	}, nil
}

// ED スキャンの実行結果を、RSSI 値で一覧表示します。
type EEDSCAN struct {
	Result []EEDSCANResult
}

type EEDSCANResult struct {
	Channel uint8
	RSSI    uint8
}

func NewEEDSCAN(lines []string) (*EEDSCAN, error) {
	if !strings.HasPrefix(lines[0], EEDSCAN_ID) {
		return nil, ErrInvalidEventID
	}

	fields := strings.Fields(lines[1])
	results := make([]EEDSCANResult, len(fields)/2)
	for i := 0; i < len(fields); i += 2 {
		channel, err := strconv.ParseUint(fields[i], 16, 8)
		if err != nil {
			return nil, err
		}
		rssi, err := strconv.ParseUint(fields[i+1], 16, 8)
		if err != nil {
			return nil, err
		}

		results[i/2] = EEDSCANResult{
			Channel: uint8(channel),
			RSSI:    uint8(rssi),
		}
	}

	return &EEDSCAN{
		Result: results,
	}, nil
}

// UDP または TCP の待ち受けポート設定状態を一覧表示します。
type EPORT struct {
	UDP []uint16
	TCP []uint16
}

func NewEPORT(lines []string) (*EPORT, error) {
	if !strings.HasPrefix(lines[0], EPORT_ID) {
		return nil, ErrInvalidEventID
	}

	var udp []uint16
	var tcp []uint16
	section := "UDP"
	for _, line := range lines[1:] {
		if line == "" {
			section = "TCP"
			continue
		}

		port, err := strconv.ParseUint(line, 10, 16)
		if err != nil {
			return nil, err
		}

		if port == 0 {
			continue
		}

		switch section {
		case "UDP":
			udp = append(udp, uint16(port))
		case "TCP":
			tcp = append(tcp, uint16(port))
		}
	}

	return &EPORT{
		UDP: udp,
		TCP: tcp,
	}, nil
}

// TCP ハンドルの現在の状態を一覧表示します。
type EHANDLE struct {
	Handle []EHANDLEHandle
}

type EHANDLEHandle struct {
	Handle uint8
	IPAddr string
	Rport  uint16
	Lport  uint16
}

func NewEHANDLE(lines []string) (*EHANDLE, error) {
	if !strings.HasPrefix(lines[0], EHANDLE_ID) {
		return nil, ErrInvalidEventID
	}

	var handles []EHANDLEHandle
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) != 4 {
			return nil, ErrInvalidEventFormat
		}

		handle, err := strconv.ParseUint(fields[0], 16, 8)
		if err != nil {
			return nil, err
		}
		rport, err := strconv.ParseUint(fields[2], 16, 16)
		if err != nil {
			return nil, err
		}
		lport, err := strconv.ParseUint(fields[3], 16, 16)
		if err != nil {
			return nil, err
		}

		handles = append(handles, EHANDLEHandle{
			Handle: uint8(handle),
			IPAddr: fields[1],
			Rport:  uint16(rport),
			Lport:  uint16(lport),
		})
	}

	return &EHANDLE{
		Handle: handles,
	}, nil
}

// 汎用イベント
type EVENT struct {
	Num     EVENTNum
	Sender  string
	Param   string
	Payload string
}

type EVENTNum uint8

// https://rabbit-note.com/wp-content/uploads/2016/12/50f67559796399098e50cba8fdbe6d0a.pdf
const (
	EVENTNumNSReceived                      EVENTNum = 0x01
	EVENTNumNAReceived                      EVENTNum = 0x02
	EVENTNumEchoRequestReceived             EVENTNum = 0x05
	EVENTNumEDScanned                       EVENTNum = 0x1f
	EVENTNumBeaconReceived                  EVENTNum = 0x20
	EVENTNumUDPSent                         EVENTNum = 0x21
	EVENTNumActiveScanned                   EVENTNum = 0x22
	EVENTNumPANAConnectionFailed            EVENTNum = 0x24
	EVENTNumPANAConnected                   EVENTNum = 0x25
	EVENTNumSessionCloseRequestReceived     EVENTNum = 0x26
	EVENTNumPANASessionClosed               EVENTNum = 0x27
	EVENTNumPANASessionCloseResponseTimeout EVENTNum = 0x28
	EVENTNumPANASessionTimeout              EVENTNum = 0x29
	EVENTNumTransmissionRateLimitExceeded   EVENTNum = 0x32
	EVENTNumTransmissionRateLimitReleased   EVENTNum = 0x33
)

func (e EVENTNum) String() string {
	return fmt.Sprintf("EVENT %02X", uint8(e))
}

func NewEVENT(line string) (*EVENT, error) {
	if !strings.HasPrefix(line, "EVENT") {
		return nil, ErrInvalidEventID
	}

	fields := strings.Fields(line[len("EVENT "):])
	if len(fields) < 2 {
		return nil, ErrInvalidEventFormat
	}

	num, err := strconv.ParseUint(fields[0], 16, 8)
	if err != nil {
		return nil, err
	}

	payload := ""
	if len(fields) > 3 {
		payload = fields[3]
	}

	return &EVENT{
		Num:     EVENTNum(num),
		Sender:  fields[1],
		Param:   fields[2],
		Payload: payload,
	}, nil
}

// SKSREG
type ESREG struct {
	Val string
}

func NewESREG(line string) (*ESREG, error) {
	if !strings.HasPrefix(line, ESREG_ID) {
		return nil, ErrInvalidEventID
	}

	return &ESREG{
		Val: line[len(ESREG_ID+" "):],
	}, nil
}

// SKINFO
type EINFO struct {
	IPAddr  string
	Addr64  string
	Channel uint8
	PANID   uint16
	Addr16  uint16
}

func NewEINFO(line string) (*EINFO, error) {
	if !strings.HasPrefix(line, EINFO_ID) {
		return nil, ErrInvalidEventID
	}

	fields := strings.Fields(line[len(EINFO_ID+" "):])
	if len(fields) != 5 {
		return nil, ErrInvalidEventFormat
	}

	channel, err := strconv.ParseUint(fields[2], 16, 8)
	if err != nil {
		return nil, err
	}
	panid, err := strconv.ParseUint(fields[3], 16, 16)
	if err != nil {
		return nil, err
	}
	addr16, err := strconv.ParseUint(fields[4], 16, 16)
	if err != nil {
		return nil, err
	}

	return &EINFO{
		IPAddr:  fields[0],
		Addr64:  fields[1],
		Channel: uint8(channel),
		PANID:   uint16(panid),
		Addr16:  uint16(addr16),
	}, nil
}

// SKVER
type EVER struct {
	Version string
}

func NewEVER(line string) (*EVER, error) {
	if !strings.HasPrefix(line, EVER_ID) {
		return nil, ErrInvalidEventID
	}

	return &EVER{
		Version: line[len(EVER_ID+" "):],
	}, nil
}

// SKAPPVER
type EAPPVER struct {
	Version string
}

func NewEAPPVER(line string) (*EAPPVER, error) {
	if !strings.HasPrefix(line, EAPPVER_ID) {
		return nil, ErrInvalidEventID
	}

	return &EAPPVER{
		Version: line[len(EAPPVER_ID+" "):],
	}, nil
}

func ParseEvent(l []string) []any {
	var events []any
	i := 0
	for {
		if i >= len(l) {
			break
		}

		switch {
		case strings.HasPrefix(l[i], ERXUDP_ID):
			e, err := NewERXUDP(l[i])
			if err == nil {
				events = append(events, e)
			} else {
				fmt.Printf("ERXUDP error: %#v\n", err)
			}
			i++
			continue

		case strings.HasPrefix(l[i], ERXTCP_ID):
			e, err := NewERXTCP(l[i])
			if err == nil {
				events = append(events, e)
			}
			i++
			continue

		case strings.HasPrefix(l[i], EPONG_ID):
			e, err := NewEPONG(l[i])
			if err == nil {
				events = append(events, e)
			}
			i++
			continue

		case strings.HasPrefix(l[i], ETCP_ID):
			e, err := NewETCP(l[i])
			if err == nil {
				events = append(events, e)
			}
			i++
			continue

		case strings.HasPrefix(l[i], EADDR_ID):
			j := slices.IndexFunc(l[i:], func(s string) bool {
				return s == "OK"
			})
			if j == -1 {
				i++
				continue
			}
			e, err := NewEADDR(l[i : i+j])
			if err == nil {
				events = append(events, e)
			}
			i += j
			continue

		case strings.HasPrefix(l[i], ENEIGHBOR_ID):
			j := slices.IndexFunc(l[i:], func(s string) bool {
				return s == "OK"
			})
			if j == -1 {
				i++
				continue
			}
			e, err := NewENEIGHBOR(l[i : i+j])
			if err == nil {
				events = append(events, e)
			}
			i += j
			continue

		case strings.HasPrefix(l[i], EPANDESC_ID):
			j := slices.IndexFunc(l[i:], func(s string) bool {
				return s == ""
			})
			if j == -1 {
				i++
				continue
			}
			e, err := NewEPANDESC(l[i : i+j])
			if err == nil {
				events = append(events, e)
			}
			i += j
			continue

		case strings.HasPrefix(l[i], EEDSCAN_ID):
			j := slices.IndexFunc(l[i:], func(s string) bool {
				return s == ""
			})
			if j == -1 {
				i++
				continue
			}
			e, err := NewEEDSCAN(l[i : i+j])
			if err == nil {
				events = append(events, e)
			}
			i += j
			continue

		case strings.HasPrefix(l[i], EPORT_ID):
			j := slices.IndexFunc(l[i:], func(s string) bool {
				return s == "OK"
			})
			if j == -1 {
				i++
				continue
			}
			e, err := NewEPORT(l[i : i+j])
			if err == nil {
				events = append(events, e)
			}
			i += j
			continue

		case strings.HasPrefix(l[i], EHANDLE_ID):
			j := slices.IndexFunc(l[i:], func(s string) bool {
				return s == "OK"
			})
			if j == -1 {
				i++
				continue
			}
			e, err := NewEHANDLE(l[i : i+j])
			if err == nil {
				events = append(events, e)
			}
			i += j
			continue

		case strings.HasPrefix(l[i], EVENT_ID):
			e, err := NewEVENT(l[i])
			if err == nil {
				events = append(events, e)
			}
			i++
			continue

		case strings.HasPrefix(l[i], ESREG_ID):
			e, err := NewESREG(l[i])
			if err == nil {
				events = append(events, e)
			}
			i++
			continue

		case strings.HasPrefix(l[i], EINFO_ID):
			e, err := NewEINFO(l[i])
			if err == nil {
				events = append(events, e)
			}
			i++
			continue

		case strings.HasPrefix(l[i], EVER_ID):
			e, err := NewEVER(l[i])
			if err == nil {
				events = append(events, e)
			}
			i++
			continue

		case strings.HasPrefix(l[i], EAPPVER_ID):
			e, err := NewEAPPVER(l[i])
			if err == nil {
				events = append(events, e)
			}
			i++
			continue

		default:
			i++
		}
	}

	return events
}
