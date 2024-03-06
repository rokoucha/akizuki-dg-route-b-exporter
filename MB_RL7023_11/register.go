package MB_RL7023_11

import (
	"fmt"
)

type Register uint8

// 内容 属性 初期値 値域 保存
const (
	// 自端末が使用する周波数の論理チャンネル番号 R/W 0x21 0x21 – 0x3C ○
	RegisterChannel Register = 0x02
	// 自端末の PAN ID R/W 0xFFFF 0x0000 – 0xFFFF ○
	RegisterPANID Register = 0x03
	// MAC 層セキュリティのフレームカウンタ R 0x00000000 0x00000000 – 0xFFFFFFFF ×
	RegisterFrameCounter Register = 0x07
	// Pairing ID R/W CCDDEEFF ASCII 8 文字 ×
	RegisterPairingID Register = 0x0a
	// ビーコン応答の制御フラグ R/W 0 0 or 1 ×
	RegisterRespondBeaconRequest Register = 0x15
	// PANA セッションライフタイム R/W 0x00000384(900 秒) 0x0000003C – 0xFFFFFFFF ×
	RegisterPANASessionLifetime Register = 0x16
	// 自動再認証フラグ R/W 1 0 or 1 ×
	RegisterAutoReAuthentication Register = 0x17
	// MAC 層ブロードキャストに対するセキュリティ制御 R/W 1 0 or 1 ×
	RegisterEncryptBroadcastIPPacket Register = 0xa0
	// ICMP メッセージ処理制御 R/W 1 0 or 1 ×
	RegisterPlainICMPMessageAccept Register = 0xa1
	// 送信時間制限中フラグ R 0 0 or 1 ×
	RegisterTransmissionRateLimitExceeded Register = 0xfb
	// 無線送信の積算時間（単位ミリ秒） R(0 のみ書き込み可能) 0 0x0 – 0xFFFFFFFFFFFFFFFF ×
	RegisterTransmissionSunTime Register = 0xfd
	// エコーバックフラグ R/W 1 0 or 1 ×
	RegisterEchoback Register = 0xfe
	// オートロード R/W 0 0 or 1 ○
	RegisterAutoload Register = 0xff
)

func (r Register) String() string {
	return fmt.Sprintf("S%02X", uint8(r))
}
