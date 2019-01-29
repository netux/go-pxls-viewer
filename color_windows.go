// +build windows

package main

import (
	"github.com/lucasb-eyer/go-colorful"
	"github.com/nsf/termbox-go"
)

// TODO(netux): wait for Termbox to add support for Output256 on Windows.
var termboxPalette = map[colorful.Color]termbox.Attribute{
	colorful.Color{R: 0, G: 0, B: 0}: termbox.ColorBlack,
	colorful.Color{R: 1, G: 0, B: 0}: termbox.ColorRed,
	colorful.Color{R: 0, G: 1, B: 0}: termbox.ColorGreen,
	colorful.Color{R: 0, G: 1, B: 0}: termbox.ColorBlue,
	colorful.Color{R: 1, G: 1, B: 0}: termbox.ColorYellow,
	colorful.Color{R: 1, G: 0, B: 1}: termbox.ColorMagenta,
	colorful.Color{R: 0, G: 1, B: 1}: termbox.ColorCyan,
	colorful.Color{R: 1, G: 1, B: 1}: termbox.ColorWhite,
}

func colorToTermboxAttr(color colorful.Color) termbox.Attribute {
	var closest colorful.Color
	var by float64 = 2
	for c := range termboxPalette {
		d := color.DistanceLab(c)

		if d < by {
			closest = c
			by = d
		}
	}

	return termboxPalette[closest]
}
