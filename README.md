# adbee

Easily pair and connect to ADB over Wi-Fi via
mDNS discovery ([Zeroconf](https://ru.wikipedia.org/wiki/Zeroconf))
without Android Studio or desktop environment installed.

![](demo.gif)

## Requirements

- adb installed and on PATH
- Android device with Wireless debugging enabled
- Same network

## Install

- Download a binary from Releases (tag v1.x), or build:
  - Linux/macOS: `VERSION=v1.0.0 ./build.sh`
  - Windows: `set VERSION=v1.0.0 && build.bat`

## Usage

```bash
# defaults: random name/password, shows QR, pairs, then connects
./adbee

# verbose, skip QR (manual entry of name/password on device)
./adbee -v -no-qr -name ADB_WIFI_1234 -password 5678
```

Flags:

| Flag            | Type     | Description                                        | Default              |
| --------------- | -------- | -------------------------------------------------- | -------------------- |
| `-attempts`     | int      | Max discovery attempts before giving up            | 30                   |
| `-connect-wait` | duration | Max time to wait for connect service after pairing | 12s                  |
| `-name`         | string   | Device name to advertise                           | random ADB*WIFI*<id> |
| `-password`     | string   | Pairing password                                   | random               |
| `-sleep`        | duration | Sleep between discovery attempts                   | 1s                   |
| `-timeout`      | duration | Per-query mDNS timeout                             | 2s                   |

## See also

- https://github.com/eeriemyxi/lyto (Python)
- https://github.com/Vazgen005/adb-wifi-py (Python)
- https://github.com/saleehk/adb-wifi (JS)
- https://github.com/AngelKrak/adb-qc (JS)
