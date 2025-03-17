package controller

import (
	"iter"

	"github.com/Tariomka/rpi-led-communication/internal/common"
)

type Color uint8

const (
	NoColor Color = 0b0
	Green   Color = 0b1
	Blue    Color = 0b10
	Red     Color = 0b100
	Cyan    Color = 0b11
	Yellow  Color = 0b101
	Violet  Color = 0b110
	White   Color = 0b111
)

const (
	none byte = 0b00000000
	all  byte = 0b11111111
)

type LayoutWorker interface {
	LedSingleWorker
	LedRowIndividualWorker
	LedRowWorker
	LedLayerWorker
	LedBlockWorker
	Slicer
}

type LedSingleWorker interface {
	ChangeSingle(x, y, z uint8, c Color) error
	SetSingle(x, y, z uint8, c Color) error
	ResetSingle(x, y, z uint8) error
}

type LedRowWorker interface {
	ChangeRow(y, z uint8, c Color) error
	SetRow(y, z uint8, c Color) error
	ResetRow(y, z uint8) error
}

type LedRowIndividualWorker interface {
	ChangeRowIndividual(y, z uint8, c Color, values byte) error
	SetRowIndividual(y, z uint8, c Color, values byte) error
	ResetRowIndividual(y, z uint8, values byte) error
}

type LedLayerWorker interface {
	ChangeLayer(z uint8, c Color) error
	SetLayer(z uint8, c Color) error
	ResetLayer(z uint8) error
}

type LedBlockWorker interface {
	ChangeBlock(c Color)
	SetBlock(c Color)
	ResetBlock()
}

type Slicer interface {
	IterateSlices() iter.Seq2[uint8, []byte]
}

type Frame func(LayoutWorker) // Single frame of a light show
type LightShow []Frame        // Collection of light show frames

// State representation of all led colors of the cube.
//
// There are 8 layers, each having 64 leds, each having 3 contacts (GBR),
// making a single layer hold 192 bits of information.
//
// 12 shift registers control all 192 activations, each shift register having 16 activations
// and all shift registers connected in series.
// Shift registers receive data in little-endian (Least significant bit first)
//
// First 8 bytes [0-7] control the Green color, next 8 [8-15] - Blue, last 8 [16-23] - Red
// First byte in a color block controlls the back side of the layer, last (8'th) byte - the front.
// Bits from right to left control each led from right to left led,
// i.e. 0b00000001 turns on the right most led, 0b10000000 - left most led.
type LedLayout [8][24]byte

func (ll *LedLayout) IterateSlices() iter.Seq2[uint8, []byte] {
	return func(yield func(i uint8, v []byte) bool) {
		for zAxis := uint8(0); zAxis < 8; zAxis++ {
			if !yield(zAxis, ll[zAxis][:]) {
				return
			}
		}
	}
}

func (ll *LedLayout) ChangeSingle(x, y, z uint8, c Color) error {
	if err := validateAxes(x, y, z); err != nil {
		return err
	}

	ll.resetBit(x, y, z)
	if c != NoColor {
		ll.setBit(x, y, z, c)
	}
	return nil
}

func (ll *LedLayout) SetSingle(x, y, z uint8, c Color) error {
	if err := validateAxes(x, y, z); err != nil {
		return err
	}

	if c == NoColor {
		ll.resetBit(x, y, z)
		return nil
	}

	ll.setBit(x, y, z, c)
	return nil
}

func (ll *LedLayout) ResetSingle(x, y, z uint8) error {
	if err := validateAxes(x, y, z); err != nil {
		return err
	}

	ll.resetBit(x, y, z)
	return nil
}

func (ll *LedLayout) ChangeRowIndividual(y, z uint8, c Color, rowValues byte) error {
	if err := validateRow(y, z); err != nil {
		return err
	}

	ll.resetByte(y, z, rowValues)
	if c != NoColor {
		ll.setByte(y, z, c, rowValues)
	}
	return nil
}

func (ll *LedLayout) SetRowIndividual(y, z uint8, c Color, rowValues byte) error {
	if err := validateRow(y, z); err != nil {
		return err
	}

	if c == NoColor {
		ll.resetByte(y, z, rowValues)
		return nil
	}

	ll.setByte(y, z, c, rowValues)
	return nil
}

func (ll *LedLayout) ResetRowIndividual(y, z uint8, rowValues byte) error {
	if err := validateRow(y, z); err != nil {
		return err
	}

	ll.resetByte(y, z, rowValues)
	return nil
}

func (ll *LedLayout) ChangeRow(y, z uint8, c Color) error {
	if err := validateRow(y, z); err != nil {
		return err
	}

	ll.resetByte(y, z, all)
	if c != NoColor {
		ll.setByte(y, z, c, all)
	}
	return nil
}

func (ll *LedLayout) SetRow(y, z uint8, c Color) error {
	if err := validateRow(y, z); err != nil {
		return err
	}

	if c == NoColor {
		ll.resetByte(y, z, all)
		return nil
	}

	ll.setByte(y, z, c, all)
	return nil
}

func (ll *LedLayout) ResetRow(y, z uint8) error {
	if err := validateRow(y, z); err != nil {
		return err
	}

	ll.resetByte(y, z, all)
	return nil
}

func (ll *LedLayout) ChangeLayer(z uint8, c Color) error {
	if err := validateLayer(z); err != nil {
		return err
	}

	ll.resetBytes(z)
	if c != NoColor {
		ll.setBytes(z, c)
	}
	return nil
}

func (ll *LedLayout) SetLayer(z uint8, c Color) error {
	if err := validateLayer(z); err != nil {
		return err
	}

	if c == NoColor {
		ll.resetBytes(z)
		return nil
	}

	ll.setBytes(z, c)
	return nil
}

func (ll *LedLayout) ResetLayer(z uint8) error {
	if err := validateLayer(z); err != nil {
		return err
	}

	ll.resetBytes(z)
	return nil
}

func (ll *LedLayout) ChangeBlock(c Color) {
	ll.resetAll()
	if c != NoColor {
		ll.setAll(c)
	}
}

func (ll *LedLayout) SetBlock(c Color) {
	if c == NoColor {
		ll.resetAll()
		return
	}

	ll.setAll(c)
}

func (ll *LedLayout) ResetBlock() {
	ll.resetAll()
}

func (ll *LedLayout) setBit(x, y, z uint8, c Color) {
	for _, index := range layoutOffsetIndex(y, c) {
		ll.mutateByteWithOr(index, z, 1<<x)
	}
}

func (ll *LedLayout) resetBit(x, y, z uint8) {
	for _, index := range layoutOffsetIndex(y, White) {
		ll.mutateByteWithAnd(index, z, ^(1 << x))
	}
}

func (ll *LedLayout) setByte(y, z uint8, c Color, value byte) {
	for _, index := range layoutOffsetIndex(y, c) {
		ll.mutateByteWithOr(index, z, value)
	}
}

func (ll *LedLayout) resetByte(y, z uint8, value byte) {
	for _, index := range layoutOffsetIndex(y, White) {
		ll.mutateByteWithAnd(index, z, ^value)
	}
}

func (ll *LedLayout) setBytes(z uint8, c Color) {
	size := uint8(len(ll))
	for _, offset := range layoutOffsetIndex(0, c) {
		for index := offset; index < offset+size; index++ {
			ll.mutateByteWithOr(index, z, all)
		}
	}
}

func (ll *LedLayout) resetBytes(z uint8) {
	size := uint8(len(ll[z]))
	for index := uint8(0); index < size; index++ {
		ll.mutateByteWithAnd(index, z, none)
	}
}

func (ll *LedLayout) setAll(c Color) {
	size := uint8(len(ll))
	for layer := uint8(0); layer < size; layer++ {
		for _, offset := range layoutOffsetIndex(0, c) {
			for index := offset; index < offset+size; index++ {
				ll.mutateByteWithOr(index, layer, all)
			}
		}
	}
}

func (ll *LedLayout) resetAll() {
	size := uint8(len(ll))
	for layer := uint8(0); layer < size; layer++ {
		for index := uint8(0); index < size*3; index++ {
			ll.mutateByteWithAnd(index, layer, none)
		}
	}
}

func (ll *LedLayout) mutateByteWithAnd(index, layer, value uint8) {
	ll[layer][index] &= value
}

func (ll *LedLayout) mutateByteWithOr(index, layer, value uint8) {
	ll[layer][index] |= value
}

func layoutOffsetIndex(index uint8, c Color) []uint8 {
	offsets := []uint8{}

	for shift := uint8(0); shift < 3; shift++ {
		if c>>shift&1 == 1 {
			offsets = append(offsets, shift*8+index)
		}
	}

	return offsets
}

func validateAxes(x, y, z uint8) error {
	if x > 7 || y > 7 || z > 7 {
		return common.ErrOutOfBounds
	}
	return nil
}

func validateRow(y, z uint8) error {
	if y > 7 || z > 7 {
		return common.ErrOutOfBounds
	}
	return nil
}

func validateLayer(z uint8) error {
	if z > 7 {
		return common.ErrOutOfBounds
	}
	return nil
}
