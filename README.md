# akizuki-dg-route-b-exporter

秋月電子で販売されていたWi-SUN搭載LinuxボードでスマートメーターのBルートから計測値を読み出し、MQTT経由でHome Assistantに連携するツール

## 使い方

PANをスキャンして、スマートメーターのチャンネル・PAN ID・アドレスを調べる

```sh
export ROUTE_B_ID=... ROUTE_B_PASSWORD=...
go run . --scan /dev/ttyUSB0
```

調べた値を設定して起動、`MQTT_BROKER` を設定しない場合はログに出力するだけになる

```sh
export ROUTE_B_CHANNEL=21 ROUTE_B_PANID=1234 ROUTE_B_ADDR=FE80:0000:0000:0000:...
export MQTT_BROKER=tcp://homeassistant.local:1883 MQTT_USERNAME=... MQTT_PASSWORD=...
go run . /dev/ttyUSB0
```

## Home Assistant

[MQTT 統合](https://www.home-assistant.io/integrations/mqtt/)を設定しておくと、電流プロパティの初回応答後にMQTT Discoveryで「スマートメーター」デバイスが自動で追加される。計測値なし（`0x7FFE`）の相はDiscoveryにも含めない。

| エンティティ             | 単位 | 備考                                                 |
| ------------------------ | ---- | ---------------------------------------------------- |
| 瞬時電力                 | W    |                                                      |
| 瞬時電流 R相 / T相       | A    |                                                      |
| 積算電力量 正方向 / 逆方向 | kWh  | `total_increasing` なのでエネルギーダッシュボードで使えます |
| 直近1時間の送信時間      | s    | 診断用                                               |
| 送信時間制限中           |      | 診断用                                               |

## links

- <https://rabbit-note.com/wp-content/uploads/2016/12/50f67559796399098e50cba8fdbe6d0a.pdf>
- <https://echonet.jp/wp/wp-content/uploads/pdf/General/Standard/ECHONET_lite_V1_14_jp/ECHONET-Lite_Ver.1.14(02).pdf>
- <https://echonet.jp/wp/wp-content/uploads/pdf/General/Standard/Release/Release_R/Appendix_Release_R_r1.pdf>
- <https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery>

## ライセンス

Copyright (c) 2026 Rokoucha

Licensed under the Apache License, Version 2.0.

Mahiron's logo is licensed under a Creative Commons Attribution-ShareAlike 4.0 International License.