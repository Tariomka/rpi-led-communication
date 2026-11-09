# Pico W ↔ 2.8" ILI9341 / XPT2046 module wiring

The module is the WeAct 2.8" LCD (ILI9341 controller, XPT2046 resistive touch controller, 3.3 V).
It has two JST 1.25 mm connectors: a 10-pin display connector and a 6-pin touch connector. Pin assignments are those used
by the firmware (`internal/component/screen.go`, `internal/component/pin.go`) and match the assembled
unit in [pico-and-display-on-cube-1.jpg](pico-and-display-on-cube-1.jpg) …
[-3.jpg](pico-and-display-on-cube-3.jpg). The complete drawing is [full-schematic.svg](full-schematic.svg).

References: [module schematic](WeAct-2.8InchLCDModule_V10.pdf), [ILI9341 datasheet](../manual/ILI9341.pdf),
[XPT2046 datasheet](../manual/xpt2046-datasheet.pdf), [Pico W datasheet](../manual/pico-w-datasheet.pdf).

## Connection diagram

```mermaid
flowchart LR
    subgraph Pico["Raspberry Pi Pico W"]
        direction TB
        GP14["GP14 (pin 19)"]
        GND18["GND (pin 18)"]
        GP15["GP15 (pin 20)"]
        GP6["GP6 (pin 9)"]
        GP12["GP12 SPI1 RX (pin 16)"]
        GP11["GP11 SPI1 TX (pin 15)"]
        GP7["GP7 (pin 10)"]
        GP10["GP10 SPI1 SCK (pin 14)"]
        GP8["GP8 (pin 11)"]
        GP9["GP9 (pin 12)"]
        GP20["GP20 (pin 26)"]
        GP19["GP19 (pin 25)"]
        GP18["GP18 (pin 24)"]
        GND23["GND (pin 23)"]
        GP17["GP17 (pin 22)"]
        GP16["GP16 (pin 21)"]
    end

    subgraph J1["J1 display connector (10-pin JST 1.25 mm) — ILI9341"]
        direction TB
        VCC["1 VCC"]
        LGND["2 GND"]
        BL["3 BL"]
        RES["4 RES"]
        SDO["5 SDO"]
        SDA["6 SDA"]
        DC["7 D/C"]
        SCL["8 SCL"]
        CS["9 CS"]
        FM["10 FM"]
    end

    subgraph J3["J3 touch connector (6-pin JST 1.25 mm) — XPT2046"]
        direction TB
        TSCK["1 TSCK"]
        TCS["2 TCS"]
        TMOSI["3 TMOSI"]
        TMISO["4 TMISO"]
        TPEN["5 TPEN"]
        TGND["6 GND"]
    end

    GP14 -- "3.3 V, driven high" --> VCC
    GND18 --- LGND
    GP15 -- "backlight on" --> BL
    GP6 -- "reset" --> RES
    SDO -. "never read" .-> GP12
    GP11 -- "MOSI 40 MHz" --> SDA
    GP7 -- "data/command" --> DC
    GP10 -- "SCK mode 2" --> SCL
    GP8 -- "chip select" --> CS
    GP9 -. "wired, not driven" .-> FM
    GP18 -- "clock" --> TSCK
    GP17 -- "chip select" --> TCS
    GP19 -- "MOSI" --> TMOSI
    TMISO -- "MISO" --> GP16
    TPEN -- "pen down (low)" --> GP20
    GND23 --- TGND
```

## Pin tables

### J1 — display connector (10-pin JST 1.25 mm)

| J1 pin | Module label | ILI9341 signal | Pico W GPIO | Pico W pin | Firmware |
|---|---|---|---|---|---|
| 1 | VCC | VCC / IOVCC (3.3 V) | GP14 (output, high) | 19 | "Main Power" |
| 2 | GND | GND | GND | 18 | |
| 3 | BL | Backlight anode | GP15 (output, high) | 20 | `backlight` |
| 4 | RES | RESX, hardware reset (active low) | GP6 | 9 | `reset` |
| 5 | SDO | SPI data out (MISO) | GP12, SPI1 RX | 16 | `SPI1_SDI_PIN`, never read |
| 6 | SDA | SPI data in (MOSI) | GP11, SPI1 TX | 15 | `SPI1_SDO_PIN` |
| 7 | D/C | Data (high) / command (low) | GP7 | 10 | `dc` |
| 8 | SCL | SPI clock | GP10, SPI1 SCK | 14 | `SPI1_SCK_PIN` |
| 9 | CS | CSX, chip select (active low) | GP8 | 11 | `cs` |
| 10 | FM | RDX, parallel-bus read strobe | GP9 | 12 | wired, not driven |

### J3 — touch connector (6-pin JST 1.25 mm)

| J3 pin | Module label | XPT2046 signal | Pico W GPIO | Pico W pin | Firmware |
|---|---|---|---|---|---|
| 1 | TSCK | DCLK | GP18 | 24 | `clk` |
| 2 | TCS | CS (active low) | GP17 | 22 | `cs` |
| 3 | TMOSI | DIN | GP19 | 25 | `din` |
| 4 | TMISO | DOUT | GP16 | 21 | `dout` |
| 5 | TPEN | PENIRQ (low while touched) | GP20 | 26 | `irq` |
| 6 | GND | GND | GND | 23 | |

J1 lands on Pico pins 9–12, 14–16 and 18–20 on the left edge; J3 on pins 21–26 on the right edge. Neither
connector is in Pico pin order, so the wires cross — [full-schematic.svg](full-schematic.svg) marks every
crossing with a hop.

## Pico W header view

```
                           ┌─────[USB]─────┐
   cube RXD   ◄── GP0   1 ─┤               ├─ 40 VBUS
   cube TXD   ──► GP1   2 ─┤               ├─ 39 VSYS  ◄── cube base board 5 V
   cube GND       GND   3 ─┤               ├─ 38 GND   ◄── cube base board GND
   cube +5V (!)   GP2   4 ─┤               ├─ 37 3V3_EN
                  GP3   5 ─┤               ├─ 36 3V3(OUT)
                  GP4   6 ─┤               ├─ 35 ADC_VREF
                  GP5   7 ─┤               ├─ 34 GP28
                  GND   8 ─┤               ├─ 33 GND
   J1-4  RES  ◄── GP6   9 ─┤               ├─ 32 GP27
   J1-7  D/C  ◄── GP7  10 ─┤               ├─ 31 GP26
   J1-9  CS   ◄── GP8  11 ─┤               ├─ 30 RUN
   J1-10 FM   ─── GP9  12 ─┤    Pico W     ├─ 29 GP22
                  GND  13 ─┤               ├─ 28 GND
   J1-8  SCL  ◄── GP10 14 ─┤               ├─ 27 GP21
   J1-6  SDA  ◄── GP11 15 ─┤               ├─ 26 GP20 ◄── J3-5 TPEN
   J1-5  SDO  ──► GP12 16 ─┤               ├─ 25 GP19 ──► J3-3 TMOSI
                  GP13 17 ─┤               ├─ 24 GP18 ──► J3-1 TSCK
   J1-2  GND  ─── GND  18 ─┤               ├─ 23 GND  ─── J3-6 GND
   J1-1  VCC  ◄── GP14 19 ─┤               ├─ 22 GP17 ──► J3-2 TCS
   J1-3  BL   ◄── GP15 20 ─┤               ├─ 21 GP16 ◄── J3-4 TMISO
                           └───────────────┘
```

## Electrical notes

- SPI1 runs at 40 MHz in mode 2 (CPOL = 1, CPHA = 0), the fastest the ILI9341 accepts over short wires.
  Keep the SCL/SDA leads short and of similar length.
- SDO is attached to the SPI1 RX pin because the peripheral needs the pin, but the firmware never reads
  from the panel; the RX FIFO is drained after each DMA transfer.
- FM (RDX) only matters for the parallel interface; it is wired to GP9 so the 10-way cable is fully
  seated, and GP9 is left unconfigured.
- The touch controller is clocked in software by the `xpt2046` driver (no hardware SPI), so its four
  signal pins can be any GPIOs; the firmware uses GP16–GP20. Its ground returns separately on pin 23.
- Pin 4 (GP2) of the cube's serial cable sits on the cube's +5 V: see the warning in
  [pico-w-cube-wiring.md](pico-w-cube-wiring.md#pin-table).
- VCC and BL are switched from GP14/GP15 so the panel starts in a known state. The RP2040 pad current
  is limited (see the Pico W datasheet); if the module's total draw with backlight exceeds it, feed
  VCC and BL from 3V3(OUT) (pin 36) instead and leave GP14/GP15 unconnected.
- The Wi-Fi chip shares DMA with the display: SPI1 transmit is done by DMA channel 1 with chaining
  disabled, because TinyGo's default chains it into channel 0, which the CYW43439 PIO bus uses. See
  [hardware.md](../documentation/hardware.md#peripheral-sharing-on-the-rp2040).
