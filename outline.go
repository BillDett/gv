package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/gdamore/tcell/v2"
)

/*
Okay- let's rip this thing apart and change the internal model entirely over to a hierarchy instead of a flat stream of runes
Go back to a proper tree structure- headline text is now managed individually through separate PieceTables.
lineIndex can stay the same- except we need to add a way to find out which Headline this line is part of.
The outline has a list of top level Headlines, each may have their own children Headlines, etc...

*/

type outline struct {
	headlines         []*Headline       // list of top level headlines (this denotes the structure of the outline)
	headlineIndex     map[int]*Headline // index to all Headlines (keyed by ID- this makes serialization easier than using pointers)
	lineIndex         []line            // Text Position index for each "line" after editor has been laid out.
	linePtr           int               // index of the line currently beneath the cursor
	editorWidth       int               // width of an editor column
	editorHeight      int               // height of the editor window
	currentHeadlineID int               // ID of headline cursor is on
	currentPosition   int               // the current position within the currentHeadline.Buf
	topLine           int               // index of the topmost "line" of the window in lineIndex
}

// Headline is an entry in the headlineIndex map
// Headline ID is set by its key in the headlineIndex
type Headline struct {
	ID       int
	ParentID int
	Expanded bool
	Buf      PieceTable // buffer holding the text of the headline
	Children []*Headline
}

// a line is a logical representation of a line that is rendered in the window
type line struct {
	headlineID    int  // Which headline's text are we representing?
	bullet        rune // What bullet should precede this line (if any)?
	indent        int  // Initial indent before a bullet
	hangingIndent int  // Indent for text without a bullet
	position      int  // Text position in o.lineIndex[headlineID].Buf.Runes()
	length        int  // How many runes in this "line"
}

const nodeDelim = '\ufeff'

const emptyHeadlineText = string(nodeDelim) // every Headline's text ends with a nonprinting rune so we can append to it easily

var defStyle tcell.Style

var currentLine int // which "line" we are currently on
var cursX int       // X coordinate of the cursor
var cursY int       // Y coordinate of the cursor

var dbg int
var dbg2 int

func newOutline(s tcell.Screen) *outline {
	o := &outline{[]*Headline{}, make(map[int]*Headline), nil, 0, 0, 0, 0, 0, 0}
	o.setScreenSize(s)
	return o
}

// initialize a new outline to be used as a blank outline for editing
func (o *outline) init() error {
	id, _ := o.addHeadline("", -1)
	o.currentHeadlineID = id
	o.currentPosition = 0
	return nil
}

func (h *Headline) toString(level int) string {
	buf := "\n"
	for c := 0; c < level; c++ {
		buf += "   "
	}
	text := h.Buf.Text()
	buf += fmt.Sprintf("ID: %d;Parent ID %d;", h.ID, h.ParentID)
	buf += text
	buf += fmt.Sprintf("(%d chars, %d children)", len(text), len(h.Children))
	for _, child := range h.Children {
		buf += child.toString(level + 1)
	}
	return buf
}

func (o *outline) dump() {
	text := (*o.headlineIndex[o.currentHeadlineID].Buf.Runes())
	out := "Headlines\n"
	for _, h := range o.headlines {
		out += h.toString(0) + "\n"
	}
	out += fmt.Sprintf("\ncurrentHeadline %d, currentPosition %d, current Rune (%#U) num Headlines %d, dbg %d, dbg2 %d\n",
		o.currentHeadlineID, o.currentPosition, text[o.currentPosition], len(o.headlineIndex), dbg, dbg2)
	ioutil.WriteFile("dump.txt", []byte(out), 0644)
}

// save the outline buffer to a file
func (o *outline) save(filename string) error {
	buf, err := json.Marshal(o.headlines)
	if err != nil {
		return err
	}
	ioutil.WriteFile(filename, buf, 0644)
	return nil
}

// load a .gv file and use it to populate the outline's buffer
func (o *outline) load(filename string) error {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	// Extract the outline JSON
	err = json.Unmarshal(buf, &o.headlines)
	if err != nil {
		return err
	}
	if len(o.headlines) == 0 {
		return fmt.Errorf("Error: did not read any headlines from the input file")
	}
	// (Re)build the headlineIndex
	o.headlineIndex = make(map[int]*Headline)
	for _, h := range o.headlines {
		o.addHeadlineToIndex(h)
	}
	o.currentHeadlineID = o.headlines[0].ID
	o.currentPosition = 0
	return nil
}

// Add a Headline (and all of its children) into the o.headlineIndex
func (o *outline) addHeadlineToIndex(h *Headline) {
	o.headlineIndex[h.ID] = h
	for _, c := range h.Children {
		o.addHeadlineToIndex(c)
	}
}

func (o *outline) setScreenSize(s tcell.Screen) {
	var width int
	width, height := s.Size()
	o.editorWidth = int(float64(width) * 0.7)
	o.editorHeight = height - 3 // 2 rows for border, 1 row for interaction
}

// appends a new headline onto the outline under the parent
func (o *outline) addHeadline(text string, parent int) (int, error) {
	id := nextHeadlineID(o.headlineIndex)
	h := Headline{id, parent, true, *NewPieceTable(text + emptyHeadlineText), []*Headline{}} // Note we're adding extra non-printing char to end of text
	if parent == -1 {                                                                        // Is this a top-level headline?
		o.headlines = append(o.headlines, &h)
	} else {
		p, found := o.headlineIndex[parent]
		if !found {
			return -1, fmt.Errorf("Unable to append headline to parent %d", parent)
		}
		p.Children = append(p.Children, &h)
	}
	o.headlineIndex[id] = &h
	return id, nil
}

// utility to get the next Headline id based on maximum key value in headlineIndex
func nextHeadlineID(headlines map[int]*Headline) int {
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
func (o *outline) recordLogicalLine(id int, bullet rune, indent int, hangingIndent int, position int, length int) {
	o.lineIndex = append(o.lineIndex, line{id, bullet, indent, hangingIndent, position, length})
}

/*
// Find position of next nodeDelim (based on direction) from position
func (o *outline) delim(text []rune, position int, direction int) (int, error) {
	//text := (*o.buf.Runes())
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
func (o *outline) integer(text []rune, position int, direction int) (int, int, int, error) {
	//text := (*o.buf.Runes())
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
	text := *(o.buf.Runes())
	dp1, _ := o.delim(text, 3, backward)
	dp2, _ := o.delim(text, 3, forward)
	fmt.Printf("Previous delim %d, Next delim %d\n", dp1+1, dp2+1)
	d, s, e := o.currentHeadline(text, o.currentPosition)
	fmt.Printf("Current Headline delim %d/%d/%d, start %d, end %d\n", d.lhs+1, d.id, d.rhs+1, s+1, e+1)
	d, s, e = o.nextHeadline(text, o.currentPosition)
	var last int
	c := 0
	var d2 *delimiter
	for d != nil {
		if c == 2 {
			d2 = d
		}
		fmt.Printf("Next Headline delim %d/%d/%d, start %d, end %d\n", d.lhs+1, d.id, d.rhs+1, s+1, e+1)
		d, s, e = o.nextHeadline(text, s)
		if e > last {
			last = s
		}
		c++
	}
	d, s, e = o.nextHeadline(text, 10)
	if d != nil {
		fmt.Printf("Second Headline delim %d/%d/%d, start %d, end %d\n", d.lhs+1, d.id, d.rhs+1, s+1, e+1)
	}
	fmt.Printf("d2 delim %d/%d/%d\n", d.lhs+1, d.id, d.rhs+1)
	o.setLevel(d2, 5)
	d2, s2, e2 := o.currentHeadline(text, d2.rhs+1)
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
	return o.headlineIndex[id].Level
}
*/

// Get the current Headline
func (o *outline) currentHeadline() *Headline {
	return o.headlineIndex[o.currentHeadlineID]
}

// Get "next" headline in a Depth-first approach based upon current Headline
//  Return first child.  If no children, return sibling.  If no sibling, get parent's sibling.  If at top level, return nil
//  Return nil if no next Headline
func (o *outline) nextHeadline() *Headline {
	return nil
}

// Get "previous" headline in tree
func (o *outline) previousHeadline() *Headline {
	return nil
}

/*
// Adjust the level of current outline by adding difference and adjust all subsequent "child" headlines.
// We determine when we are finished finding children when there are no more headlines or next headline's level > this headline
func (o *outline) changeRank(difference int) {
	text := *(o.buf.Runes())
	d, _, _ := o.currentHeadline(text, o.currentPosition)
	//fmt.Printf("current delim %d/%d/%d\n", d.lhs, d.level, d.rhs)
	origLevel := o.headlineIndex[d.id].Level
	o.setLevel(d, origLevel+difference)
	// add the difference to all children
	var s int
	d, s, _ = o.nextHeadline(text, o.currentPosition)
	if d != nil {
		dl := o.headlineIndex[d.id].Level
		for d != nil && dl > origLevel {
			//fmt.Printf("next delim %d/%d/%d\n", d.lhs, d.level, d.rhs)
			offset := o.setLevel(d, dl+difference)
			d, s, _ = o.nextHeadline(text, s+offset)
		}
	}
}
*/
// Modify the level in the buffer for this delimiter (replace characters for the integer)
// Return the # of runes that we have adjusted the buffer (add/remove) based on size of newLevel compared to d.level

// TODO: REMOVE THE RETURN VALUE- NOT NEEDED SINCE LEVEL IS NO LONGER IN BUFFER
/*
func (o *outline) setLevel(d *delimiter, newLevel int) int {
	//newLevelStr := strconv.FormatInt(int64(newLevel), 10)
	//newLevelLen := len(newLevelStr)
	levelLen := d.rhs - d.lhs - 1
	//fmt.Printf("Update >%d< with new level %d\n", d.level, newLevel)
	//o.buf.Delete(d.lhs+1, levelLen)
	//o.buf.Insert(d.lhs+1, newLevelStr)
	o.headlineIndex[d.id].Level = newLevel
	//return newLevelLen - levelLen // Did we add or remove runes to the buffer with this change of level?
	return levelLen
}
*/

func (o *outline) moveRight() {
	o.currentPosition++
	/*
		if o.currentPosition < o.buf.lastpos-1 {
			text := *(o.buf.Runes())
			if text[o.currentPosition] == nodeDelim {
				peek := o.currentPosition + 1
				if text[peek] != eof {
					// at last character of current headline, move to beginning of next headline
					_, o.currentPosition, _ = o.nextHeadline(text, o.currentPosition)
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
	*/
}

// TODO: We can probably simplify this a bit removing o.buf.Runes and loop skipping over delims
func (o *outline) moveLeft() {
	/*
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
	*/
}

func (o *outline) moveDown() {
	/*
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
	*/
}

func (o *outline) moveUp() {
	/*
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
	*/
}

func (o *outline) insertRuneAtCurrentPosition(r rune) {
	h := o.currentHeadline()
	h.Buf.InsertRunes(o.currentPosition, []rune{r})
	o.moveRight()
}

func (o *outline) backspace() {
	/*
		if o.currentPosition > 3 {
			text := *(o.buf.Runes())
			posToRemove := o.currentPosition - 1
			if text[posToRemove] == nodeDelim { // Are we trying to join with previous headline?
				if posToRemove != 2 { // Make sure we're not on first headline
					var start int
					for start = posToRemove - 1; text[start] != nodeDelim; start-- { // find start/end of the nodeDelims
					}
					d, _, _ := o.currentHeadline(text, o.currentPosition)
					delete(o.headlineIndex, d.id)
					o.buf.Delete(start, posToRemove-start+1) // Remove the <nodeDelim><Level><nodeDelim> fragment
				}
			} else {
				o.moveLeft()
				o.buf.Delete(o.currentPosition, 1)
			}
			o.dump()
		}
	*/
}

// TODO: This is pretty clunky when we join headlines- we can streamline this code a lot
func (o *outline) delete() {
	/*
		if o.buf.lastpos > 2 {
			text := *(o.buf.Runes())
			if text[o.currentPosition] == nodeDelim { // Are we trying to join the next headline?
				if o.currentPosition != o.buf.lastpos-2 { // Make sure we're not on <nodeDelim> after last headline
					var start int
					for start = o.currentPosition + 1; text[start] != nodeDelim; start++ { // find start/end of the nodeDelims
					}
					d, _, _ := o.nextHeadline(text, o.currentPosition)
					delete(o.headlineIndex, d.id)
					o.buf.Delete(o.currentPosition, start-o.currentPosition+1) // Remove the <nodeDelim><Level><nodeDelim> fragment
				}
			} else { // Just delete the current position
				o.buf.Delete(o.currentPosition, 1)
			}
		}
	*/
}

// Enter always creates a new headline at current position at same level as current headline
func (o *outline) enterPressed() {
	h := o.currentHeadline()
	o.addHeadline("", h.ParentID)
	o.moveRight()
}

/*
If Tab is hit on the first headline, do nothing.
    Otherwise if previous headline is <= level as this headline, promote this headline
*/
func (o *outline) tabPressed(promote bool) {
	/*
		if o.linePtr != 0 {
			text := *(o.buf.Runes())
			dCurrent, _, _ := o.currentHeadline(text, o.currentPosition)
			//fmt.Printf("dCurrent %d/%d/%d\n", dCurrent.lhs, dCurrent.level, dCurrent.rhs)
			dPrevious, _, _ := o.previousHeadline(text, o.currentPosition)
			//fmt.Printf("dPrevious %d/%d/%d\n", dPrevious.lhs, dPrevious.level, dPrevious.rhs)
			currentLevel := o.headlineIndex[dCurrent.id].Level
			previousLevel := o.headlineIndex[dPrevious.id].Level
			if promote && currentLevel <= previousLevel {
				o.changeRank(1)
			} else if !promote && currentLevel > 0 {
				o.changeRank(-1)
			}
		}
	*/
}

// Delete the current headline (if we're not on the first headline).  Also delete all children.
// If on first headline and this is the only headline, remove all of the text (but keep the headline there since we always need at least one headline)
func (o *outline) deleteHeadline() {
	/*
		var s, e, currentStart, start, end int
		var dCurrent, d *delimiter
		text := *(o.buf.Runes())
		dCurrent, currentStart, end = o.currentHeadline(text, o.currentPosition)
		d, s, e = o.nextHeadline(text, o.currentPosition)
		if o.linePtr != 0 {
			_, prevStart, _ := o.previousHeadline(text, o.currentPosition)
			start = dCurrent.lhs
			for d != nil && o.headlineIndex[d.id].Level > o.headlineIndex[dCurrent.id].Level {
				end = e
				d, s, e = o.nextHeadline(text, s)
			}
			// Now we should be able to delete start:end-1 from buf
			extent := end - start
			o.buf.Delete(start, extent)
			o.currentPosition = prevStart
			delete(o.headlineIndex, dCurrent.id)
		} else if d == nil { // delete all text in first headline if it's the only one left
			extent := end - currentStart
			o.buf.Delete(currentStart, extent)
			o.currentPosition = 3
		}
	*/
}

// Collapse the current headline and all children
//  (this just marks each headline as invisible)
func (o *outline) collapse() {

}

// Expand the current headline (if necessary) and all children
//  (this just marks each headline as visible)
func (o *outline) expand() {

}
