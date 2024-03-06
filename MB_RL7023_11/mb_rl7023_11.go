// https://rabbit-note.com/wp-content/uploads/2016/12/50f67559796399098e50cba8fdbe6d0a.pdf
package MB_RL7023_11

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/rokoucha/akizuki-dg-route-b-exporter/serial"
)

const (
	execTimeout = 5
)

var (
	ErrAddressUnavaiable = errors.New("address unavaiable")
	ErrEchobackMismatch  = errors.New("echoback mismatch")
	ErrExecFailed        = errors.New("exec failed")
	ErrFailedToConnect   = errors.New("failed to connect")
	ErrPortUnavaiable    = errors.New("port unavaiable")
	ErrUnexpectedOutput  = errors.New("unexpected output")
)

type MB_RL7023_11 struct {
	addrs  []string
	logger *slog.Logger
	ports  []uint16
	serial *serial.Serial
}

type Config struct {
	Logger *slog.Logger
	Serial *serial.Serial
}

func New(c Config) *MB_RL7023_11 {
	return &MB_RL7023_11{
		addrs:  []string{},
		logger: c.Logger,
		ports:  []uint16{},
		serial: c.Serial,
	}
}

func (m *MB_RL7023_11) Initialize(ctx context.Context) error {
	var err error
	_, err = m.serial.Write([]uint8("\r\n"))
	if err != nil {
		return err
	}

	err = m.SKRESET(ctx)
	if err != nil {
		return err
	}

	res, err := m.SKTABLE(ctx, SKTABLEModeAvailableIPAddresses)
	if err != nil {
		return err
	}
	eaddr, ok := res.(*EADDR)
	if !ok {
		return ErrAddressUnavaiable
	}
	m.addrs = eaddr.IPAddr
	if len(m.addrs) == 0 {
		return ErrAddressUnavaiable
	}

	res, err = m.SKTABLE(ctx, SKTABLEModeListeningPort)
	if err != nil {
		return err
	}
	eport, ok := res.(*EPORT)
	if !ok {
		return ErrPortUnavaiable
	}
	m.ports = eport.UDP
	if len(m.ports) == 0 {
		return ErrPortUnavaiable
	}

	return nil
}

func startWithStopper(targets []string) func(l []string) bool {
	return func(l []string) bool {
		return slices.ContainsFunc(l, func(s string) bool {
			return slices.ContainsFunc(targets, func(t string) bool {
				return strings.HasPrefix(s, t)
			})
		})
	}
}

type execOptions struct {
	Payload []uint8
	Stopper func(l []string) bool
	Timeout time.Duration
}

func (m *MB_RL7023_11) exec(ctx context.Context, command string, options ...execOptions) ([]string, []any, error) {
	o := execOptions{}
	if len(options) > 0 {
		o = options[0]
	}
	if o.Timeout == 0 {
		o.Timeout = execTimeout * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()

	cmd := []uint8(command)
	if o.Payload == nil {
		cmd = append(cmd, "\r\n"...)
	} else {
		cmd = append(cmd, o.Payload...)
	}

	stopper := func(l []string) bool {
		if slices.ContainsFunc(l, func(s string) bool { return strings.HasPrefix(s, "FAIL") }) {
			return true
		}
		if o.Stopper != nil {
			return o.Stopper(l)
		}
		return len(l) == 1 && l[0] == ""
	}

	res, err := m.serial.Exec(ctx, cmd, stopper)
	if err != nil {
		return nil, nil, err
	}

	echobackLine := slices.Index(res, command)
	if echobackLine == -1 {
		return nil, nil, ErrEchobackMismatch
	}
	linebase := echobackLine + 1

	failLine := slices.IndexFunc(res[linebase:], func(line string) bool { return strings.HasPrefix(line, "FAIL") })
	if failLine != -1 {
		return []string{res[failLine+linebase]}, nil, ErrExecFailed
	}

	okLine := slices.IndexFunc(res[linebase:], func(line string) bool { return strings.HasPrefix(line, "OK") })
	if okLine == -1 {
		okLine = len(res)
	} else {
		okLine += linebase
	}

	output := res[linebase:okLine]

	events := ParseEvent(res[linebase:])

	//fmt.Printf("res: %#v, echobackLine: %d, failLine: %d, okLine: %d, output: %#v, events: %#v\n", res, echobackLine, failLine, okLine, output, events)

	return output, events, nil
}

// 仮想レジスタの内容を表示・設定します。
func (m *MB_RL7023_11) SKSREG(ctx context.Context, sreg Register, val string) (*ESREG, error) {
	command := "SKSREG " + sreg.String()
	if val != "" {
		command += " " + val
	}

	res, events, err := m.exec(ctx, command)
	if err != nil {
		return nil, parseError(res, err)
	}

	if val != "" && len(events) == 0 {
		return nil, nil
	}

	for _, v := range events {
		if e, ok := v.(*ESREG); ok {
			return e, nil
		}
	}

	return nil, ErrUnexpectedOutput
}

// 現在の主要な通信設定値を表示します。
func (m *MB_RL7023_11) SKINFO(ctx context.Context) (*EINFO, error) {
	res, events, err := m.exec(ctx, "SKINFO")
	if err != nil {
		return nil, parseError(res, err)
	}
	for _, v := range events {
		if e, ok := v.(*EINFO); ok {
			return e, nil
		}
	}

	return nil, ErrUnexpectedOutput
}

// 端末を PAA (PANA 認証サーバ)として動作開始します。
func (m *MB_RL7023_11) SKSTART(ctx context.Context) error {
	res, _, err := m.exec(ctx, "SKSTART")
	return parseError(res, err)
}

// 指定した<IPADDR>に対して PaC（PANA 認証クライアント）として PANA 接続シーケンスを開始します。
func (m *MB_RL7023_11) SKJOIN(ctx context.Context, ipaddr string) error {
	stopper := startWithStopper([]string{
		EVENTNumPANAConnectionFailed.String(),
		EVENTNumPANAConnected.String(),
	})
	res, events, err := m.exec(ctx, "SKJOIN "+ipaddr, execOptions{Stopper: stopper, Timeout: execTimeout * 2 * time.Second})
	if err != nil {
		return parseError(res, err)
	}

	for _, v := range events {
		if e, ok := v.(*EVENT); ok {
			if e.Num == EVENTNumPANAConnectionFailed {
				return ErrFailedToConnect
			}
			if e.Num == EVENTNumPANAConnected {
				return nil
			}
		}
	}

	return ErrUnexpectedOutput
}

// 現在接続中の相手に対して再認証シーケンスを開始します。
func (m *MB_RL7023_11) SKREJOIN(ctx context.Context) error {
	stopper := startWithStopper([]string{
		EVENTNumPANAConnectionFailed.String(),
		EVENTNumPANAConnected.String(),
	})
	res, events, err := m.exec(ctx, "SKREJOIN", execOptions{Stopper: stopper, Timeout: execTimeout * 2 * time.Second})
	if err != nil {
		return parseError(res, err)
	}

	for _, v := range events {
		if e, ok := v.(*EVENT); ok {
			if e.Num == EVENTNumPANAConnectionFailed {
				return ErrFailedToConnect
			}
			if e.Num == EVENTNumPANAConnected {
				return nil
			}
		}
	}

	return ErrUnexpectedOutput
}

// 現在確立している PANA セッションの終了を要請します。
func (m *MB_RL7023_11) SKTERM(ctx context.Context) error {
	stopper := startWithStopper([]string{
		EVENTNumPANASessionClosed.String(),
		EVENTNumPANASessionCloseResponseTimeout.String(),
	})
	res, events, err := m.exec(ctx, "SKTERM", execOptions{Stopper: stopper, Timeout: execTimeout * 2 * time.Second})
	if err != nil {
		return parseError(res, err)
	}

	for _, v := range events {
		if e, ok := v.(*EVENT); ok {
			if e.Num == EVENTNumPANASessionClosed {
				return nil
			}
			if e.Num == EVENTNumPANASessionCloseResponseTimeout {
				return ErrFailedToConnect
			}
		}
	}

	return ErrUnexpectedOutput
}

// 暗号化オプション
type SKSENDTOSec uint8

const (
	// 必ず平文で送信
	SKSENDTOSecDisable SKSENDTOSec = 0x00
	// SKSECENABLE コマンドで送信先がセキュリティ有効で登録されている場合、暗号化して送ります。登録されてない場合、または、暗号化無しで登録されている場合、データは送信されません。
	SKSENDTOSecStrict SKSENDTOSec = 0x01
	// SKSECENABLE コマンドで送信先がセキュリティ有効で登録されている場合、暗号化して送ります。登録されてない場合、または、暗号化無しで登録されている場合、データは平文で送信されます。
	SKSENDTOSecModerate SKSENDTOSec = 0x02
)

type SKSENDTOReserved uint8

const (
	SKSENDTOReservedValue SKSENDTOReserved = 0x00
)

func (m *MB_RL7023_11) erxudpMatcher(received *ERXUDP, handle uint8, ipaddr string, port uint16, frame *ECHONETLiteFrame) bool {
	// sender mismatch
	if received.Sender != ipaddr {
		m.logger.Debug("sender mismatch", "expected", ipaddr, "actual", received.Sender)
		return false
	}

	// receiver mismatch
	if !slices.Contains(m.addrs, received.Dest) {
		m.logger.Debug("receiver mismatch", "expected", m.addrs, "actual", received.Dest)
		return false
	}

	// source port mismatch
	if received.Rport != port {
		m.logger.Debug("source port mismatch", "expected", port, "actual", received.Rport)
		return false
	}

	// destination port mismatch
	if received.Lport != m.ports[handle-1] {
		m.logger.Debug("destination port mismatch", "expected", m.ports[handle-1], "actual", received.Lport)
		return false
	}

	// ECHONET Lite frame mismatch
	if !frame.IsPairFrame(received.Data) {
		m.logger.Debug("ECHONET Lite frame mismatch", "expected", frame, "actual", received.Data)
		return false
	}

	return true
}

// 指定した宛先に UDP でデータを送信します。
func (m *MB_RL7023_11) SKSENDTO(ctx context.Context, handle uint8, ipaddr string, port uint16, sec SKSENDTOSec, reserved SKSENDTOReserved, data *ECHONETLiteFrame) (*ERXUDP, error) {
	if len(m.addrs) == 0 {
		return nil, ErrAddressUnavaiable
	}

	if len(m.ports) == 0 || len(m.ports) < int(handle) {
		return nil, ErrPortUnavaiable
	}

	payload := data.Bytes()

	command := fmt.Sprintf(
		"SKSENDTO %X %s %04X %X %X %04X ",
		handle,
		ipaddr,
		port,
		sec,
		reserved,
		len(payload),
	)
	stopper := func(l []string) bool {
		return slices.ContainsFunc(l, func(s string) bool {
			if !strings.HasPrefix(s, ERXUDP_ID) {
				return false
			}
			e, err := NewERXUDP(s)
			if err != nil {
				return false
			}

			return m.erxudpMatcher(e, handle, ipaddr, port, data)
		})
	}
	res, events, err := m.exec(ctx, command, execOptions{Payload: payload, Stopper: stopper, Timeout: execTimeout * 2 * time.Second})
	if err != nil {
		return nil, parseError(res, err)
	}

	for _, v := range events {
		if e, ok := v.(*ERXUDP); ok {
			if m.erxudpMatcher(e, handle, ipaddr, port, data) {
				return e, nil
			}
		}
	}

	return nil, ErrUnexpectedOutput
}

// 指定した宛先に TCP の接続要求を発行します。
func (m *MB_RL7023_11) SKCONNECT(ctx context.Context, ipaddr string, rport uint16, lport uint16) (*ETCP, error) {
	panic("not supported")
}

// 指定したハンドル番号に対応する TCP コネクションを介して接続相手にデータを送信します。
func (m *MB_RL7023_11) SKSEND(ctx context.Context, handle uint8, data []uint8) (*ETCP, error) {
	stopper := startWithStopper([]string{ETCP_ID})
	command := fmt.Sprintf("SKSEND %X %04X ", handle, len(data))
	res, events, err := m.exec(ctx, command, execOptions{Payload: data, Stopper: stopper, Timeout: execTimeout * 2 * time.Second})
	if err != nil {
		return nil, parseError(res, err)
	}

	for _, v := range events {
		if e, ok := v.(*ETCP); ok {
			return e, nil
		}
	}

	return nil, ErrUnexpectedOutput
}

// 指定したハンドルに対応する TCP コネクションの切断要求を発行します。
func (m *MB_RL7023_11) SKCLOSE(ctx context.Context, handle uint8) error {
	panic("not supported")
}

type SKPINGReserved uint8

const SKPINGReservedValue SKPINGReserved = 0x00

// 指定した IPv6 宛てに ICMP Echo request を送信します。
func (m *MB_RL7023_11) SKPING(ctx context.Context, reserved SKPINGReserved, ipaddr string) error {
	stopper := startWithStopper([]string{EPONG_ID})
	res, _, err := m.exec(ctx, fmt.Sprintf("SKPING %X %s", reserved, ipaddr), execOptions{Stopper: stopper, Timeout: execTimeout * 2 * time.Second})
	return parseError(res, err)
}

type SKSCANMode uint8

const (
	// ED スキャン
	SKSCANModeEDScan SKSCANMode = 0x00
	// アクティブスキャン（IE あり）
	SKSCANModeActiveWithIE SKSCANMode = 0x02
	// アクティブスキャン（IE なし）
	SKSCANModeActiveWithoutIE SKSCANMode = 0x03
)

type SKSCANReserved uint8

const SKSCANReservedValue SKSCANReserved = 0x00

// 指定したチャンネルに対してアクティブスキャンまたは ED スキャンを実行します。
func (m *MB_RL7023_11) SKSCAN(ctx context.Context, mode SKSCANMode, channelMask uint32, duration uint8, reserved SKSCANReserved) ([]any, error) {
	command := fmt.Sprintf("SKSCAN %X %08X %X %X", mode, channelMask, duration, reserved)
	stopper := startWithStopper([]string{
		EVENTNumActiveScanned.String(),
		EEDSCAN_ID,
	})
	res, events, err := m.exec(ctx, command, execOptions{Stopper: stopper, Timeout: 1 * time.Minute})
	if err != nil {
		return nil, parseError(res, err)
	}

	var result []any
	for _, v := range events {
		switch mode {
		case SKSCANModeEDScan:
			if e, ok := v.(*EEDSCAN); ok {
				result = append(result, e)
			}

		case SKSCANModeActiveWithIE, SKSCANModeActiveWithoutIE:
			if e, ok := v.(*EPANDESC); ok {
				result = append(result, e)
			}
		}
	}

	return result, nil
}

// セキュリティを適用するため、指定した IP アドレスを端末に登録します。
func (m *MB_RL7023_11) SKREGDEV(ctx context.Context, ipaddr string) error {
	res, _, err := m.exec(ctx, "SKREGDEV "+ipaddr)
	return parseError(res, err)
}

// 指定した IP アドレスのエントリーをネイバーテーブル、ネイバーキャッシュから強制的に削除します。
func (m *MB_RL7023_11) SKRMDEV(ctx context.Context, target string) error {
	res, _, err := m.exec(ctx, "SKRMDEV "+target)
	return parseError(res, err)
}

// 指定されたキーインデックスに対する暗号キー(128bit)を、MAC 層セキュリティコンポーネントに登録します。
func (m *MB_RL7023_11) SKSETKEY(ctx context.Context, index uint8, key string) error {
	res, _, err := m.exec(ctx, fmt.Sprintf("SKSETKEY %X %s", index, key))
	return parseError(res, err)
}

// 指定されたキーインデックスに対する暗号キー(128bit)を、MAC 層セキュリティコンポーネントから削除します。
func (m *MB_RL7023_11) SKRMKEY(ctx context.Context, index uint8) error {
	res, _, err := m.exec(ctx, fmt.Sprintf("SKRMKEY %X", index))
	return parseError(res, err)
}

type SKSECENABLEMode uint16

const (
	// セキュリティ無効
	SKSECENABLEModeDisable SKSECENABLEMode = 0x00
	// セキュリティ適用
	SKSECENABLEModeEnable SKSECENABLEMode = 0x01
)

// 指定した IP アドレスに対する MAC 層セキュリティの有効・無効を指定します。
func (m *MB_RL7023_11) SKSECENABLE(ctx context.Context, mode SKSECENABLEMode, ipaddr string, macaddr string) error {
	res, _, err := m.exec(ctx, fmt.Sprintf("SKSECENABLE %X %s %s", mode, ipaddr, macaddr))
	return parseError(res, err)
}

// PANA 認証に用いる PSK を登録します。
func (m *MB_RL7023_11) SKSETPSK(ctx context.Context, psk string) error {
	res, _, err := m.exec(ctx, fmt.Sprintf("SKSETPSK %X %s", len(psk), psk))
	return parseError(res, err)
}

// PWD で指定したパスワードから PSK を生成して登録します。
func (m *MB_RL7023_11) SKSETPWD(ctx context.Context, pwd string) error {
	res, _, err := m.exec(ctx, fmt.Sprintf("SKSETPWD %X %s", len(pwd), pwd))
	return parseError(res, err)
}

// 指定された<ID>から各 Route-B ID を生成して設定します。
func (m *MB_RL7023_11) SKSETRBID(ctx context.Context, id string) error {
	res, _, err := m.exec(ctx, "SKSETRBID "+id)
	return parseError(res, err)
}

// 指定した IP アドレスと 64bit アドレス情報を、IP 層のネイバーキャッシュに Reachable 状態で登録します。これによってアドレス要請を省略して直接 IP パケットを出力することができます。
func (m *MB_RL7023_11) SKADDNBR(ctx context.Context, ipaddr string, macaddr string) error {
	res, _, err := m.exec(ctx, "SKADDNBR "+ipaddr+" "+macaddr)
	return parseError(res, err)
}

// UDP の待ち受けポートを指定します。
func (m *MB_RL7023_11) SKUDPPORT(ctx context.Context, handle uint8, port uint16) error {
	res, _, err := m.exec(ctx, fmt.Sprintf("SKUDPPORT %X %04X", handle, port))
	return parseError(res, err)
}

// TCP の待ち受けポートを指定します。
func (m *MB_RL7023_11) SKTCPPORT(ctx context.Context, index uint8, port uint16) error {
	res, _, err := m.exec(ctx, fmt.Sprintf("SKTCPPORT %X %04X", index, port))
	return parseError(res, err)
}

// 現在の仮想レジスタの内容を不揮発性メモリに保存します。
func (m *MB_RL7023_11) SKSAVE(ctx context.Context) error {
	res, _, err := m.exec(ctx, "SKSAVE")
	return parseError(res, err)
}

// 不揮発性メモリに保存されている仮想レジスタの内容をロードします。
func (m *MB_RL7023_11) SKLOAD(ctx context.Context) error {
	res, _, err := m.exec(ctx, "SKLOAD")
	return parseError(res, err)
}

// レジスタ保存用の不揮発性メモリエリアを初期化して、未保存状態に戻します。
func (m *MB_RL7023_11) SKERASE(ctx context.Context) error {
	res, _, err := m.exec(ctx, "SKERASE")
	return parseError(res, err)
}

// SKSTACK IP のファームウェアバージョンを表示します。
func (m *MB_RL7023_11) SKVER(ctx context.Context) (*EVER, error) {
	res, events, err := m.exec(ctx, "SKVER")
	if err != nil {
		return nil, parseError(res, err)
	}

	for _, v := range events {
		if e, ok := v.(*EVER); ok {
			return e, nil
		}
	}

	return nil, ErrUnexpectedOutput
}

// (not supported) アプリケーションのファームウェアバージョンを表示します。
func (m *MB_RL7023_11) SKAPPVER(ctx context.Context) (string, error) {
	panic("not supported")
}

// プロトコル・スタックの内部状態を初期化します。
func (m *MB_RL7023_11) SKRESET(ctx context.Context) error {
	res, _, err := m.exec(ctx, "SKRESET")
	return parseError(res, err)
}

type SKTABLEMode uint8

const (
	// 端末で利用可能な IP アドレス一覧
	SKTABLEModeAvailableIPAddresses SKTABLEMode = 0x01
	// ネイバーキャッシュ
	SKTABLEModeNeighborCache SKTABLEMode = 0x02
	// 待ち受けポート設定状態一覧
	SKTABLEModeListeningPort SKTABLEMode = 0x0e
	// TCP ハンドル状態一覧
	SKTABLEModeTCPHandle SKTABLEMode = 0x0f
)

// SKSTACK IP 内の各種テーブル内容を画面表示します。
func (m *MB_RL7023_11) SKTABLE(ctx context.Context, mode SKTABLEMode) (any, error) {
	res, events, err := m.exec(ctx, fmt.Sprintf("SKTABLE %X", mode))
	if err != nil {
		return nil, parseError(res, err)
	}

	for _, v := range events {
		switch mode {
		case SKTABLEModeAvailableIPAddresses:
			if e, ok := v.(*EADDR); ok {
				return e, nil
			}

		case SKTABLEModeNeighborCache:
			if e, ok := v.(*ENEIGHBOR); ok {
				return e, nil
			}

		case SKTABLEModeListeningPort:
			if e, ok := v.(*EPORT); ok {
				return e, nil
			}

		case SKTABLEModeTCPHandle:
			if e, ok := v.(*EHANDLE); ok {
				return e, nil
			}
		}
	}

	return nil, ErrUnexpectedOutput
}

// (not supported) スリープモードに移行します。
func (m *MB_RL7023_11) SKDSLEEP(ctx context.Context) error {
	panic("not supported")
}

// (not supported) 受信時のローカル周波数を Lower Local か Upper Local に設定します。
func (m *MB_RL7023_11) SKRFLO(ctx context.Context, mode uint8) error {
	panic("not supported")
}

// MAC アドレス(64bit)から IPv6 リンクローカルアドレスへ変換した結果を表示します。
func (m *MB_RL7023_11) SKLL64(ctx context.Context, addr64 string) (string, error) {
	res, _, err := m.exec(ctx, "SKLL64 "+addr64)
	return res[0], parseError(res, err)
}

// (not supported) ERXUDP、ERXTCP のデータ部の表示形式を設定します。
func (m *MB_RL7023_11) WOPT(mctx context.Context, ode uint8) error {
	panic("not supported")
}

// (not supported) WOPT コマンドの設定状態を表示します。
func (m *MB_RL7023_11) ROPT(ctx context.Context) (uint8, error) {
	panic("not supported")
}

// (not supported) UART 設定（ボーレート、キャラクター間インターバル、フロー制御）を設定します。
func (m *MB_RL7023_11) WUART(ctx context.Context, mode uint8) error {
	panic("not supported")
}

// (not supported) WUART コマンドの設定状態を表示します。
func (m *MB_RL7023_11) RUART(ctx context.Context) (uint8, error) {
	panic("not supported")
}
