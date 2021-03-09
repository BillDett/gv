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
	buf             PieceTable        // buffer holding the text of the outline
	lineIndex       []line            // Text Position index for each "line" after editor has been laid out.
	linePtr         int               // index of the line currently beneath the cursor
	headlineIndex   map[int]*headline // map of headline metadata keyed by headline id
	editorWidth     int               // width of an editor column
	editorHeight    int               // height of the editor window
	currentPosition int               // the current position within the buf
	topLine         int               // index of the topmost "line" of the window in lineIndex
}

type headline struct {
	id          int
	level       int
	visible     bool
	haschildren bool
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
	lhs int // position of nodeDelim on left
	//level int // level of headline
	id  int // identifier for headline
	rhs int // position of nodeDelim on right
}

// for tokenization
const (
	forward = iota
	backward
)

/*
 Outline structure takes the following form:
 	<nodeDelim><id><nodeDelim><headline><nodeDelim><id><nodeDelim><headline><nodeDelim><EOF>

 where headline order denotes tree structure.  <id> is the unique id of the headline.   Tree is
 denoted by an increasing headline.level value in consecutive headlines.  Once headline.level is
 found to be less than previous headline.level, we have just seen a leaf node.

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

func delimiterString(level int) string {
	return fmt.Sprintf("%c%d%c", nodeDelim, level, nodeDelim)
}

func newOutline(s tcell.Screen) *outline {
	o := &outline{*NewPieceTable(""), nil, 0, make(map[int]*headline), 0, 0, 3, 0}
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
	var out string
	for _, r := range *text {
		out += fmt.Sprintf("(%#U)", r)
	}
	out += "===========================================================================\nHeadlineIndex\n"
	out += fmt.Sprintf("num keys %d\n", len(o.headlineIndex))
	d, s, e := o.currentHeadline(o.currentPosition)
	out += fmt.Sprintf("\nlastPos %d\tcurrentPosition %d (%#U)\tlinePtr %d\nlineIndex %v\nlineIndex.position %d\tlineIndex.length %v\tdbg %d\tdbg2 %d\n# of lines: %d\ttopLine %d\teditorheight %d\ncurrent delim %v (%d:%d)\n",
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
	// TODO
	// TODO: RECONSTRUCT THE HEADLINE INDEX!!
	// TODO
}

func (o *outline) setScreenSize(s tcell.Screen) {
	var width int
	width, height := s.Size()
	o.editorWidth = int(float64(width) * 0.7)
	o.editorHeight = height - 3 // 2 rows for border, 1 row for interaction
}

// appends a new headline onto the outline (before the final <EOFDelim>)
func (o *outline) addHeadline(text string, level int) {
	id := nextHeadlineID(o.headlineIndex)
	h := &headline{id, level, true, false}
	o.headlineIndex[id] = h
	o.buf.Insert(o.buf.lastpos-2, delimiterString(id)+text)
}

// utility to get the next Headline id based on maximum key value in headlineIndex
func nextHeadlineID(headlines map[int]*headline) int {
	var maxNumber int
	for n := range headlines {
		if n > maxNumber {
			maxNumber = n
		}
	}
	return maxNumber + 1
}

// Store a 'logical' line- this is a rendered line of text on the screen. We use this index
// to figure out where in the outline buffer to move to when we navigate visually
func (o *outline) recordLogicalLine(bullet rune, indent int, hangingIndent int, position int, length int) {
	o.lineIndex = append(o.lineIndex, line{bullet, indent, hangingIndent, position, length})
}

// Find position of next nodeDelim (based on direction) from position
func (o *outline) delim(position int, direction int) (int, error) {
	text := (*o.buf.Runes())
	var start int
	if direction == forward {
		for start = position; text[start] != nodeDelim && start < len(text); start++ {
		}
	} else {
		for start = position; text[start] != nodeDelim && start > 0; start-- {
		}
	}
	if start == len(text) {
		return 0, fmt.Errorf("unable to find delimiter from %d in direction %d", position, direction)
	}
	return start, nil
}

// Find the start and end position and numerical value of an integer (based on direction) from position
//   Position must be sitting on a digit character
func (o *outline) integer(position int, direction int) (int, int, int, error) {
	text := (*o.buf.Runes())
	var start, end, level int
	if !unicode.IsDigit(text[position]) { // This is an error- we must be on a digit
		//fmt.Printf("Saw (%#U)\n", text[position])
		return 0, 0, 0, fmt.Errorf("unable to convert level integer from position %d (%#U)", position, text[position])
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
	level, err := strconv.Atoi(is)
	return start, end, level, err
}

func (o *outline) test() {
	dp1, _ := o.delim(3, backward)
	dp2, _ := o.delim(3, forward)
	fmt.Printf("Previous delim %d, Next delim %d\n", dp1+1, dp2+1)
	d, s, e := o.currentHeadline(o.currentPosition)
	fmt.Printf("Current Headline delim %d/%d/%d, start %d, end %d\n", d.lhs+1, d.id, d.rhs+1, s+1, e+1)
	d, s, e = o.nextHeadline(o.currentPosition)
	var last int
	c := 0
	var d2 *delimiter
	for d != nil {
		if c == 2 {
			d2 = d
		}
		fmt.Printf("Next Headline delim %d/%d/%d, start %d, end %d\n", d.lhs+1, d.id, d.rhs+1, s+1, e+1)
		d, s, e = o.nextHeadline(s)
		if e > last {
			last = s
		}
		c++
	}
	d, s, e = o.nextHeadline(10)
	if d != nil {
		fmt.Printf("Second Headline delim %d/%d/%d, start %d, end %d\n", d.lhs+1, d.id, d.rhs+1, s+1, e+1)
	}
	fmt.Printf("d2 delim %d/%d/%d\n", d.lhs+1, d.id, d.rhs+1)
	o.setLevel(d2, 5)
	d2, s2, e2 := o.currentHeadline(d2.rhs + 1)
	fmt.Printf("now d2 delim %d/%d/%d, start %d, end %d\n", d2.lhs+1, d2.id, d2.rhs+1, s2+1, e2+1)
}

// Get positional information for headline under o.currentPosition
// Walk backwards from current positon until you find the leading <nodeDelim> and then extract the id to get the level from the headlineIndex
// TODO: CONVERT TO USE TOKENIZER METHODS INSTEAD
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
	// extract the id
	for start = rhs - 1; text[start] != nodeDelim; start-- {
	}
	id, _ := strconv.Atoi(string(text[start+1 : rhs]))
	return o.headlineIndex[id].level
}

// Get the delimiter and start/end of the current headline based on current position
func (o *outline) currentHeadline(position int) (*delimiter, int, int) {
	text := *(o.buf.Runes())
	d := delimiter{0, 0, 0}
	var err error
	var extent int
	if text[position] == nodeDelim { // "back up" if we're sitting on a trailing <nodeDelim> at end of a line
		position--
	}
	d.rhs, err = o.delim(position, backward)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	s, _, l, err := o.integer(d.rhs-1, backward)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	if s != -1 {
		//d.level = l
		d.id = l
		d.lhs, err = o.delim(s, backward)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		extent, err = o.delim(d.rhs+1, forward) // find end of headline text
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		return &d, d.rhs + 1, extent
	} else {
		return nil, 0, 0
	}
}

// Tokenize the text to extract the next Headline, returning delimiter, start position and end position of headline
//  Return a nil delimiter if we're on last Headline
func (o *outline) nextHeadline(position int) (*delimiter, int, int) {
	text := *(o.buf.Runes())
	//fmt.Printf("Getting next headline from position %d (%#U)\n", position, text[position])
	var p, q int
	var err error
	// scan to end of this headline, find the first nodeDelim
	p, err = o.delim(position, forward)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	// make sure we're not at end of the outline
	if text[p+1] != eof {
		// scan forward find the matching nodeDelim
		q, err = o.delim(p+1, forward)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
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

// Adjust the level of current outline by adding difference and adjust all subsequent "child" headlines.
// We determine when we are finished finding children when there are no more headlines or next headline's level > this headline
func (o *outline) changeRank(difference int) {
	d, _, _ := o.currentHeadline(o.currentPosition)
	//fmt.Printf("current delim %d/%d/%d\n", d.lhs, d.level, d.rhs)
	origLevel := o.headlineIndex[d.id].level
	o.setLevel(d, origLevel+difference)
	// add the difference to all children
	var s int
	d, s, _ = o.nextHeadline(o.currentPosition)
	if d != nil {
		dl := o.headlineIndex[d.id].level
		for d != nil && dl > origLevel {
			//fmt.Printf("next delim %d/%d/%d\n", d.lhs, d.level, d.rhs)
			offset := o.setLevel(d, dl+difference)
			d, s, _ = o.nextHeadline(s + offset)
		}
	}
}

// Modify the level in the buffer for this delimiter (replace characters for the integer)
// Return the # of runes that we have adjusted the buffer (add/remove) based on size of newLevel compared to d.level

// TODO: REMOVE THE RETURN VALUE- NOT NEEDED SINCE LEVEL IS NO LONGER IN BUFFER

func (o *outline) setLevel(d *delimiter, newLevel int) int {
	//newLevelStr := strconv.FormatInt(int64(newLevel), 10)
	//newLevelLen := len(newLevelStr)
	levelLen := d.rhs - d.lhs - 1
	//fmt.Printf("Update >%d< with new level %d\n", d.level, newLevel)
	//o.buf.Delete(d.lhs+1, levelLen)
	//o.buf.Insert(d.lhs+1, newLevelStr)
	o.headlineIndex[d.id].level = newLevel
	//return newLevelLen - levelLen // Did we add or remove runes to the buffer with this change of level?
	return levelLen
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
	id := nextHeadlineID(o.headlineIndex)
	h := &headline{id, o.currentLevel(), true, false}
	o.headlineIndex[id] = h
	delim := delimiterString(id)
	o.buf.Insert(o.currentPosition, delim)
	o.moveRight()
}

/*
If Tab is hit on the first headline, do nothing.
    Otherwise if previous headline is <= level as this headline, promote this headline
*/
func (o *outline) tabPressed(promote bool) {
	if o.linePtr != 0 {
		dCurrent, _, _ := o.currentHeadline(o.currentPosition)
		//fmt.Printf("dCurrent %d/%d/%d\n", dCurrent.lhs, dCurrent.level, dCurrent.rhs)
		dPrevious, _, _ := o.previousHeadline(o.currentPosition)
		//fmt.Printf("dPrevious %d/%d/%d\n", dPrevious.lhs, dPrevious.level, dPrevious.rhs)
		currentLevel := o.headlineIndex[dCurrent.id].level
		previousLevel := o.headlineIndex[dPrevious.id].level
		if promote && currentLevel <= previousLevel {
			o.changeRank(1)
		} else if !promote && currentLevel > 0 {
			o.changeRank(-1)
		}
	}
}

// Delete the current headline (if we're not on the first headline).  Also delete all children.
// If on first headline and this is the only headline, remove all of the text (but keep the headline there since we always need at least one headline)
func (o *outline) deleteHeadline() {
	var s, e, currentStart, start, end int
	var dCurrent, d *delimiter
	dCurrent, currentStart, end = o.currentHeadline(o.currentPosition)
	d, s, e = o.nextHeadline(o.currentPosition)
	if o.linePtr != 0 {
		_, prevStart, _ := o.previousHeadline(o.currentPosition)
		start = dCurrent.lhs
		for d != nil && o.headlineIndex[d.id].level > o.headlineIndex[dCurrent.id].level {
			end = e
			d, s, e = o.nextHeadline(s)
		}
		// Now we should be able to delete start:end-1 from buf
		extent := end - start
		o.buf.Delete(start, extent)
		o.currentPosition = prevStart
		delete(o.headlineIndex, d.id)
	} else if d == nil { // delete all text in first headline if it's the only one left
		extent := end - currentStart
		o.buf.Delete(currentStart, extent)
		o.currentPosition = 3
	}
}

// Collapse the current headline and all children
//  (this just marks each headline as invisible)
func (o *outline) collapse() {

}

// Expand the current headline (if necessary) and all children
//  (this just marks each headline as visible)
func (o *outline) expand() {

}
