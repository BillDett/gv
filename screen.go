package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

type outline struct {
	buf             PieceTable // buffer holding the text of the outline
	lineIndex       []line     // Text Position index for each "line" after editor has been laid out.
	linePtr         int        // index of the line currently beneath the cursor
	editorWidth     int        // width of an editor column
	editorHeight    int        // height of the editor window
	currentPosition int        // the current position within the buf
	topLine         int        // index of the topmost "line" of the window in lineIndex
}

// a line is a logical representation of a line that is rendered in the window
type line struct {
	bullet        rune // What bullet should precede this line (if any)?
	indent        int  // Initial indent before a bullet
	hangingIndent int  // Indent for text without a bullet
	position      int  // Text position in editor.text()
	length        int  // How many runes in this "line"
}

// Render buffer- where we write the outline characters to during layout phase
var renderBuf [][]rune

/*
 Outline structure takes the following form:
 	<nodeDelim><level><nodeDelim><headline><nodeDelim><level><nodeDelim><headline><nodeDelim><EOF>

 where headline order denotes tree structure.  When <level> increases from previous <headline> this
 indicates a new level of the outline.  When <level> decreases it indicates going 'up' the outline
 a level.

 The end of the outline is denoted by <nodeDelim><EOL>- this allows us to always append text to
 a headline by placing the o.currentPosition on the <nodeDelim> right after the headline for insertion.

*/
const nodeDelim = '\ufeff'
const eof = '\u0000'

var defStyle tcell.Style

var currentLine int // which "line" we are currently on
var cursX int       // X coordinate of the cursor
var cursY int       // Y coordinate of the cursor

var dbg int
var dbg2 int

var currentFilename string

// Standard line drawing characters
const tlcorner = '\u250c'
const trcorner = '\u2510'
const llcorner = '\u2514'
const lrcorner = '\u2518'
const hline = '\u2500'
const vline = '\u2502'

const lf = '\u000A'

const htriangle = '\u25B6'
const small_htriangle = '\u25B8'
const vtriangle = '\u25BC'
const small_vtriangle = '\u25BE'
const solid_bullet = '\u25CF'
const open_bullet = '\u25CB'
const tri_bullet = '\u2023'
const dash_bullet = '\u2043'
const shear_bullet = '\u25B0'
const box_bullet = '\u25A0'

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

func indentCode(level int) string {
	return fmt.Sprintf("%c%d%c", nodeDelim, level, nodeDelim)
}

func newOutline(s tcell.Screen) *outline {
	o := &outline{*NewPieceTable(""), nil, 0, 0, 0, 3, 1}
	o.setScreenSize(s)
	return o
}

// initialize a new outline to be used as a blank outline for editing
func (o *outline) init() {
	eofDelim := []rune{nodeDelim, eof}
	o.buf.InsertRunes(0, eofDelim)
	o.addHeadline("", 0)
}

func (o *outline) dump() {
	text := o.buf.Runes()
	out := fmt.Sprintf("lastPos %d\tcurrentPosition %d (%#U)\tlinePtr %d\nlineIndex %v\nlineIndex.position %d\tlineIndex.length %v\tdbg %d\tdbg2 %d\nFirst rune (%#U)\ttopLine %d\teditorheight %d\n",
		o.buf.lastpos, o.currentPosition, (*text)[o.currentPosition], o.linePtr, o.lineIndex,
		o.lineIndex[o.linePtr].position, o.lineIndex[o.linePtr].length, dbg, dbg2, (*text)[0],
		o.topLine, o.editorHeight)
	ioutil.WriteFile("dump.txt", []byte(out), 0644)
}

// save the outline buffer to a file
func (o *outline) save(filename string) {
	ioutil.WriteFile(filename, []byte(o.buf.Text()), 0644)
}

// load a .gv file and use it to populate the outline's buffer
func (o *outline) load(filename string) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Unable to read file %s\n", filename)
		os.Exit(1)
	}
	o.buf.InsertRunes(0, []rune(string(buf)))
}

func (o *outline) setScreenSize(s tcell.Screen) {
	var width int
	width, height := s.Size()
	o.editorWidth = int(float64(width) * 0.7)
	o.editorHeight = height - 2
}

// appends a new headline onto the outline (before the final <EOFDelim>)
func (o *outline) addHeadline(text string, level int) {
	o.buf.Insert(o.buf.lastpos-2, indentCode(level)+text)
}

// Store a 'logical' line- this is a rendered line of text on the screen. We use this index
// to figure out where in the outline buffer to move to when we navigate visually
// Optionally remember if this is the current line we're sitting on with the cursor
func (o *outline) recordLogicalLine(bullet rune, indent int, hangingIndent int, position int, length int, current bool) {
	o.lineIndex = append(o.lineIndex, line{bullet, indent, hangingIndent, position, length})
	if current {
		o.linePtr = len(o.lineIndex) - 1
	}
}

// Figure out what is level of headline under o.currentPosition
// Walk backwards from current positon until you find the leading <nodeDelim> and then extract the level
func (o *outline) currentLevel() int {
	text := (*o.buf.Runes())
	var begin, rhs, start int
	if text[o.currentPosition] == nodeDelim { // "back up" if we're sitting on a trailing <nodeDelim> at end of a line
		begin = o.currentPosition - 1
	} else {
		begin = o.currentPosition
	}
	// find right hand nodeDelim
	for rhs = begin; text[rhs] != nodeDelim; rhs-- {
	}
	// extract the level
	for start = rhs - 1; text[start] != nodeDelim; start-- {
	}
	level, _ := strconv.Atoi(string(text[start+1 : rhs]))
	return level
}

// Tokenize the text to extract the next Headline, returning level, start position and end position of headline
func (o *outline) nextHeadline(position int) (int, int, int, error) {
	// find the first delimiter
	text := *(o.buf.Runes())
	var p, q int
	for p = position; p < len(text); p++ {
		if text[p] == nodeDelim {
			break
		}
	}
	if p == len(text) { // Didn't find any delmiter
		return 0, 0, 0, errors.New("Didn't find any delmiter in text")
	} else {
		// text[p] is first delimiter, find the matching one
		for q = p + 1; q < len(text); q++ {
			if text[q] == nodeDelim {
				break
			}
		}
		level, err := strconv.Atoi(string(text[p+1 : q]))
		if err != nil {
			// badly formed level
			return 0, 0, 0, fmt.Errorf("Badly formed delimiter %s seen", string(text[p+1:q]))
		}
		// Find extent of this headline
		for p = q + 1; p < len(text); p++ {
			if text[p] == nodeDelim {
				break
			}
		}
		return level, q + 1, p, nil
	}
}

func layoutOutline(s tcell.Screen, o *outline, width int, height int) {
	y := 1
	o.linePtr = 0
	o.lineIndex = []line{}
	o.lineIndex = append(o.lineIndex, line{0, 0, 0, 0, -1})
	text := o.buf.Runes()
	var level, start, end int
	var err error
	for end < len(*text)-2 { // Scan thru all headlines (stop before you hit EOL Delim)
		level, start, end, err = o.nextHeadline(end)
		if err != nil {
			fmt.Printf("%v\n", err)
			break
		}
		y = layoutHeadline(s, o, text, start, end+1, y, level, false) // we use end+1 so we render the <nodeDelim>- this gives us something at end of headline to start typing on when appending text to headline
	}
}

func handleScroll(scroll int, o *outline) {
	if scroll == 1 {
		o.topLine++
	} else if scroll == -1 {
		o.topLine--
	}
}

// Format headline according to indent and word-wrap the text.
func layoutHeadline(s tcell.Screen, o *outline, text *[]rune, start int, end int, y int, level int, collapsed bool) int {
	o.setScreenSize(s)
	origX := 1
	level++
	var bullet rune
	endY := y
	if collapsed {
		bullet = solid_bullet
	} else {
		bullet = vtriangle
	}
	indent := origX + (level * 3)
	hangingIndent := indent + 3
	headlineLength := end - start + 1
	if headlineLength <= o.editorWidth-hangingIndent { // headline fits entirely within a single line
		cursorOnThisLine, scroll := layoutLine(s, o, text, start, end, hangingIndent, endY)
		o.recordLogicalLine(bullet, indent, hangingIndent, start, headlineLength-1, cursorOnThisLine)
		handleScroll(scroll, o)
		endY++
	} else { // going to have to wrap it
		pos := start
		firstLine := true
		for pos < end {
			endPos := pos + o.editorWidth
			if endPos > end { // overshot end of text, we're on the last fragment
				cursorOnThisLine, scroll := layoutLine(s, o, text, pos, end, hangingIndent, endY)
				handleScroll(scroll, o)
				o.recordLogicalLine(0, indent, hangingIndent, pos, end-pos, cursorOnThisLine)
				endPos = end
				endY++
			} else { // on first or middle fragment
				var mybullet rune
				if firstLine { // if we're laying out first line of a multi-line headline, remember that we want to use a bullet
					mybullet = bullet
					firstLine = false
				}
				if !unicode.IsSpace((*text)[endPos]) {
					// Walk backwards until you see your first whitespace
					p := endPos
					for p > pos && !unicode.IsSpace((*text)[p]) {
						p--
					}
					if p != pos { // split at the space (hitting pos means beginning of text or last starting point)
						endPos = p + 1
					}
				}
				cursorOnThisLine, scroll := layoutLine(s, o, text, pos, endPos, hangingIndent, endY)
				handleScroll(scroll, o)
				o.recordLogicalLine(mybullet, indent, hangingIndent, pos, endPos-pos, cursorOnThisLine)
				endY++
			}
			pos = endPos
		}
	}

	return endY
}

// Layout each individual character of this logical line.
// Watch for the current position and update cursor to match it.
// Return whether or not we updated the cursor on this line, and whether we should scroll or not {0=no, -1=up, 1=down}
func layoutLine(s tcell.Screen, o *outline, text *[]rune, start int, end int, indent int, y int) (bool, int) {
	updatedCursor := false
	//lastLine := o.topLine + o.editorHeight
	scroll := 0
	x := 0
	for c := start; c < end; c++ {
		if o.currentPosition == c { // If we're rendering the current position, place cursor here
			cursX = indent + x
			// Decide whether we've scrolled up or down based upon previous cursY value
			if y > cursY && cursY == o.editorHeight {
				scroll = 1
			} else if y < cursY && cursY == 1 {
				scroll = -1
			} else {
				cursY = y
			}
			updatedCursor = true
		}
		x++
	}
	return updatedCursor, scroll
}

// Walk thru the lineIndex and render each logical line that is within the window's boundaries
func renderOutline(s tcell.Screen, o *outline) {
	runes := *(o.buf.Runes())
	y := 1
	lastLine := o.topLine + o.editorHeight - 1
	for l := o.topLine; l <= lastLine && l < len(o.lineIndex); l++ {
		x := 0
		line := o.lineIndex[l]
		if line.length != -1 {
			s.SetContent(x+line.indent, y, line.bullet, nil, defStyle)
			for p := line.position; p < line.position+line.length; p++ {
				s.SetContent(x+line.hangingIndent, y, runes[p], nil, defStyle)
				x++
			}
			y++
		}
	}
}

func genTestOutline(s tcell.Screen) *outline {
	o := newOutline(s)
	o.addHeadline("What is this odd beast GrandView?", 0)
	o.addHeadline("In a single-pane outliner, all the components of your outline and its accompanying information are visible in one window.", 1)
	o.addHeadline("Project and task manager", 2)
	o.addHeadline("Information manager", 2)
	o.addHeadline("Here's a headline that has children hidden", 0)
	o.addHeadline("Here's a headline that has no children", 0)
	o.addHeadline("What makes GrandView so unique even today?  How is it possible that such a product like this could exist?", 0)
	o.addHeadline("Multiple Views", 1)
	o.addHeadline("ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890", 1)
	o.addHeadline("Outline View", 1)
	o.addHeadline("You can associate any headline (node) with a document. Document view is essentially a hoist that removes all the other elements of your outline from the screen so you can focus on writing the one document. When you are done writing this document (or section of your outline), you can return to outline view, where your document text.", 1)
	o.addHeadline("Category & Calendar Views", 1)
	o.addHeadline("Way over the top.", 2)
	o.addHeadline("ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVW XYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ123456 7890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890", 2)
	o.addHeadline("Fully customizable meta-data", 1)
	return o
}

func drawScreen(s tcell.Screen, o *outline) {
	width, height := s.Size()
	s.Clear()
	drawBorder(s, 0, 0, width, height)
	layoutOutline(s, o, width, height)
	renderOutline(s, o)
	s.ShowCursor(cursX, cursY)
	s.Show()
}

// TODO: We can probably simplify this a lot w/out using o.buf.Runes and o.nextHeadline
func (o *outline) moveRight() {
	if o.currentPosition < o.buf.lastpos-1 {
		text := *(o.buf.Runes())
		if text[o.currentPosition] == nodeDelim {
			peek := o.currentPosition + 1
			if text[peek] != eof {
				// at last character of current headline, move to beginning of next headline
				_, o.currentPosition, _, _ = o.nextHeadline(o.currentPosition)
				return
			}
		} else {
			// Just move to next character in this headline
			o.currentPosition++
		}
	}
}

// TODO: We can probably simplify this a bit removing o.buf.Runes and loop skipping over delims
func (o *outline) moveLeft() {
	if o.currentPosition > 3 { // Do nothing if on first character of first headline
		text := *(o.buf.Runes())
		if text[o.currentPosition-1] == nodeDelim {
			// at first character of current headline, move to end of previous headline
			// We want to land on the <nodeDelim> just prior to this headline so we are on an append point for new text
			o.currentPosition -= 2
			for text[o.currentPosition] != nodeDelim {
				o.currentPosition--
			}
		} else { // Just move to previous character in this headline
			o.currentPosition--
		}
	}
}

func (o *outline) moveDown() {
	if o.currentPosition < o.buf.lastpos-1 {
		offset := o.currentPosition - o.lineIndex[o.linePtr].position // how far 'in' are we on the logical line?
		dbg = offset
		newLinePtr := o.linePtr + 1
		if newLinePtr < len(o.lineIndex) { // There are more lines below us
			dbg2 = o.lineIndex[newLinePtr].length
			if offset >= o.lineIndex[newLinePtr].length { // Are we moving down to a smaller line with x too far right?
				o.currentPosition = o.lineIndex[newLinePtr].position + o.lineIndex[newLinePtr].length - 1
			} else {
				o.currentPosition = offset + o.lineIndex[newLinePtr].position
			}
		}
	}
}

func (o *outline) moveUp() {
	if o.currentPosition > 3 { // Do nothing if on first character of first headline
		offset := o.currentPosition - o.lineIndex[o.linePtr].position // how far 'in' are we on the logical line?
		newLinePtr := o.linePtr - 1
		if newLinePtr > 0 { // There are more lines above
			dbg2 = o.lineIndex[newLinePtr].length
			if offset >= o.lineIndex[newLinePtr].length { // Are we moving up to a smaller line with x too far right?
				o.currentPosition = o.lineIndex[newLinePtr].position + o.lineIndex[newLinePtr].length - 1
			} else {
				o.currentPosition = offset + o.lineIndex[newLinePtr].position
			}
		}
	}
}

func (o *outline) insertRuneAtCurrentPosition(r rune) {
	o.buf.InsertRunes(o.currentPosition, []rune{r})
	o.moveRight()
}

func (o *outline) backspace() {
	if o.currentPosition > 3 {
		text := *(o.buf.Runes())
		posToRemove := o.currentPosition - 1
		if text[posToRemove] == nodeDelim { // Are we trying to join with previous headline?
			if posToRemove != 2 { // Make sure we're not on first headline
				var start int
				for start = posToRemove - 1; text[start] != nodeDelim; start-- { // find start/end of the nodeDelims
				}
				o.buf.Delete(start, posToRemove-start+1) // Remove the <nodeDelim><Level><nodeDelim> fragment
			}
		} else {
			o.moveLeft()
			o.buf.Delete(o.currentPosition, 1)
		}
	}
}

func (o *outline) delete() {
	if o.buf.lastpos > 2 {
		text := *(o.buf.Runes())
		if text[o.currentPosition] == nodeDelim { // Are we trying to join the next headline?
			if o.currentPosition != o.buf.lastpos-2 { // Make sure we're not on <nodeDelim> after last headline
				var start int
				for start = o.currentPosition + 1; text[start] != nodeDelim; start++ { // find start/end of the nodeDelims
				}
				o.buf.Delete(o.currentPosition, start-o.currentPosition+1) // Remove the <nodeDelim><Level><nodeDelim> fragment
			}
		} else { // Just delete the current position
			o.buf.Delete(o.currentPosition, 1)
		}
	}
}

// Enter always creates a new headline at current position at same level as current headline
func (o *outline) enterPressed() {
	delim := indentCode(o.currentLevel())
	o.buf.Insert(o.currentPosition, delim)
	o.moveRight()
}

// Tab indents a headline and all subsequent headlines with levels > this level, if not on the first headline
// TODO: Think we should factor out some more low-level utilities for finding/changing headlines within the
//   stream to make this easier.
func (o *outline) tabPressed() {

}

func handleEvents(s tcell.Screen, o *outline) {
	for {
		switch ev := s.PollEvent().(type) {
		case *tcell.EventResize:
			s.Sync()
			drawScreen(s, o)
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape:
				s.Fini()
				os.Exit(0)
			case tcell.KeyDown:
				o.moveDown()
				drawScreen(s, o)
			case tcell.KeyUp:
				o.moveUp()
				drawScreen(s, o)
			case tcell.KeyRight:
				o.moveRight()
				drawScreen(s, o)
			case tcell.KeyLeft:
				o.moveLeft()
				drawScreen(s, o)
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				o.backspace()
				drawScreen(s, o)
			case tcell.KeyDelete:
				o.delete()
				drawScreen(s, o)
			case tcell.KeyEnter:
				o.enterPressed()
				drawScreen(s, o)
			case tcell.KeyTab:
				o.tabPressed()
				drawScreen(s, o)
			case tcell.KeyRune:
				o.insertRuneAtCurrentPosition(ev.Rune())
				drawScreen(s, o)
			case tcell.KeyCtrlF: // for debugging
				o.dump()
			case tcell.KeyCtrlS:
				if currentFilename == "" {
					// TODO: Should prompt for a filename
					currentFilename = "outline.gv"
				}
				o.save(currentFilename)
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
	//o := genTestOutline(s)
	o := newOutline(s)
	if len(os.Args) > 1 {
		currentFilename = os.Args[1]
		o.load(currentFilename)
	} else {
		o.init()
	}

	drawScreen(s, o)

	handleEvents(s, o)

}
