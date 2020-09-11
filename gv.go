package main

import (
	"fmt"

	"github.com/nsf/termbox-go"
)

// Standard line drawing characters
const tlcorner = '\u250c'
const trcorner = '\u2510'
const llcorner = '\u2514'
const lrcorner = '\u2518'
const hline = '\u2500'
const vline = '\u2502'

// Bold line drawing characters
const btlcorner = '\u250f'
const btrcorner = '\u2513'
const bllcorner = '\u2517'
const blrcorner = '\u251b'
const bhline = '\u2501'
const bvline = '\u2503'

type category struct {
	name string
}

type headline struct {
	text       string
	children   []*headline
	document   *string
	categories []*category
	parent     *headline
	visible    bool // Is this Node visible or collapsed?
}

func print(x int, y int, text string) {
	for _, c := range text {
		termbox.SetCell(x, y, c, termbox.ColorWhite, termbox.ColorBlack)
		x++
	}
}

func drawBorder() {
	//get rows/cols from current terminal size & draw the border & status bar
	width, height := termbox.Size()
	// Corners
	termbox.SetCell(0, 0, tlcorner, termbox.ColorWhite, termbox.ColorBlack)
	termbox.SetCell(width-1, 0, trcorner, termbox.ColorWhite, termbox.ColorBlack)
	termbox.SetCell(0, height-1, llcorner, termbox.ColorWhite, termbox.ColorBlack)
	termbox.SetCell(width-1, height-1, lrcorner, termbox.ColorWhite, termbox.ColorBlack)
	// Horizontal
	for x := 1; x < width-1; x++ {
		termbox.SetCell(x, 0, hline, termbox.ColorWhite, termbox.ColorBlack)
		termbox.SetCell(x, height-1, hline, termbox.ColorWhite, termbox.ColorBlack)
	}
	// Vertical
	for y := 1; y < height-1; y++ {
		termbox.SetCell(0, y, vline, termbox.ColorWhite, termbox.ColorBlack)
		termbox.SetCell(width-1, y, vline, termbox.ColorWhite, termbox.ColorBlack)
	}
}

func drawScreen() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	drawBorder()
	termbox.Flush()
}

func drawResize(width int, height int) {
	message := fmt.Sprintf("Hello World %d X %d", width, height)
	print(5, 3, message)
	termbox.Flush()
}

func main() {

	// Draw a bordered window that resizes when terminal resizes
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	termbox.SetInputMode(termbox.InputEsc | termbox.InputMouse)

	drawScreen()
	w, h := termbox.Size()
	drawResize(w, h)

loop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			if ev.Key == termbox.KeyCtrlQ {
				break loop
			}
		case termbox.EventResize:
			drawScreen()
			drawResize(ev.Width, ev.Height)
		case termbox.EventMouse:
			drawScreen()
		case termbox.EventError:
			panic(ev.Err)
		}
	}

}
