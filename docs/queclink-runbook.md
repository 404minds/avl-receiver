# Queclink GV200 / GT500 — operator runbook

Receiver package: `internal/protocols/queclink/`. Design, wire formats and execution history: `docs/queclink-implementation-plan.md`.
Scope: ASCII @Track over TCP, positions + events, heartbeat replies, buffered replay. No server→device commands, no HEX mode.

## 1. Deploy order (enum dependency)

`DeviceType QUECLINK_GV200=6 / QUECLINK_GT500=7` and `DeviceProtocolType QUECLINK=6` exist in **both** protos.
`VerifyDevice` replies with the new enum, and the receiver rejects the device unless its own build knows it.

1. **findnsecure consumer** (`fns-consumer-grpc-server`) — new proto + `deviceTypeFromModel` + GT500 event filter.
2. **avl-receiver** — same deploy window (`deployment/ansible/playbooks/receiver.yaml`; no config change, port stays `21000`,
   `CUSTOM_REMOTE_STORE_ADDR='fns-consumer-grpc-server:8000'` unchanged).
3. **findnsecure api** — `trackerModelList` gains `Queclink - GV200` / `Queclink - GT500` (dropdown only; the API never validated the list).

Rollback / kill switch: remove `types.DeviceProtocolType_QUECLINK` from `allowedProtocols` in `internal/handlers/handlers.go` and
redeploy the receiver — Queclink connections are declined at sniff time, nothing else changes. Enum values are additive; a half
rolled-back pair still builds.

## 2. Onboard the device in findnsecure (before it connects)

Create the Vehicle with the device **IMEI**, `deviceModel` = `Queclink - GT500` or `Queclink - GV200` (dropdown), and a **VIN**
(mandatory + unique; a GT500 has none — use the IMEI or `QL-<IMEI>`). A registration plate is optional (the consumer used to fail on
a NULL `regnPlateNo`; fixed in this change). If the row is missing when the device connects,
`VerifyDevice` fails and the receiver logs `Failed to verify device` / `unauthorized` and closes the socket.

## 3. Provision the device (SMS or USB tool; the platform cannot send commands)

Replace `<APN>`, `<user>`, `<pwd>`, `<host>`. Password defaults: `gt500` / `gv200`. Count numbers (`0001`…) are free.

**GT500**
```
AT+GTQSS=gt500,<APN>,<user>,<pwd>,3,0,1,<host>,21000,,,,10,0,,,0001$
AT+GTFRI=gt500,1,1,,,0000,0000,,,30,30,,,1000,0,30,100,5,,,0002$
AT+GTNMD=gt500,6,2,3,,,,,,,,,,,0003$
```
report mode 3 = TCP long-connect · bearer 0 = GPRS · buffer 1 · heartbeat 10 min · SACK 0 · fixed report every 30 s, one point per message.
**`AT+GTNMD` is mandatory:** mode 6 makes the motion sensor report `+RESP:GTNMR` on every still↔moving change, and keeps
motion detection on so `+RESP:GTSTT 41/42` fire. With factory defaults the GT500 turns motion detection **off** (state `99`
in GTINF) and the platform can only fall back to GPS speed > 3 km/h for moving/stopped. Do not set bit0 (suspends GTFRI while still).

**GV200**
```
AT+GTBSI=gv200,<APN>,<user>,<pwd>,,,,,0001$
AT+GTSRI=gv200,3,,1,<host>,21000,,,,5,0,0,,,,0002$
AT+GTFRI=gv200,1,1,0,1,0000,0000,30,30,,,3F,0,600,00000000,,0003$
AT+GTRTO=gv200,8,,,,,,0004$
```
mode 3 TCP long-connect · buffer 1 · heartbeat 5 min · SACK 0 · **Protocol Format 0 = ASCII** · ERI mask 0 (GTFRI, not GTERI) ·
`GTRTO 8` requests `+RESP:GTVER` — record firmware and the version prefix (`04` or `35` expected).
Speed alarm, if wanted: `AT+GTSPD=gv200,1,<limit>,400,...` (mode 1 = alarm while inside the range → every GTSPD is an over-speed).

Leave `AT+GTCFG` report-items mask at default `003F`. Do not enable HEX mode (`Protocol Format 1`).

## 4. What to expect in the receiver log

| Log line | Meaning |
| --- | --- |
| `queclink: imei 8…, protocol version 070002 (prefix 07 => QUECLINK_GT500)` then `Login successful … deviceType QUECLINK_GT500` | identified + authorised |
| `version prefix "04" suggests QUECLINK_GV200 but device is registered as QUECLINK_GT500` | wrong `deviceModel` on the Vehicle — fix the row; parsing follows the registered type |
| `unknown version prefix "xx"` | new firmware prefix — add to `modelByPrefix` in `queclink.go` (WARN only, nothing breaks) |
| `queclink: HEX-mode header 2B 52 53 50 …` | device provisioned with Protocol Format 1 — reprovision `AT+GTSRI … ,0,` (ASCII) |
| `Failed to verify device` / `Device is not authorized` | no Vehicle row for the IMEI, or `deviceModel` brand is not `Queclink` |
| `+RESP:GTFRI: 24 fields, expected 30` | layout mismatch — usually a GT500 registered as GV200 (or vice versa) |
| `headers seen but not decoded on this connection: map[+RESP:GTGSM:3 …]` | summary on disconnect; harmless, tells you which report types the device sends that we ignore |
| `point 0 has no fix, skipped` | device has no GPS fix yet (cold start) — wait |

Heartbeats: every `+ACK:GTHBD` gets `+SACK:GTHBD,<ver>,<count>$` back; nothing is stored for them.

## 5. First-device checklist (plan §15.12)

1. Power on → `+RESP:GTPNA` (ignored) then first `+RESP:GTFRI` → position in the web UI within one report interval.
2. GT500 walk test → `moving` / `stopped`, never `idling`; no `ignition_on/off` notifications. Confirm in the receiver log
   that `+RESP:GTSTT,…,42,` / `+RESP:GTNMR,…,1,` arrive when you start walking and `41` / `0` when you stop; if GTINF shows
   state `99`, `AT+GTNMD` was not applied. Leave it parked 1 h and note the max GPS speed in the raw log (should stay < 3 km/h).
3. GV200: ignition on/off on the bench → `+RESP:GTIGN` / `+RESP:GTIGF` (and `GTIGL`, `GTSTT 21/11`) → `ignition` follows them,
   `ignition_on` / `ignition_off` events appear; record which digital input is wired to ignition.
4. Cut GPRS 20 min → restore → `+BUFF:` lines; Scylla rows upsert on identical GPS time (no duplicates).
5. Set send interval = 2 × check interval → `Number=2` messages → 2 rows each.
6. Trigger SOS (GT500 button / GV200 input) → `sos_button_pressed` event; tow the GV200 → `towing`.
7. Save 24 h of raw lines (`tcpdump -A port 21000` or the receiver INFO log) to `docs/fixtures/<model>-<imei>-<date>.txt` and add
   them to `queclink_test.go`.

## 6. Local testing without a device

```
mkdir -m 755 logs                                   # pre-existing bug: the receiver creates ./logs with mode 000
go run cmd/receiver/receiver.go -port 21000 -storeType local -remoteStoreAddr x
docs/scripts/queclink-replay.sh docs/fixtures/gv200-doc-examples.txt      # one connection per line
docs/scripts/queclink-replay.sh --split 7 docs/fixtures/gt500-doc-examples.txt
docs/scripts/queclink-replay.sh --concat docs/fixtures/gv200-events.txt   # ignition/alarm story
```
Local store registers the device as `GetDeviceTypesForProtocol(QUECLINK)[0]` = GV200, so GT500 fixtures parse with the GV200
layout there (expected WARN). Records land in `./logs/<imei>.json`.

Full stack (plan IT6–IT11, passed 2026-08-26 — plan §20): in `findnsecure/deployment` run
`docker compose -p qltest up -d postgres scylla redis` (isolated project → fresh volumes; the shared dev volumes are incompatible
with the current Scylla image), `DATABASE_URL=postgresql://findnsecure:AhZUE5ur4DKL@localhost:5432/findnsecure npx prisma migrate deploy`
in `backend` (stops at a pre-existing broken migration after creating the tables we need), `docker compose -p qltest up -d --build consumer`,
insert a Company + Vehicle rows with psql, start the receiver with `-storeType remote -remoteStoreAddr localhost:8000`, replay
fixtures, then in `cqlsh`: `SELECT … FROM vehicle_data.vehicle_tracking_1 WHERE vehicle_id='…' ALLOW FILTERING` and
`vehicle_event_1`; `vehicle_status` lives in Redis key `<vehicleId>_avl`. Allow ~10 s per record (geocoding/notification timeouts).
Teardown: `docker compose -p qltest down -v`.

Tests: `go test ./internal/protocols/queclink/ ./internal/store/` and
`go test -run='^$' -fuzz=FuzzParseReport -fuzztime=30s ./internal/protocols/queclink/`.

## 7. Commands (Phase 5)

The platform's command tab sends the literal strings `ignition_on` / `ignition_off` or a custom string. For a **GV200** the two
presets become `AT+GTOUT=gv200,<out1>,0,0,0,0,0,0,0,0,0,0,0,,,,,<serial>$` (output 1: `1` = immobilised, `0` = released). Anything
else is sent verbatim with `$` appended, so operators can send any @Track command (e.g. `AT+GTRTO=gv200,8,,,,,,0004` for GTVER).
Every `+ACK:GTxxx` the device answers is stored on the command in `vehicle_data.command_responses` (status SUCCESS, response = ack).
The **GT500 has no outputs**: the presets are refused (`has no controllable output`); custom commands work.

**Before exposing the presets:** confirm on the bench which output the immobiliser relay is wired to and whether `1` cuts the
engine (`immobiliserOutput` / `immobiliserActive` in `queclink.go`). The device password is assumed to be the factory `gv200`.
The consumer dials the receiver as `avl-receiver:15000` — that hostname must resolve on the consumer's network.

## 8. Known limits

- GT500 has no ignition: its motion sensor (`GTSTT 41/42`, `GTNMR 0/1`) is the moving/stopped signal, GPS speed > 3 km/h only
  until the first state arrives (or after state `99`); `ignition_on/off` events are suppressed and `idling` is mapped to
  `moving` in the consumer for `QUECLINK_GT500`.
- GT500 `GTBPL` (low battery, volts) and `GTMSA` (fall) produce no event.
- Device-side geofence flags are stored but not turned into platform geofence events.
- GTERI: the **first 1-wire temperature sensor** is stored as `temperature` (°C); further sensors, the digital fuel-sensor field
  (no documented unit), cell IDs, Wi-Fi MAC/RSSI, GSM level, analog inputs and DI/DO bitmasks survive only in `raw_data`.
  To get temperatures the GV200 must be provisioned with ERI mask bit1 (`AT+GTFRI … ,00000002,`) and an AC100 on the UART.
- Records queued at the instant a socket closes are now flushed (store drain); idle sockets are dropped after 30 min without data.

## 9. The client's tstGW units (platform model "GL300", protocol id 280518)

Source of truth: the client's `Tracker data.xlsx` (2021 sample + field map) — copied into
`internal/protocols/queclink/testdata/` with a production capture; `TestClientSampleAndProductionLines` replays both.

- Prefix `28` is Traccar's GL300VC, but these units send the **GT500** report shapes (`GTFRI/GTSOS`: …,MAC,RSSI,Odo,Batt%,SendTime,Count;
  `GTSTC/GTBTC`: …,Reserved,MAC,RSSI,SendTime,Count) with the `Number` field blank, and a **GV300W**-shaped `GTDIS`
  (…,Reserved,ReportID/Type,Number,point,Reserved,Mileage,SendTime,Count). `specByPrefix["28"]` therefore points at `gwSpec`.
- `<Device name>` carries the status: `<method> S<gsm 0-31> V<sats> B<batt%> C<charging> H<accuracy> v<fw>` (2021 firmware) or
  ` tstGW v<fw> S<gsm> B<batt>% C<n>|<method> V<sats> H<acc> A<alt>m D<heading> <speed>kph <odo>m|` (2026). S/V/B are mapped to
  `gsm_network`, `satellites`, `battery_level`; method (G/B/W/D, `O` prefix = old fix) and charging state stay in `raw_data`.
- Odometer is in **metres** (the 2026 name says `11634m`; the 2021 sample's deltas equal the GPS distance in metres) → stored as km.
- Every report arrives on its own TCP connection, so nothing cached per connection (motion state, battery, GSM) carries over.
- `GTDIS` ReportID/Type per the client: 90 check-out (bottom button), 91 check-in (top button), 93 welfare-timer timeout, 51 fall alert;
  production also sends 71 (undocumented). All raise `inputs_triggering` only — mapping them to platform events is a product decision.
- Send time in these reports is the client's server-local time, not UTC; the record timestamp is the GPS UTC field.
