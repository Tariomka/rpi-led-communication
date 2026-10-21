package component

import (
	"device/rp"
	"machine"
	"runtime"
	"runtime/volatile"
	"unsafe"
)

const displayDMAChannel = 1 // the channel TinyGo reserves for SPI1

type dmaChannel struct {
	READ_ADDR   volatile.Register32
	WRITE_ADDR  volatile.Register32
	TRANS_COUNT volatile.Register32
	CTRL_TRIG   volatile.Register32
	_           [12]volatile.Register32 // aliases
}

var dmaChannels = (*[12]dmaChannel)(unsafe.Pointer(rp.DMA))

// DisplaySPI sends on SPI1 by DMA like machine.SPI, but with DMA chaining off. TinyGo's SPI1
// transfers chain to DMA channel 0, which the Wi-Fi chip's PIO bus uses: every display transfer
// restarted a Wi-Fi transfer and stalled the Wi-Fi driver.
type DisplaySPI struct{ *machine.SPI }

func (this DisplaySPI) Tx(w, r []byte) error {
	if r != nil || len(w) == 0 || this.SPI != machine.SPI1 {
		return this.SPI.Tx(w, r)
	}

	channel := &dmaChannels[displayDMAChannel]
	channel.READ_ADDR.Set(uint32(uintptr(unsafe.Pointer(unsafe.SliceData(w)))))
	channel.WRITE_ADDR.Set(uint32(uintptr(unsafe.Pointer(&rp.SPI1.SSPDR))))
	channel.TRANS_COUNT.Set(uint32(len(w)))
	channel.CTRL_TRIG.Set(rp.DMA_CH0_CTRL_TRIG_INCR_READ |
		rp.DMA_CH0_CTRL_TRIG_DATA_SIZE_SIZE_BYTE<<rp.DMA_CH0_CTRL_TRIG_DATA_SIZE_Pos |
		rp.DREQ_SPI1_TX<<rp.DMA_CH0_CTRL_TRIG_TREQ_SEL_Pos |
		displayDMAChannel<<rp.DMA_CH0_CTRL_TRIG_CHAIN_TO_Pos | // chaining to itself disables it
		rp.DMA_CH0_CTRL_TRIG_EN)
	for channel.CTRL_TRIG.Get()&rp.DMA_CH0_CTRL_TRIG_BUSY != 0 {
	}
	runtime.KeepAlive(w)

	// Nothing was read while sending, so drain the RX FIFO and clear its overrun flag.
	bus := rp.SPI1
	for bus.SSPSR.HasBits(rp.SPI0_SSPSR_RNE) {
		bus.SSPDR.Get()
	}
	for bus.SSPSR.HasBits(rp.SPI0_SSPSR_BSY) {
	}
	for bus.SSPSR.HasBits(rp.SPI0_SSPSR_RNE) {
		bus.SSPDR.Get()
	}
	bus.SSPICR.Set(rp.SPI0_SSPICR_RORIC)
	return nil
}
