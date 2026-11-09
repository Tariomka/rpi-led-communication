# rpi-led-communication documentation

Firmware for a Raspberry Pi Pico W that sits between a desktop application and an LED cube:
the desktop talks to the Pico over Wi-Fi (TCP), the Pico talks to the cube's controller
(GD32F103RET6) over UART, and a 2.8" ILI9341 touchscreen on the Pico shows the cube's state and
gives local control.

| Document | Content |
|---|---|
| [hardware.md](hardware.md) | Infrastructure: the boards, how they are wired and powered, every pin used |
| [architecture.md](architecture.md) | Software architecture: packages, start-up, main loop, goroutines, memory, logging, restarts |
| [protocols.md](protocols.md) | The desktop ↔ Pico TCP protocol and the Pico ↔ cube UART protocol as the firmware uses them |

Diagrams (Mermaid, rendered by GitHub and VS Code):

| Diagram | Content |
|---|---|
| [full-schematic.svg](../diagrams/full-schematic.svg) | Full schematic of the assembled unit: Pico W, the LCD module's 10-pin display and 6-pin touch headers, the cube's serial cable and power, with hops marking crossing wires |
| [pico-and-display-on-cube-*.jpg](../diagrams/pico-and-display-on-cube-1.jpg) | Photos of the assembled Pico W and display on the cube (1–3) |
| [system-overview.md](../diagrams/system-overview.md) | Desktop, Pico W, display and cube and the links between them |
| [pico-w-display-wiring.md](../diagrams/pico-w-display-wiring.md) | Pico W ↔ 2.8" ILI9341 / XPT2046 module, pin by pin |
| [pico-w-cube-wiring.md](../diagrams/pico-w-cube-wiring.md) | Pico W ↔ cube core board (GD32F103RET6) serial link and power |

Manuals and vendor drawings kept in this repository:

| File | What it is |
|---|---|
| [pico-w-datasheet.pdf](../manual/pico-w-datasheet.pdf) | Raspberry Pi Pico W board datasheet: pinout, power, electrical limits |
| [ILI9341.pdf](../manual/ILI9341.pdf) | ILI9341 LCD controller datasheet (SPI interface, commands) |
| [xpt2046-datasheet.pdf](../manual/xpt2046-datasheet.pdf) | XPT2046 resistive touch controller datasheet |
| [WeAct-2.8InchLCDModule_V10.pdf](../diagrams/WeAct-2.8InchLCDModule_V10.pdf) | Schematic of the 2.8" LCD module used (ILI9341 + XPT2046) |

Building, flashing and configuration are described in the repository [README.md](../../README.md).
