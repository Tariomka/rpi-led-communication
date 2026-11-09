# Hardware and infrastructure

## Parts

| Part | Role | Notes |
|---|---|---|
| Raspberry Pi Pico W (RP2040 + CYW43439) | Runs the firmware; Wi-Fi station, TCP server, UART host for the cube, display driver | 3.3 V logic on all GPIOs. [Datasheet](../manual/pico-w-datasheet.pdf) |
| WeAct 2.8" LCD module (ILI9341 + XPT2046) | 320×240 touchscreen for status, control buttons, log and upload progress | 3.3 V module, 10-pin display connector (SPI) and 6-pin touch connector (bit-banged SPI), both JST 1.25 mm. [Module schematic](../diagrams/WeAct-2.8InchLCDModule_V10.pdf), [ILI9341](../manual/ILI9341.pdf), [XPT2046](../manual/xpt2046-datasheet.pdf) |
| LED cube core board (GD32F103RET6) | The cube's controller; a 4-way serial cable to the Pico | 3.3 V logic UART, 115200 baud 8N1; the cube's base board also powers the Pico (5 V / GND) |
| Desktop application | Connects to the Pico over TCP, sends commands and show uploads, shows the logs | Any machine on the same LAN |
| USB (optional) | Flashing (UF2 mass storage) and the debug log (USB CDC serial) | `go run . monitor` |

The overall topology is drawn in [system-overview.md](../diagrams/system-overview.md); the complete
wiring as a schematic is [full-schematic.svg](../diagrams/full-schematic.svg).

## Pico W pin usage

All pin assignments come from the firmware (`internal/component/screen.go`, `internal/component/pin.go`,
`internal/controller/pico_w.go`). "Pin" is the physical pin number of the Pico W header.

### UART0 — cube link

| GPIO | Pin | Direction | Cube header | Comment |
|---|---|---|---|---|
| GP0 (UART0 TX) | 1 | Pico → cube | RXD | Console commands `#?…?!` and upload frames |
| GP1 (UART0 RX) | 2 | cube → Pico | TXD | Log lines, `Status:` replies, upload acknowledgements |
| GND | 3 | — | GND | Common ground is mandatory |
| GP2 | 4 | — | +5V | Wired so the 4-way cable is fully seated; **this puts the cube's 5 V on GP2** — see the warning below |

UART0 is configured for 115200 baud, 8 data bits, no parity, 1 stop bit. No level shifting is needed:
both sides are 3.3 V CMOS. The Pico is powered from the cube's base board: 5 V to VSYS (pin 39) and
GND to pin 38. Wiring and power: [pico-w-cube-wiring.md](../diagrams/pico-w-cube-wiring.md).

> **Warning — GP2 on 5 V.** The 4th wire of the serial cable connects the cube's +5 V pin to Pico pin 4
> (GP2). RP2040 GPIOs are not 5 V tolerant (absolute maximum is IOVDD + 0.5 V ≈ 3.8 V); the pad's
> protection diode conducts into the 3.3 V rail. The firmware never configures GP2, but the pin should be
> cut out of the cable or the cable position left empty.

### SPI1 — ILI9341 display (10-pin JST 1.25 mm connector)

| GPIO | Pin | Direction | Module pin | Comment |
|---|---|---|---|---|
| GP14 | 19 | Pico → LCD | VCC | Driven high at start: the firmware powers the panel logic from this pin |
| GND | 18 | — | GND | |
| GP15 | 20 | Pico → LCD | BL | Backlight, driven high at start |
| GP6 | 9 | Pico → LCD | RES | Hardware reset, pulsed by the driver at start |
| GP12 (SPI1 RX) | 16 | LCD → Pico | SDO | Claimed by the SPI peripheral, never read |
| GP11 (SPI1 TX) | 15 | Pico → LCD | SDA | Pixel and command data (MOSI) |
| GP7 | 10 | Pico → LCD | D/C | Data (high) / command (low) select |
| GP10 (SPI1 SCK) | 14 | Pico → LCD | SCL | 40 MHz, SPI mode 2 (CPOL = 1, CPHA = 0) |
| GP8 | 11 | Pico → LCD | CS | Chip select, active low |
| GP9 | 12 | — | FM | Read strobe (RDX); wired, not driven |

### Bit-banged SPI — XPT2046 touch controller (6-pin JST 1.25 mm connector)

| GPIO | Pin | Direction | Module pin | Comment |
|---|---|---|---|---|
| GP18 | 24 | Pico → touch | TSCK | Clock |
| GP17 | 22 | Pico → touch | TCS | Chip select, active low |
| GP19 | 25 | Pico → touch | TMOSI | Data to the XPT2046 |
| GP16 | 21 | touch → Pico | TMISO | Data from the XPT2046 |
| GP20 | 26 | touch → Pico | TPEN | Pen-down, low while touched |
| GND | 23 | — | GND | Touch controller ground return |

The touch driver samples by software (4 samples averaged per read), so every read blocks the main
loop for a short time; the UI polls the touch controller every 25 ms.

Pin-by-pin drawing: [pico-w-display-wiring.md](../diagrams/pico-w-display-wiring.md).

### On-board peripherals

| Peripheral | Use |
|---|---|
| CYW43439 (Wi-Fi) | Station mode, 2.4 GHz only; driven over the PIO SPI bus by the `cyw43439` driver, which uses DMA channel 0 |
| CYW43439 GPIO 0 | The Pico W's LED: off while connecting, blinks once after joining and once after the listener is up, then steady on while the desktop link is available |
| USB CDC | Debug log at `slog.LevelDebug`; also the UF2 bootloader for flashing (hold BOOTSEL while plugging in) |
| Watchdog | Armed after start-up with a 5 s timeout, fed only by the main loop |
| Brown-out detector | Enabled at 0.946 V on the 1.1 V core rail (default 0.860 V), so a sagging supply resets instead of corrupting |
| SRAM4 (0x20040000, 4 KB) | Outside the TinyGo heap; keeps the last-run record (stage, uptime, panic text) through a watchdog reset |

## Power

| Consumer | Supply |
|---|---|
| Pico W | 5 V from the cube's base board into VSYS (pin 39), GND on pin 38 — the Pico starts and stops with the cube. USB (VBUS, pin 40) is used while flashing or watching the log |
| LCD module | VCC and BL are driven from GP14 and GP15 by the firmware. The RP2040 pads are limited (see the electrical characteristics in the [Pico W datasheet](../manual/pico-w-datasheet.pdf)); if the module's backlight current exceeds that, feed VCC/BL from 3V3(OUT) instead — GP14/GP15 then simply drive nothing |
| Cube | Its own 5 V supply (USB-C on the cube) |

When the Pico is on USB and on the cube's 5 V at the same time, put a Schottky diode in series with
the 5 V lead into VSYS as the Pico W datasheet recommends, so the two supplies never fight. See
[pico-w-cube-wiring.md](../diagrams/pico-w-cube-wiring.md).

Ground must be common between the Pico, the display and the cube.

## Peripheral sharing on the RP2040

Two details of the Pico's internals shape the wiring and the firmware:

- **DMA**: TinyGo's SPI1 transmit uses DMA channel 1 and chains to channel 0, which is the channel the
  Wi-Fi driver's PIO bus uses. Every display transfer would retrigger a Wi-Fi transfer and stall the
  Wi-Fi driver while the screen draws. `component.DisplaySPI` therefore programs channel 1 itself with
  chaining disabled (chain-to = itself). Keep the display on SPI1 and the Wi-Fi chip on its default
  PIO; moving either changes this assumption.
- **UART0 RX buffer**: the cube's lines are read from the RP2040's UART FIFO/ring buffer by the main loop.
  Because the display is drawn in strips, the cube service is polled between strips (`Display.SetYield`)
  so a full redraw does not overflow the receive buffer.

## Physical layout

```
                      USB (flash / debug log)
                              │
   2.8" LCD + touch ──SPI1──► Pico W ◄──UART0──► LED cube core board (GD32F103RET6)
   (GP6..GP12, GP14..GP20)      │   (GP0..GP2, GND)
                              Wi-Fi    5 V / GND from the cube base board (pins 39 / 38)
                                │
                             Desktop
```

Photos of the assembled unit: [pico-and-display-on-cube-1.jpg](../diagrams/pico-and-display-on-cube-1.jpg),
[-2.jpg](../diagrams/pico-and-display-on-cube-2.jpg), [-3.jpg](../diagrams/pico-and-display-on-cube-3.jpg).
