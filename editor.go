package main

import (
	_ "embed"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gdamore/tcell/v2"
)

//go:embed gulliver.txt
var mediumtext string

type editor struct {
	origX      int         // top left X corner
	origY      int         // top left y corner
	width      int         // How many runes wide is the editor
	height     int         // How many lines high is the editor
	position   int         // Current rune underneath the cursor
	cursX      int         // X coordinate of the cursor
	cursY      int         // Y coordinate of the cursor
	buf        *PieceTable // Text being edited
	lineIndex  []line      // Text Position index for each "line" after editor has been laid out.
	linePtr    int         //index of the line currently beneath the cursor
	topLine    int         // index of the topmost "line" of the window in lineIndex
	bottomLine int         // index of the bottommost "line" of the window in lineIndex
	startPos   int         // Text Position of first character in window
}

// a line is a logic representation of a line that is rendered in the window
type line struct {
	position int // Text position in editor.text()
	length   int // How many runes in this "line"
}

var defStyle tcell.Style

// Standard line drawing characters
const tlcorner = '\u250c'
const trcorner = '\u2510'
const llcorner = '\u2514'
const lrcorner = '\u2518'
const hline = '\u2500'
const vline = '\u2502'

var lorem = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.\nUt enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."

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
func drawEditorText(s tcell.Screen, ed *editor) {
	// Layout all of the text in the editor (re-create the lineIndex)
	x := 1
	y := 1
	runes := ed.text()
	lineStartPos := 0
	lineCount := 0
	ed.lineIndex = []line{}
	ed.lineIndex = append(ed.lineIndex, line{lineStartPos, -1})
	for pos, r := range *runes {
		x++
		if x > ed.width || r == '\n' { // wrap?
			x--
			ed.lineIndex[lineCount].length = x // Record length of the line we just saw
			lineStartPos = pos + 1
			ed.lineIndex = append(ed.lineIndex, line{lineStartPos, -1}) // Get ready for the next line
			x = 1
			y++
			lineCount++
		}
	}
	// handle remaining line if we had one
	if x > 1 {
		x--
		ed.lineIndex[lineCount].length = x // Record length of the line we just saw
	}

	// Draw the rendered lines that should be visible in window (topline until end of window or text )
	y = 1
	for ed.bottomLine = ed.topLine; ed.bottomLine < len(ed.lineIndex) && y < ed.height; ed.bottomLine++ {
		x = 1
		line := ed.lineIndex[ed.bottomLine]
		for p := line.position; p < line.position+line.length; p++ {
			if line.length != -1 {
				s.SetContent(x, y, (*runes)[p], nil, defStyle)
			}
			x++
		}
		y++
	}
}

func drawScreen(s tcell.Screen, ed *editor) {
	width, height := s.Size()
	ed.width = int(float64(width) * 0.7)
	ed.height = height - 2
	s.Clear()
	drawBorder(s, 0, 0, width, height)
	s.ShowCursor(ed.cursX, ed.cursY)
	drawEditorText(s, ed)
	s.Show()
}

func newEditor(s tcell.Screen) *editor {
	if s == nil {
		return nil
	}
	width, height := s.Size()
	ew := int(float64(width) * 0.7)
	return &editor{1, 1, ew, height - 2, 0, 1, 1, NewPieceTable(""), []line{}, 0, 0, 0, 0}
}

func (e *editor) dump() {
	out := fmt.Sprintf("Width %d, Height %d, curX %d, curY %d\nLastpos %d Position %d, Topline %d, Bottomline %d, startPos %d, #lines %d, linePtr %d, lineIndex %v\n",
		e.width, e.height, e.cursX, e.cursY, e.buf.lastpos, e.position, e.topLine, e.bottomLine, e.startPos, len(e.lineIndex), e.linePtr, e.lineIndex)
	ioutil.WriteFile("editordump.txt", []byte(out), 0644)
}

func (e *editor) moveRight() {
	if e.position < e.buf.lastpos {
		// TODO: THIS IS NOT CORRECT
		if e.cursX < e.width && e.cursX < e.lineIndex[e.linePtr].length-1 {
			e.cursX++
		} else { // At end of a line, several things we can do here
			// Are we at bottom right of window?
			if e.cursX == e.width && e.cursY == e.height {
				if e.linePtr < len(e.lineIndex)-1 { // More text to scroll down to
					e.topLine++
					e.cursX = e.origX
					e.linePtr++
				}
			} else { // just at end of a line, move to next visible one
				e.cursX = e.origX
				e.cursY++
				e.linePtr++
			}
		}
		e.position++
	}
}

func (e *editor) moveLeft() {
	if e.position > 0 {
		if e.cursX != e.origX {
			e.cursX--
		} else { // At beginning of a line, several things we can do  here
			// Are we at top left of window?
			if e.cursX == e.origX && e.cursY == e.origY {
				if e.linePtr > 0 { // More text to scroll up to
					e.topLine--
					e.linePtr--
					e.cursX = e.lineIndex[e.linePtr].length
				}
			} else { // just at beginning of line, move to previous visible one
				e.cursY--
				e.linePtr--
				e.cursX = e.lineIndex[e.linePtr].length
			}
		}
		e.position--
	}
}

func (e *editor) moveUp() {
	if e.linePtr > 0 {
		if e.cursY == e.origY { // Scroll
			e.topLine--
		} else if e.cursY > 0 {
			e.cursY--
		}
		e.adjustCursX(-1)
		e.position = e.lineIndex[e.linePtr].position + e.cursX - 1
	}
}

func (e *editor) moveDown() {
	if e.linePtr < len(e.lineIndex)-1 {
		if e.cursY == e.height { // Scroll
			e.topLine++
		} else if e.cursY < e.height {
			e.cursY++
		}
		e.adjustCursX(1)
		e.position = e.lineIndex[e.linePtr].position + e.cursX - 1
	}
}

// Helper function to 'adjust' the x cursor position based upon previous/next line length compared to where we're coming from
func (e *editor) adjustCursX(offset int) {
	currentLength := e.lineIndex[e.linePtr].length
	e.linePtr += offset
	newLength := e.lineIndex[e.linePtr].length
	if newLength < currentLength && e.cursX > newLength {
		e.cursX = newLength - 1
	}
}

func (e *editor) addRune(r rune) {
	e.buf.AppendRune(r)
}

func (e *editor) insertRuneAtCurrentPosition(r rune) {
	e.insertRune(e.position, r)
}

func (e *editor) insertRune(pos int, r rune) {
	e.buf.InsertRunes(pos, []rune{r})
	e.moveRight()
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
		e.moveLeft()
	}
}

func (e *editor) delete() {
	if e.buf.lastpos > 0 {
		e.buf.Delete(e.position, 1)
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
				ed.dump()
				os.Exit(0)
			case tcell.KeyDown:
				ed.moveDown()
				drawScreen(s, ed) // TODO: only if we have scrolled
			case tcell.KeyUp:
				ed.moveUp()
				drawScreen(s, ed) // TODO: only if we have scrolled
			case tcell.KeyRight:
				ed.moveRight()
				drawScreen(s, ed) // TODO: only if we have scrolled
			case tcell.KeyLeft:
				ed.moveLeft()
				drawScreen(s, ed) // TODO: only if we have scrolled
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				ed.backspace()
				drawScreen(s, ed)
			case tcell.KeyDelete:
				ed.delete()
				drawScreen(s, ed)
			case tcell.KeyCtrlF: // for debugging
				ed.dump()
			case tcell.KeyEnter, tcell.KeyTab:
				// TODO: Handle this whitespace
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
	//ed.addText(mediumtext)

	drawScreen(s, ed)

	handleEvents(s, ed)

}
