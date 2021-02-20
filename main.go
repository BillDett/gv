package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
)

const delta = 1

type category struct {
	name string
}

type headline struct {
	text       []rune
	children   []*headline
	categories []*category
	parent     *headline
	visible    bool // Is this Node visible or collapsed?
}

type editor struct {
	x     int
	y     int
	width int
	buf   [][]rune
}

var root *headline
var renderbuf [][]rune

var editorWidth int // How wide is editor as % of terminal width

var toprow int // topmost visible row in renderbuf
var width int
var height int

var defStyle tcell.Style

// Standard line drawing characters
const tlcorner = '\u250c'
const trcorner = '\u2510'
const llcorner = '\u2514'
const lrcorner = '\u2518'
const hline = '\u2500'
const vline = '\u2502'

//var lorem = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."
var lorem = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor."

func generateHeadline() string {
	length := rand.Intn(len(lorem))
	return lorem[0:length]
}

func generateOutline(depth int) *headline {
	if depth != 0 {
		t := generateHeadline()
		hl := &headline{[]rune(t), nil, nil, nil, true}
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

func drawOutline(s tcell.Screen) {
	// Draw the renderBuf to the screen
	y := 1
	for row := toprow; row < len(renderbuf); row++ {
		if y < height-2 {
			x := 1
			// TODO: change to index based when doing horizontal scrolling
			for _, col := range renderbuf[row] {
				if x < width-2 {
					s.SetContent(x, y, col, nil, defStyle)
				}
				x++
			}
			y++
		}
	}
}

func drawBorder(s tcell.Screen, x, y, width, height int) {
	// Corners
	s.SetContent(x, y, tlcorner, nil, defStyle)
	s.SetContent(x+width-1, y, trcorner, nil, defStyle)
	s.SetContent(x, y+height-1, llcorner, nil, defStyle)
	s.SetContent(x+width-1, y+height-1, lrcorner, nil, defStyle)
	// Horizontal
	for bx := x + 1; bx < x+width-1; bx++ {
		s.SetContent(bx, y, hline, nil, defStyle)
		s.SetContent(bx, y+height-1, hline, nil, defStyle)
	}
	// Vertical
	for by := y + 1; by < y+height-1; by++ {
		s.SetContent(x, by, vline, nil, defStyle)
		s.SetContent(x+width-1, by, vline, nil, defStyle)
	}
}

func drawScreen(s tcell.Screen) {
	width, height = s.Size()
	//editorWidth = int(width * 0.7)
	s.Clear()
	drawBorder(s, 0, 0, width, height)
	renderOutline()
	drawOutline(s)
	//e.Draw()
	s.Show()
}

/*
func NewEditor(s tcell.Screen) *editor {
	if s == nil {
		return nil
	}
	baseLines := 3 // Default # of lines for each editor
	width, height = s.Size()
	ew := int(width * 0.7)
	buf := make([][]rune, baseLines)
	for i := range buf {
		buf[i] = make([]rune, ew)
	}
	return &editor{5, 5, ew, buf}
}
*/

func handleEvents(s tcell.Screen) {
	for {
		switch ev := s.PollEvent().(type) {
		case *tcell.EventResize:
			s.Sync()
			drawScreen(s)
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape:
				s.Fini()
				os.Exit(0)
			case tcell.KeyDown:
				if toprow != len(renderbuf)-height-2 {
					toprow++
					drawScreen(s)
				}
			case tcell.KeyUp:
				if toprow != 0 {
					toprow--
					drawScreen(s)
				}
			}
		}
	}
}

func main() {

	root = generateOutline(5)

	s, e := tcell.NewScreen()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	if e := s.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}

	defStyle = tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorGreen)
	s.SetStyle(defStyle)

	//e := NewEditor(s)

	drawScreen(s)

	handleEvents(s)

}
