package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/nsf/termbox-go"
)

type renderStrFunc func(r rune, x, y int)

const fgColor = termbox.ColorBlack
const bgColor = termbox.ColorWhite

var posX, posY int
var smX, smY = -1, -1
var pxls *Pxls

var debug string

func dbg(vals ...interface{}) {
	cw, _ := termbox.Size()

	var strs = make([]string, len(vals))
	for i, v := range vals {
		strs[i] = fmt.Sprint(v)
	}

	debug += strings.Join(strs, " ") + "\n"

	for strings.Count(debug, "\n") > cw {
		nlIdx := strings.Index(debug, "\n")
		debug = debug[nlIdx:]
	}
}

func main() {
	pxlsHost := flag.String("host", "pxls.space", "--host <pxls instance hostname>")
	pxlsHostSecure := flag.Bool("secure", true, "--secure <bool>")
	flag.Parse()

	pxls = NewPxls(*pxlsHost, *pxlsHostSecure)

	if err := initTermbox(); err != nil {
		panicClose(err)
	}
	defer close()

	// render first frame.
	if err := render(); err != nil {
		panicClose(err)
	}

	if err := pxls.Init(); err != nil {
		panicClose(err)
	}

	// loop blocks execution.
	if err := loop(); err != nil {
		panicClose(err)
	}
}

func initTermbox() error {
	err := termbox.Init()
	if err != nil {
		return err
	}

	if runtime.GOOS != "windows" {
		// TODO(netux): wait for Termbox to add support for Output256 on Windows.
		termbox.SetOutputMode(termbox.Output256)
	}
	termbox.SetInputMode(termbox.InputMouse)

	return nil
}

func panicClose(err interface{}) {
	close()
	panic(err)
}

func close() {
	if termbox.IsInit {
		termbox.Close()
	}
}

func loop() error {
	qCh := make(chan bool)
	evCh := make(chan termbox.Event)
	go func() {
		for {
			ev := termbox.PollEvent()

			if ev.Type == termbox.EventKey && ev.Key == termbox.KeyCtrlC {
				// quit.
				close()
				os.Exit(1)
				return
			}

			evCh <- ev
		}
	}()

	for {
		cw, ch := termbox.Size()

		select {
		case <-qCh:
			return nil
		case m := <-pxls.WsMsgCh:
			switch m.Type {
			case "pixel":
				for _, p := range m.Pixels {
					pxls.Board[p.X][p.Y] = p.Color
				}
			}
		case ev := <-evCh:
			switch ev.Type {
			case termbox.EventKey:
				if pxls.IsInit() {
					// handle moving.
					switch {
					case ev.Key == termbox.KeyArrowUp:
						posY -= 5
					case ev.Key == termbox.KeyArrowDown:
						posY += 5
					case ev.Key == termbox.KeyArrowLeft:
						posX -= 5
					case ev.Key == termbox.KeyArrowRight:
						posX += 5
					}
				}
			case termbox.EventMouse:
				if pxls.IsInit() {
					// handle scrolling.
					switch ev.Key {
					case termbox.MouseRelease:
						smX = -1
						smY = -1
					case termbox.MouseLeft:
						if smX != -1 && smY != -1 {
							posX -= ev.MouseX - smX
							posY -= ev.MouseY - smY
						}

						smX = ev.MouseX
						smY = ev.MouseY
					}
				}
			case termbox.EventError:
				return ev.Err
			}
		default:
		}

		posX = clamp(posX, -cw/4, pxls.Board.Width()-2-cw/4)
		posY = clamp(posY, -ch/2, pxls.Board.Height()-2-ch/2)

		if err := render(); err != nil {
			return err
		}
	}
}

func ceil(n float64) int {
	if n > float64(int(n)) {
		return int(n) + 1
	}

	return int(n)
}

func clamp(n, min, max int) int {
	if n < min {
		return min
	} else if n > max {
		return max
	}

	return n
}

func render() error {
	termbox.Clear(fgColor, bgColor)

	if err := renderScreen(); err != nil {
		return err
	}

	if err := termbox.Flush(); err != nil {
		return err
	}

	return nil
}

func renderScreen() error {
	cw, ch := termbox.Size()

	if !pxls.IsInit() {
		const s = "Loading..."

		renderStr(s, (cw-len(s))/2, ch/2, termbox.ColorWhite, termbox.ColorBlack)
		return nil
	}

	for x := 0; x < cw/2; x++ {
		for y := 0; y < ch; y++ {
			var (
				bx = posX + x
				by = posY + y
			)

			var c termbox.Attribute
			if bx > 0 && bx < pxls.Board.Width() && by > 0 && by < pxls.Board.Height() {
				// if in bounds, get color from board.
				c = pxls.Palette[pxls.Board[posX+x][posY+y]]
			} else {
				// if out of bounds, use default background color.
				c = bgColor
			}

			for i := 0; i < 2; i++ {
				termbox.SetCell(x*2+i, y, ' ', fgColor, c)
			}
		}
	}

	boardBuf := termbox.CellBuffer()
	renderStrKeepColors(fmt.Sprintf("(%d;%d)", posX, posY), 0, 0, cw, boardBuf)
	renderStrKeepColors(debug, 0, 1, cw, boardBuf)

	return nil
}

func renderStrWithFunc(str string, x, y int, funk renderStrFunc) {
	cw, _ := termbox.Size()

	runes := []rune(str)

	var xOff, yOff int
	for _, r := range runes {
		xOff++

		if r == '\n' || xOff >= cw {
			xOff = 0
			yOff++
			continue
		}

		funk(r, x+xOff, y+yOff)
	}

	return
}

func renderStrKeepColors(str string, x, y, cellW int, cells []termbox.Cell) {
	renderStrWithFunc(str, x, y, func(r rune, x, y int) {
		i := x + y*cellW

		var (
			fg = fgColor
			bg = bgColor
		)
		if i < len(cells) {
			fg = cells[i].Fg
			bg = cells[i].Bg
		}

		termbox.SetCell(x, y, r, fg, bg)
	})

	return
}

func renderStr(str string, x, y int, fg, bg termbox.Attribute) {
	renderStrWithFunc(str, x, y, func(r rune, x, y int) {
		termbox.SetCell(x, y, r, fg, bg)
	})

	return
}

func min(a, b int) int {
	if a > b {
		return a
	}

	return b
}
