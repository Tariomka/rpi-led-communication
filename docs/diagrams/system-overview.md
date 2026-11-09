# System overview

```mermaid
flowchart LR
    subgraph LAN["Wi-Fi LAN (2.4 GHz)"]
        Desktop["Desktop application"]
        Router["Access point / DHCP"]
    end

    subgraph Pico["Raspberry Pi Pico W"]
        CYW["CYW43439<br/>Wi-Fi"]
        RP["RP2040<br/>firmware"]
        LED["On-board LED<br/>(link up)"]
        CYW --- RP
        RP --- LED
    end

    subgraph Display["WeAct 2.8&quot; LCD module"]
        ILI["ILI9341<br/>320×240 LCD"]
        XPT["XPT2046<br/>touch"]
    end

    subgraph Cube["LED cube"]
        Core["Core board<br/>GD32F103RET6"]
        LEDs["LED matrix"]
        Core --- LEDs
    end

    PC["PC (USB)"]

    Desktop <-- "TCP :Port<br/>led-common-lib/tcp" --> Router
    Router <-- "802.11" --> CYW
    RP -- "SPI1 40 MHz<br/>GP6-GP9, GP10-GP12" --> ILI
    RP <-- "bit-banged SPI<br/>GP16-GP20, GND 23" --> XPT
    RP -- "GP14 VCC, GP15 BL" --> Display
    RP <-- "UART0 115200 8N1<br/>GP0 TX, GP1 RX, GND, GP2 on +5V (!)" --> Core
    Cube -- "5 V → VSYS (39), GND → 38" --> RP
    PC -. "USB: UF2 flash,<br/>CDC debug log" .-> RP
```

The complete wiring in schematic form: [full-schematic.svg](full-schematic.svg).

## Links

| Link | Physical | Protocol | Documentation |
|---|---|---|---|
| Desktop ↔ Pico | Wi-Fi station, TCP server on the Pico | `[ver][type][len u16][payload]` messages | [protocols.md](../documentation/protocols.md#desktop--pico-tcp) |
| Pico ↔ LCD | 10-pin JST 1.25 mm: SPI1 + RES/D-C/CS/FM, VCC and BL from GPIOs | ILI9341 4-wire SPI | [pico-w-display-wiring.md](pico-w-display-wiring.md) |
| Pico ↔ touch | 6-pin JST 1.25 mm: TSCK/TCS/TMOSI/TMISO/TPEN/GND | XPT2046, software SPI | [pico-w-display-wiring.md](pico-w-display-wiring.md) |
| Pico ↔ cube | 4-way UART0 cable (pins 1–4) + 5 V/GND from the base board (pins 39/38) | `#?cmd?!` console, text lines, `A5 05 …` frames | [pico-w-cube-wiring.md](pico-w-cube-wiring.md), [protocols.md](../documentation/protocols.md#pico--cube-uart) |
| PC ↔ Pico | micro-USB | UF2 mass storage, USB CDC serial | [README.md](../../README.md) |

## Power domains

```mermaid
flowchart LR
    USB["USB 5 V<br/>(PC, flashing / log only)"] -.-> VBUS["Pico VBUS (40)"]
    VBUS -. "Schottky on board" .-> VSYS["Pico VSYS (39)"]
    CubePSU["Cube USB-C 5 V"] --> Base["Cube base board"]
    Base --> CoreBoard["Cube core board<br/>5 V → 3.3 V (AMS1117)"]
    Base -- "5 V, GND → pin 38" --> VSYS
    VSYS --> Reg["Pico buck-boost<br/>3V3"]
    Reg --> RP["RP2040 + CYW43439"]
    RP -- "GP14 / GP15" --> LCD["LCD VCC / backlight"]
    GND["Common ground"] --- RP
    GND --- Base
    GND --- LCD
```

The Pico is powered by the cube's base board and runs whenever the cube does; USB is only attached for
flashing and the debug log. Details: [hardware.md](../documentation/hardware.md#power).
