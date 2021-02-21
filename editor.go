package main

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
)

type editor struct {
	x        int
	y        int
	width    int // How many runes wide is the editor
	position int // Current rune underneath the cursor
	cursX    int // X coordinate of the cursor
	cursY    int // Y coordinate of the cursor
	buf      *PieceTable
}

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

var lorem = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."

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

// TODO: This is going to need to get much smarter- we need to draw whitespace correctly (TAB, NL, CR) and also we
//  should only render the text from the editor that fits on the screen's viewport.  Also, it should intelligently
//  word wrap for us based on whitespace.
func layoutEditorText(s tcell.Screen, runes *[]rune, width int) {
	x := 1
	y := 1
	for _, r := range *runes {
		s.SetContent(x, y, r, nil, defStyle)
		x++
		if x > width {
			x = 1
			y++
		}
	}
}

func drawScreen(s tcell.Screen, ed *editor) {
	width, height = s.Size()
	editorWidth = int(float64(width) * 0.7)
	s.Clear()
	drawBorder(s, 0, 0, width, height)
	s.ShowCursor(ed.cursX, ed.cursY)
	layoutEditorText(s, ed.text(), editorWidth)
	s.Show()
}

func newEditor(s tcell.Screen) *editor {
	if s == nil {
		return nil
	}
	width, height = s.Size()
	ew := int(float64(width) * 0.7)
	return &editor{5, 5, ew, 0, 1, 1, NewPieceTable("")}
}

func (e *editor) moveRight() {
	// TODO: Make sure we're still inside editor text
	if e.cursX != e.width {
		e.cursX++
	} else {
		e.cursX = 1
		e.cursY++
	}
	e.position++
}

func (e *editor) moveLeft() {
	// TODO: Make sure we're still inside editor text, handle beginning of line, etc
	if e.cursX != 0 {
		e.cursX--
		e.position--
	}
}

func (e *editor) moveUp() {

}

func (e *editor) moveDown() {

}

func (e *editor) addRune(r rune) {
	e.buf.AppendRune(r)
}

func (e *editor) insertRuneAtCurrentPosition(r rune) {
	e.insertRune(e.position, r)
}

func (e *editor) insertRune(pos int, r rune) {
	e.buf.InsertRunes(pos, []rune{r})
	// TODO: Handle cursor wrapping at end of line, etc.
	e.cursX++
	e.position++
}

func (e *editor) addText(s string) {
	e.buf.Append(s)
}

func (e *editor) text() *[]rune {
	return e.buf.Runes()
}

func (e *editor) backspace() {
	if e.position > 0 {
		e.buf.Delete(e.position-1, 1)
		e.position--
		// TODO: handle cursor wrapping, etc
		e.cursX--
	}
}

func handleEvents(s tcell.Screen, ed *editor) {
	for {
		switch ev := s.PollEvent().(type) {
		case *tcell.EventResize:
			s.Sync()
			drawScreen(s, ed)
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape:
				s.Fini()
				os.Exit(0)
			case tcell.KeyDown:
				ed.moveDown()
				drawScreen(s, ed)
			case tcell.KeyUp:
				ed.moveUp()
				drawScreen(s, ed)
			case tcell.KeyRight:
				ed.moveRight()
				drawScreen(s, ed)
			case tcell.KeyLeft:
				ed.moveLeft()
				drawScreen(s, ed)
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				ed.backspace()
				drawScreen(s, ed)
			case tcell.KeyRune:
				ed.insertRuneAtCurrentPosition(ev.Rune())
				drawScreen(s, ed)
			}
		}
	}
}

func main() {

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

	ed := newEditor(s)
	ed.addText(lorem)

	drawScreen(s, ed)

	handleEvents(s, ed)

}
