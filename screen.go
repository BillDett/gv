package main

import (
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

// a delimiter is a multi-rune token that separates each headline, indicating its level in the hierarchy
// these are created by scanning the outline.buf runes
type delimiter struct {
	lhs   int // position of nodeDelim on left
	level int // level of headline
	rhs   int // position of nodeDelim on right
}

// for tokenization
const (
	forward = iota
	backward
)

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

func delimiterString(level int) string {
	return fmt.Sprintf("%c%d%c", nodeDelim, level, nodeDelim)
}

func newOutline(s tcell.Screen) *outline {
	o := &outline{*NewPieceTable(""), nil, 0, 0, 0, 3, 0}
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
	d, s, e := o.currentHeadline(o.currentPosition)
	out := fmt.Sprintf("lastPos %d\tcurrentPosition %d (%#U)\tlinePtr %d\nlineIndex %v\nlineIndex.position %d\tlineIndex.length %v\tdbg %d\tdbg2 %d\n# of lines: %d\ttopLine %d\teditorheight %d\ncurrent delim %v (%d:%d)\n",
		o.buf.lastpos, o.currentPosition, (*text)[o.currentPosition], o.linePtr, o.lineIndex,
		o.lineIndex[o.linePtr].position, o.lineIndex[o.linePtr].length, dbg, dbg2, len(o.lineIndex),
		o.topLine, o.editorHeight, d, s, e)
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
	o.buf.Insert(o.buf.lastpos-2, delimiterString(level)+text)
}

// Store a 'logical' line- this is a rendered line of text on the screen. We use this index
// to figure out where in the outline buffer to move to when we navigate visually
func (o *outline) recordLogicalLine(bullet rune, indent int, hangingIndent int, position int, length int) {
	o.lineIndex = append(o.lineIndex, line{bullet, indent, hangingIndent, position, length})
}

// Find position of next nodeDelim (based on direction) from position
//  Return 0 or len(text) if nodeDelim found (which would be an error)
func (o *outline) delim(position int, direction int) int {
	text := (*o.buf.Runes())
	var start int
	if direction == forward {
		for start = position; text[start] != nodeDelim && start < len(text); start++ {
		}
	} else {
		for start = position; text[start] != nodeDelim && start > 0; start-- {
		}
	}
	return start
}

// Find the start and end position and numerical value of an integer (based on direction) from position
//   Position must be sitting on a digit character
func (o *outline) integer(position int, direction int) (int, int, int) {

	text := (*o.buf.Runes())
	var start, end, level int
	if !unicode.IsDigit(text[position]) { // This is an error- we must be on a digit
		fmt.Printf("Saw (%#U)\n", text[position])
		return -1, -1, -1
	}
	if direction == forward {
		start = position
		for end = start; unicode.IsDigit(text[end]) && end < len(text); end++ {
		}
	} else {
		end = position
		for start = end; unicode.IsDigit(text[start]) && start > 0; start-- {
		}
	}
	is := string(text[start+1 : end+1])
	//fmt.Printf("Converting >%s< to integer\n", is)
	level, _ = strconv.Atoi(is)
	return start, end, level
}

func (o *outline) test() {
	fmt.Printf("Previous delim %d, Next delim %d\n", o.delim(3, backward), o.delim(3, forward))
	d, s, e := o.currentHeadline(o.currentPosition)
	fmt.Printf("Current Headline delim %d/%d/%d, start %d, end %d\n", d.lhs, d.level, d.rhs, s, e)
	d, s, e = o.nextHeadline(o.currentPosition)
	var last int
	c := 0
	var d2 *delimiter
	for d != nil {
		if c == 2 {
			d2 = d
		}
		fmt.Printf("Next Headline delim %d/%d/%d, start %d, end %d\n", d.lhs, d.level, d.rhs, s, e)
		d, s, e = o.nextHeadline(s)
		if e > last {
			last = s
		}
		c++
	}
	d, s, e = o.previousHeadline(last)
	if d != nil {
		fmt.Printf("Second to last Headline delim %d/%d/%d, start %d, end %d\n", d.lhs, d.level, d.rhs, s, e)
	}
	fmt.Printf("d2 delim %d/%d/%d\n", d2.lhs, d2.level, d2.rhs)
	o.setLevel(d2, 5)
	d2, s2, e2 := o.currentHeadline(d2.rhs + 1)
	fmt.Printf("now d2 delim %d/%d/%d, start %d, end %d\n", d2.lhs, d2.level, d2.rhs, s2, e2)
}

// Get positional information for headline under o.currentPosition
// Walk backwards from current positon until you find the leading <nodeDelim> and then extract the level
// TODO: REMOVE??
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

// Get the delimiter and start/end of the current headline based on current position
func (o *outline) currentHeadline(position int) (*delimiter, int, int) {
	text := *(o.buf.Runes())
	d := delimiter{0, 0, 0}
	var extent int
	if text[position] == nodeDelim { // "back up" if we're sitting on a trailing <nodeDelim> at end of a line
		position--
	}
	d.rhs = o.delim(position, backward)
	s, _, l := o.integer(d.rhs-1, backward)
	if s != -1 {
		d.level = l
		d.lhs = o.delim(s, backward)
		extent = o.delim(d.rhs+1, forward) // find end of headline text
		return &d, d.rhs + 1, extent
	} else {
		return nil, 0, 0
	}
}

// Tokenize the text to extract the next Headline, returning delimiter, start position and end position of headline
//  Return a nil delimiter if we're on last Headline
func (o *outline) nextHeadline(position int) (*delimiter, int, int) {
	text := *(o.buf.Runes())
	var p, q int
	// scan to end of this headline, find the first nodeDelim
	p = o.delim(position, forward)
	// make sure we're not at end of the outline
	if text[p+1] != eof {
		// scan forward find the matching nodeDelim
		q = o.delim(p+1, forward)
		q++ // skip over nodeDelim
		return o.currentHeadline(q)
	} else {
		return nil, 0, 0
	}
}

// Tokenize the text to extract the previous Headline, returning delimiter, start position and end position of headline
//  Return a nil delimiter if we're on first Headline
func (o *outline) previousHeadline(position int) (*delimiter, int, int) {
	d, _, _ := o.currentHeadline(position)
	if d.lhs != 0 { // Are we not on first headline?
		return o.currentHeadline(d.lhs - 1)
	}

	return nil, 0, 0
}

// Set the level of current outline to newLevel and adjust all subsequent "child" headlines.
// if promote == true, increase each child's level by 1
// if promote == false, decrease each child's level by 1
// We determine when we are finished finding children when there are no more headlines or next headline's level == this headline
func (o *outline) changeRank(promote bool) {
	difference := 1
	if !promote {
		difference = -1
	}
	d, _, _ := o.currentHeadline(o.currentPosition)
	origLevel := d.level
	o.setLevel(d, d.level+difference)
	// add the difference to all children
	var s int
	d, s, _ = o.nextHeadline(o.currentPosition)
	for d != nil && d.level != origLevel {
		o.setLevel(d, d.level+difference)
		d, s, _ = o.nextHeadline(s)
	}
}

// Modify the level in the buffer for this delimiter (replace characters for the integer)
func (o *outline) setLevel(d *delimiter, newLevel int) {
	levelLen := d.rhs - d.lhs - 1
	o.buf.Delete(d.lhs+1, levelLen)
	o.buf.Insert(d.lhs+1, strconv.FormatInt(int64(newLevel), 10))
}

func layoutOutline(s tcell.Screen, o *outline, width int, height int) {
	y := 1
	o.lineIndex = []line{}
	text := o.buf.Runes()
	var start, end int
	var delim *delimiter
	var err error
	delim, start, end = o.nextHeadline(start)
	//	for end < len(*text)-2 && delim != nil { // Scan thru all headlines (stop before you hit EOL Delim)
	for delim != nil { // Scan thru all headlines (stop before you hit EOL Delim)

		if err != nil {
			fmt.Printf("%v\n", err)
			break
		}
		y = layoutHeadline(s, o, text, start, end+1, y, delim.level, false) // we use end+1 so we render the <nodeDelim>- this gives us something at end of headline to start typing on when appending text to headline
		delim, start, end = o.nextHeadline(end)
	}
}

// Format headline according to indent and word-wrap the text.
func layoutHeadline(s tcell.Screen, o *outline, text *[]rune, start int, end int, y int, level int, collapsed bool) int {
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
		o.recordLogicalLine(bullet, indent, hangingIndent, start, headlineLength-1)
		endY++
	} else { // going to have to wrap it
		pos := start
		firstLine := true
		for pos < end {
			endPos := pos + o.editorWidth
			if endPos > end { // overshot end of text, we're on the last fragment
				o.recordLogicalLine(0, indent, hangingIndent, pos, end-pos)
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
				o.recordLogicalLine(mybullet, indent, hangingIndent, pos, endPos-pos)
				endY++
			}
			pos = endPos
		}
	}

	return endY
}

// Walk thru the lineIndex and render each logical line that is within the window's boundaries
func renderOutline(s tcell.Screen, o *outline) {
	runes := *(o.buf.Runes())
	y := 1
	lastLine := o.topLine + o.editorHeight - 1
	for l := o.topLine; l <= lastLine && l < len(o.lineIndex); l++ {
		x := 0
		line := o.lineIndex[l]
		s.SetContent(x+line.indent, y, line.bullet, nil, defStyle)
		for p := line.position; p < line.position+line.length; p++ {
			// If we're rendering the current position, place cursor here, remember this is current logical line
			if o.currentPosition == p {
				cursX = line.hangingIndent + x
				cursY = y
				o.linePtr = l
			}
			s.SetContent(x+line.hangingIndent, y, runes[p], nil, defStyle)
			x++
		}
		y++
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

func (o *outline) moveRight() {
	if o.currentPosition < o.buf.lastpos-1 {
		text := *(o.buf.Runes())
		if text[o.currentPosition] == nodeDelim {
			peek := o.currentPosition + 1
			if text[peek] != eof {
				// at last character of current headline, move to beginning of next headline
				_, o.currentPosition, _ = o.nextHeadline(o.currentPosition)
			}
		} else {
			// Just move to next character in this headline
			o.currentPosition++
		}
		// Scroll?
		//   see if we've moved into next logical line
		newPtr := o.linePtr + 1
		if newPtr < len(o.lineIndex) {
			if o.currentPosition >= o.lineIndex[newPtr].position && o.linePtr-o.topLine+1 >= o.editorHeight {
				// If we've 'moved' to next logical line and at bottom row of window
				o.topLine++
			}
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
		// Scroll?
		//   see if we've moved into previous logical line
		newPtr := o.linePtr - 1
		if newPtr >= 0 {
			previousLastPosition := o.lineIndex[newPtr].position + o.lineIndex[newPtr].length - 1
			if o.currentPosition <= previousLastPosition && o.linePtr-o.topLine+1 == 1 {
				// If we've 'moved' to previous logical line and at top row of window
				o.topLine--
			}
		}

	}
}

func (o *outline) moveDown() {
	if o.currentPosition < o.buf.lastpos-1 {
		if o.linePtr != len(o.lineIndex)-1 { // Make sure we're not on last line
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
			// Scroll?
			if o.linePtr-o.topLine+1 >= o.editorHeight {
				o.topLine++
			}
		}
	}
}

func (o *outline) moveUp() {
	if o.currentPosition > 3 { // Do nothing if on first character of first headline
		offset := o.currentPosition - o.lineIndex[o.linePtr].position // how far 'in' are we on the logical line?
		newLinePtr := o.linePtr - 1
		if newLinePtr >= 0 { // There are more lines above
			dbg2 = o.lineIndex[newLinePtr].length
			if offset >= o.lineIndex[newLinePtr].length { // Are we moving up to a smaller line with x too far right?
				o.currentPosition = o.lineIndex[newLinePtr].position + o.lineIndex[newLinePtr].length - 1
			} else {
				o.currentPosition = offset + o.lineIndex[newLinePtr].position
			}
		}
		// Scroll?
		if o.linePtr != 0 && o.linePtr-o.topLine+1 == 1 {
			o.topLine--
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
	delim := delimiterString(o.currentLevel())
	o.buf.Insert(o.currentPosition, delim)
	o.moveRight()
}

/*
If Tab is hit on the first headline, do nothing.
    Otherwise if previous headline is <= level as this headline, promote this headline
	TODO: SOME BUGS WHEN WE GET TO LEVEL 10 GO FIGURE...
*/
func (o *outline) tabPressed() {
	if o.linePtr != 0 {
		dCurrent, _, _ := o.currentHeadline(o.currentPosition)
		dPrevious, _, _ := o.previousHeadline(o.currentPosition)
		if dCurrent.level <= dPrevious.level {
			o.changeRank(true)
		}
	}
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

	//s, _ := tcell.NewScreen()
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

	//o.test()

	drawScreen(s, o)

	handleEvents(s, o)

}
