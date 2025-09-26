package common

import "image/color"

const (
	full  = 255
	light = 192
	half  = 128
	dull  = 64
	none  = 0
)

var (
	ColorBlack     = color.RGBA{none, none, none, full}
	ColorNavy      = color.RGBA{none, none, half, full}
	ColorWhite     = color.RGBA{full, full, full, full}
	ColorRed       = color.RGBA{full, none, none, full}
	ColorBlue      = color.RGBA{none, none, full, full}
	ColorGreen     = color.RGBA{none, full, none, full}
	ColorDarkGreen = color.RGBA{none, half, none, full}
	ColorDarkCyan  = color.RGBA{none, half, half, full}
	ColorMaroon    = color.RGBA{half, none, none, full}
	ColorPuple     = color.RGBA{half, none, half, full}
	ColorOlive     = color.RGBA{half, half, none, full}
	ColorLightGray = color.RGBA{light, light, light, full}
	ColorDarkGray  = color.RGBA{half, half, half, full}
	ColorCyan      = color.RGBA{none, full, full, full}
	ColorMagenta   = color.RGBA{full, none, full, full}
	ColorYellow    = color.RGBA{full, full, none, full}
	ColorOrange    = color.RGBA{full, half, none, full}
	ColorVomit     = color.RGBA{light, full, dull, full}
)
