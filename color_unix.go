// +build !windows linux,darwin

package main

import (
	"github.com/lucasb-eyer/go-colorful"
	"github.com/nsf/termbox-go"
)

const termboxXtermOffset = 0x11

// modification of https://github.com/ichinaski/pxl/blob/master/color.go.

// reduceRGB reduces color values to the range [0, 15].
func reduceRGB(color colorful.Color) (uint16, uint16, uint16) {
	r, g, b, _ := color.RGBA()

	return uint16(r >> 8), uint16(g >> 8), uint16(b >> 8)
}

// termColor converts a 24-bit RGB color into a term256 compatible approximation.
func colorToTermboxAttr(color colorful.Color) termbox.Attribute {
	r, g, b := reduceRGB(color)

	var (
		termR = (((r * 5) + 127) / 255) * 36
		termG = (((g * 5) + 127) / 255) * 6
		termB = (((b * 5) + 127) / 255)
	)

	return termbox.Attribute(termR + termG + termB + termboxXtermOffset)
}
