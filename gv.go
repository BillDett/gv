package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/nsf/termbox-go"
)

/*

	So we will use Termbox to do direct screen stuff as an editor, and use gocui Views as a way to do pop-ups and other effects/special things.

*/

const delta = 1

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

var root *headline
var renderbuf [][]rune

var toprow int // topmost visible row in renderbuf
var width int
var height int

var gui *gocui.Gui

// Standard line drawing characters
const tlcorner = '\u250c'
const trcorner = '\u2510'
const llcorner = '\u2514'
const lrcorner = '\u2518'
const hline = '\u2500'
const vline = '\u2502'

var lorem = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."

func generateHeadline() string {
	length := rand.Intn(len(lorem))
	return lorem[0:length]
}

func generateOutline(depth int) *headline {
	if depth != 0 {
		t := generateHeadline()
		hl := &headline{t, nil, nil, nil, nil, true}
		rand.Seed(time.Now().UnixNano())
		hlc := rand.Intn(5) + 5 // # children
		hl.children = make([]*headline, hlc)
		for h, _ := range hl.children {
			hl.children[h] = generateOutline(depth - 1)
		}
		return hl
	}
	return nil
}

func renderHeadline(h *headline, indent int) {
	if h != nil {
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
}

func renderOutline() {
	renderbuf = nil
	for _, h := range root.children {
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
					//termbox.SetCell(x, y, col, termbox.ColorWhite, termbox.ColorBlack)
					gui.SetRune(x, y, col, gocui.ColorWhite, gocui.ColorBlack)
				}
				x++
			}
			y++
		}
	}
}

func drawBorder(x, y, width, height int) {
	// Corners
	gui.SetRune(x, y, tlcorner, gocui.ColorWhite, gocui.ColorBlack)
	gui.SetRune(x+width-1, y, trcorner, gocui.ColorWhite, gocui.ColorBlack)
	gui.SetRune(x, y+height-1, llcorner, gocui.ColorWhite, gocui.ColorBlack)
	gui.SetRune(x+width-1, y+height-1, lrcorner, gocui.ColorWhite, gocui.ColorBlack)
	// Horizontal
	for bx := x + 1; bx < x+width-1; bx++ {
		gui.SetRune(bx, y, hline, gocui.ColorWhite, gocui.ColorBlack)
		gui.SetRune(bx, y+height-1, hline, gocui.ColorWhite, gocui.ColorBlack)
	}
	// Vertical
	for by := y + 1; by < y+height-1; by++ {
		gui.SetRune(x, by, vline, gocui.ColorWhite, gocui.ColorBlack)
		gui.SetRune(x+width-1, by, vline, gocui.ColorWhite, gocui.ColorBlack)
	}
}

func drawScreen() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	drawBorder(0, 0, width, height)
	drawOutline()
	termbox.Flush()
}

func layout(g *gocui.Gui) error {
	width, height = g.Size()
	renderOutline()
	drawBorder(0, 0, width, height)
	drawOutline()
	return nil
}

func initKeybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			return gocui.ErrQuit
		}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			return nil
		}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			if toprow != len(renderbuf)-height-2 {
				toprow++
				drawScreen()
			}
			return nil
		}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			if toprow != 0 {
				toprow--
				drawScreen()
			}
			return nil
		}); err != nil {
		return err
	}
	return nil
}

func main() {

	root = generateOutline(5)

	var err error
	gui, err = gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer gui.Close()

	gui.SetManagerFunc(layout)

	if err := initKeybindings(gui); err != nil {
		log.Panicln(err)
	}

	if err := gui.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}
