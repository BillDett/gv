package main

import (
	"fmt"
	"math/rand"
	"time"

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

var outline []*headline
var renderbuf [][]rune

var toprow int // topmost visible row in renderbuf
var width int
var height int

func print(x int, y int, text string) {
	for _, c := range text {
		termbox.SetCell(x, y, c, termbox.ColorWhite, termbox.ColorBlack)
		x++
	}
}

func generateOutline(depth int) {
	rand.Seed(time.Now().UnixNano())
	hlc := rand.Intn(50) + 10 // # headlines
	for h := 0; h < hlc; h++ {
		title := fmt.Sprintf("Headline %d", h)
		outline = append(outline, &headline{title, nil, nil, nil, nil, true})
		cc := rand.Intn(5) + 3 // # children
		for c := 0; c < cc; c++ {
			title := fmt.Sprintf("Child %d", c)
			outline[h].children = append(outline[h].children, &headline{title, nil, nil, nil, outline[h], true})
		}
	}
}

func renderHeadline(h *headline, indent int) {
	var hlrow []rune
	//indent
	for i := 0; i < indent; i++ {
		hlrow = append(hlrow, '-')
	}
	//title
	for _, r := range h.text {
		hlrow = append(hlrow, r)
	}
	renderbuf = append(renderbuf, hlrow)
	for _, c := range h.children {
		renderHeadline(c, indent+1)
	}
}

func renderOutline() {
	renderbuf = nil
	for _, h := range outline {
		renderHeadline(h, 0)
	}
}

func drawOutline() {
	// Draw the renderBuf to the screen
	y := 1
	for row := toprow; row < len(renderbuf); row++ {
		if y < height-2 {
			x := 1
			// TODO: change to index based when doing horizontal scrolling
			for _, col := range renderbuf[row] {
				if x < width-2 {
					termbox.SetCell(x, y, col, termbox.ColorWhite, termbox.ColorBlack)
				}
				x++
			}
			y++
		}
	}
}

func drawBorder(x, y, width, height int) {
	// Corners
	termbox.SetCell(x, y, tlcorner, termbox.ColorWhite, termbox.ColorBlack)
	termbox.SetCell(x+width-1, y, trcorner, termbox.ColorWhite, termbox.ColorBlack)
	termbox.SetCell(x, y+height-1, llcorner, termbox.ColorWhite, termbox.ColorBlack)
	termbox.SetCell(x+width-1, y+height-1, lrcorner, termbox.ColorWhite, termbox.ColorBlack)
	// Horizontal
	for bx := x + 1; bx < x+width-1; bx++ {
		termbox.SetCell(bx, y, hline, termbox.ColorWhite, termbox.ColorBlack)
		termbox.SetCell(bx, y+height-1, hline, termbox.ColorWhite, termbox.ColorBlack)
	}
	// Vertical
	for by := y + 1; by < y+height-1; by++ {
		termbox.SetCell(x, by, vline, termbox.ColorWhite, termbox.ColorBlack)
		termbox.SetCell(x+width-1, by, vline, termbox.ColorWhite, termbox.ColorBlack)
	}
}

func drawScreen() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	drawBorder(0, 0, width, height)
	drawOutline()
	termbox.Flush()
}

func drawResize(width int, height int) {
	message := fmt.Sprintf("Hello World %d X %d", width, height)
	print(10, 15, message)
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
	width, height = termbox.Size()

	generateOutline(3)
	renderOutline()

	drawScreen()
	drawResize(width, height)

loop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			if ev.Key == termbox.KeyCtrlQ {
				break loop
			}
			if ev.Key == termbox.KeyArrowDown {
				if toprow != len(renderbuf)-height-2 {
					toprow++
					drawScreen()
				}
			}
			if ev.Key == termbox.KeyArrowUp {
				if toprow != 0 {
					toprow--
					drawScreen()
				}
			}
		case termbox.EventResize:
			drawScreen()
			width = ev.Width
			height = ev.Height
			drawResize(ev.Width, ev.Height)
		case termbox.EventMouse:
			drawScreen()
		case termbox.EventError:
			panic(ev.Err)
		}
	}

}
