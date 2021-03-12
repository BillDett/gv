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
	out := "Headline and children\n"
	//i, c := o.childrenSliceFor(13)
	for _, h := range o.headlines {
		out += h.toString(0) + "\n"
	}
	out += fmt.Sprintf("\nlinePtr %d, currentHeadline %d, currentPosition %d, current Rune (%#U) num Headlines %d, dbg %d, dbg2 %d\n",
		o.linePtr, o.currentHeadlineID, o.currentPosition, text[o.currentPosition], len(o.headlineIndex), dbg, dbg2)
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

func (o *outline) newHeadline(text string, parent int) *Headline {
	id := nextHeadlineID(o.headlineIndex)
	return &Headline{id, parent, true, *NewPieceTable(text + emptyHeadlineText), []*Headline{}} // Note we're adding extra non-printing char to end of text
}

// appends a new headline onto the outline under the parent
func (o *outline) addHeadline(text string, parent int) (int, error) {
	h := o.newHeadline(text, parent)
	if parent == -1 { // Is this a top-level headline?
		o.headlines = append(o.headlines, h)
	} else {
		p, found := o.headlineIndex[parent]
		if !found {
			return -1, fmt.Errorf("Unable to append headline to parent %d", parent)
		}
		p.Children = append(p.Children, h)
	}
	o.headlineIndex[h.ID] = h
	return h.ID, nil
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

// Return the IDs of the Headlines just before and after the Headline at given ID.  Return -1 for either if at beginning or end of outline.
// We leverage the fact that o.lineIndex is really a 'flattened' DFS list of Headlines, so it has the ordered list of Headlines
func (o *outline) prevNextFrom(ID int) (int, int) {
	previous := -1
	next := -1
	// generate list of ordered, unique headline IDs from o.lineIndex
	var headlines []int
	for _, l := range o.lineIndex {
		if len(headlines) == 0 || headlines[len(headlines)-1] != l.headlineID {
			headlines = append(headlines, l.headlineID)
		}
	}

	// now find previous and next
	for c, i := range headlines {
		if i == ID {
			if c < len(headlines)-1 {
				next = headlines[c+1]
			}
			if c > 0 {
				previous = headlines[c-1]
			}
			break
		}
	}

	return previous, next
}

// Look up the index in the []*Headline where this Headline is being managed by its parent
func (o *outline) childrenSliceFor(ID int) (int, *[]*Headline) {
	index := -1
	var children *[]*Headline
	h := o.headlineIndex[ID]
	if h != nil {
		if h.ParentID == -1 {
			children = &o.headlines
		} else {
			children = &o.headlineIndex[h.ParentID].Children
		}
		for i, c := range *children {
			if c.ID == ID {
				index = i
				break
			}
		}
	}
	return index, children
}

// Find the "next" Headline after the Headline with ID.  Return nil if no more Headlines are next.
func (o *outline) nextHeadline(ID int) *Headline {
	if ID == -1 {
		return nil
	}
	_, n := o.prevNextFrom(ID)
	if n == -1 {
		return nil
	}
	return o.headlineIndex[n]
}

// Find the "previous" Headline befoire the Headline with ID.  Return nil if no more Headlines are prior.
func (o *outline) previousHeadline(ID int) *Headline {
	if ID == -1 {
		return nil
	}
	p, _ := o.prevNextFrom(ID)
	if p == -1 {
		return nil
	}
	return o.headlineIndex[p]
}

// Find the "current" Headline
func (o *outline) currentHeadline() *Headline {
	return o.headlineIndex[o.currentHeadlineID]
}

func (o *outline) moveRight() {
	previousHeadlineID := o.currentHeadlineID
	if o.currentPosition < o.headlineIndex[o.currentHeadlineID].Buf.lastpos-1 { // are we within the text of current Headline?
		o.currentPosition++
	} else { // move to the first character of next Headline (if one exists)
		h := o.nextHeadline(o.currentHeadlineID)
		if h != nil {
			o.currentHeadlineID = h.ID
			o.currentPosition = 0
		} else { // no more Headlines
			return
		}
	}
	// Do we need to scroll?
	newPtr := o.linePtr + 1
	if newPtr < len(o.lineIndex) { // we have additional logical lines beneath us
		if o.linePtr-o.topLine+1 >= o.editorHeight { // we are on last row of editor window
			if o.currentHeadlineID == previousHeadlineID { // we are on same Headline
				if o.currentPosition >= o.lineIndex[newPtr].position { // We've 'moved' to next logical line
					o.topLine++
				}
			} else { // We moved to a new headline, so is must be a new logical line
				o.topLine++
			}
		}
	}
}

func (o *outline) moveLeft() {
	if o.currentPosition == 0 && o.linePtr == 0 { // Do nothing if on first character of first headline
		return
	} else {
		previousHeadlineID := o.currentHeadlineID
		if o.currentPosition > 0 { // Just move to previous character in this headline
			o.currentPosition--
		} else { // at first character of current headline, move to end of previous headline
			p := o.previousHeadline(o.currentHeadlineID)
			if p != nil {
				o.currentHeadlineID = p.ID
				o.currentPosition = p.Buf.lastpos - 1
			}
		}
		// Do we need to scroll?
		newPtr := o.linePtr - 1
		if newPtr >= 0 {
			if o.linePtr-o.topLine+1 == 1 { // we are on first row of editor window
				if o.currentHeadlineID == previousHeadlineID { // we are on same Headline
					if o.currentPosition <= o.lineIndex[newPtr].position+o.lineIndex[newPtr].length { // We've 'moved' to previous logical line
						o.topLine--
					}
				} else { // We moved to a new headline, so is must be a new logical line
					o.topLine--
				}
			}
		}
	}
}

func (o *outline) moveDown() {
	if o.linePtr != len(o.lineIndex)-1 { // Make sure we're not on last line
		offset := o.currentPosition - o.lineIndex[o.linePtr].position // how far 'in' are we on the logical line?
		dbg = offset
		newLinePtr := o.linePtr + 1
		if newLinePtr < len(o.lineIndex) { // There are more lines below us
			if offset >= o.lineIndex[newLinePtr].length { // Are we moving down to a smaller line with x too far right?
				o.currentPosition = o.lineIndex[newLinePtr].position + o.lineIndex[newLinePtr].length - 1
			} else {
				o.currentPosition = offset + o.lineIndex[newLinePtr].position
			}
			o.currentHeadlineID = o.lineIndex[newLinePtr].headlineID // pick up this logical line's headlineID just in case we move to a new Headline
		}
		// Scroll?
		if o.linePtr-o.topLine+1 >= o.editorHeight {
			o.topLine++
		}
	}
}

func (o *outline) moveUp() {
	if o.linePtr != 0 { // Do nothing if on first logical line
		offset := o.currentPosition - o.lineIndex[o.linePtr].position // how far 'in' are we on the logical line?
		newLinePtr := o.linePtr - 1
		if newLinePtr >= 0 { // There are more lines above
			if offset >= o.lineIndex[newLinePtr].length { // Are we moving up to a smaller line with x too far right?
				o.currentPosition = o.lineIndex[newLinePtr].position + o.lineIndex[newLinePtr].length - 1
			} else {
				o.currentPosition = offset + o.lineIndex[newLinePtr].position
			}
			o.currentHeadlineID = o.lineIndex[newLinePtr].headlineID // pick up this logical line's headlineID just in case we move to a new Headline
		}
		// Scroll?
		if o.linePtr != 0 && o.linePtr-o.topLine+1 == 1 {
			o.topLine--
		}
	}
}

func (o *outline) insertRuneAtCurrentPosition(r rune) {
	h := o.currentHeadline()
	h.Buf.InsertRunes(o.currentPosition, []rune{r})
	o.moveRight()
}

func (o *outline) backspace() {
	if o.currentPosition == 0 && o.linePtr == 0 { // Do nothing if on first character of first headline
		return
	} else {
		currentHeadline := o.currentHeadline()
		if o.currentPosition > 0 { // Remove previous character
			posToRemove := o.currentPosition - 1
			currentHeadline.Buf.Delete(posToRemove, 1)
			o.moveLeft()
		} else { // Join this headline with previous one
			/*
				TODO: Need to think this thru- getting pretty convoluted

				previousHeadline := o.previousHeadline(currentHeadline.ID)
				if previousHeadline != nil {
					o.currentPosition = previousHeadline.Buf.lastpos - 1
					previousHeadline.Buf.Append(currentHeadline.Buf.Text())
					// Tidy up the parent's children slice
					var children []*Headline
					if currentHeadline.ParentID != -1 {
						children = o.headlineIndex[currentHeadline.ParentID].Children
					} else {
						children = o.headlines
					}
					for c := range(children) {
						if children[c].ID == currentHeadline.ID {
							break
						}
					}
					o.currentHeadlineID = previousHeadline.ID
					delete(o.headlineIndex, currentHeadline.ID)
				}
			*/
		}
	}
}

func (o *outline) delete() {
	currentHeadline := o.currentHeadline()
	if o.currentPosition != currentHeadline.Buf.lastpos-1 { // Just delete the current position
		currentHeadline.Buf.Delete(o.currentPosition, 1)
	} else { // Join the next Headline onto this one
		// TODO
	}
}

/*
Enter always creates a new headline at current position at same level as current headline

Split the current Headline's text at the cursor point.  Put all text from cursor to end into a
new Headline that is the next sibling of current Headline.
*/
func (o *outline) enterPressed() {

	// "Split" current Headline at cursor position and create a new Headline with remaining text
	currentHeadline := o.currentHeadline()
	text := (*currentHeadline.Buf.Runes())
	newText := text[o.currentPosition : len(text)-1] // Extract remaining text (except trailing nodeDelim)
	currentHeadline.Buf.Delete(o.currentPosition, len(newText))
	newHeadline := o.newHeadline(string(newText), currentHeadline.ParentID)

	// Where to put the new Headline?  If we have children, make it first child.  Otherwise it should
	//  be our next sibling.
	if len(currentHeadline.Children) == 0 {
		// Insert the new headline as next sibling after current Headline
		idx, children := o.childrenSliceFor(currentHeadline.ID)
		insertSibling(children, idx+1, newHeadline)
	} else {
		newHeadline.ParentID = currentHeadline.ID
		insertSibling(&currentHeadline.Children, 0, newHeadline)
	}

	// Update the o.headlinesIndex
	o.headlineIndex[newHeadline.ID] = newHeadline
	o.currentHeadlineID = newHeadline.ID
	o.currentPosition = 0

}

// Insert a Headline into a children slice at the given index
//  Updates the provided slice of Headlines
// 0 <= index <= len(children)
func insertSibling(children *[]*Headline, index int, value *Headline) { //*[]*Headline {
	*children = append(*children, nil)
	copy((*children)[index+1:], (*children)[index:])
	(*children)[index] = value
}

// Find childID in list of children, remove it from the list
func (o *outline) removeChildFrom(children *[]*Headline, childID int) {
	var i int
	var c *Headline
	for i, c = range *children {
		if c.ID == childID {
			break
		}
	}
	if i == len(*children) { // We didn't find this childID
		fmt.Printf("Hm- was asked to remove child %d from list %v but didn't find it", childID, *children)
		return
	}
	// Remove the child
	s := children
	copy((*s)[i:], (*s)[i+1:]) // Shift s[i+1:] left one index.
	(*s)[len(*s)-1] = nil      // Erase last element (write zero value).
	*s = (*s)[:len(*s)-1]      // Truncate slice.
}

/*
 Promote a Headline further down the outline one level
*/
func (o *outline) tabPressed() {
	if o.linePtr != 0 {
		currentHeadline := o.currentHeadline()
		previousHeadline := o.previousHeadline(o.currentHeadlineID)
		if currentHeadline.ParentID != previousHeadline.ID { // Are we already "promoted"?
			idx, children := o.childrenSliceFor(currentHeadline.ID)
			o.removeChildFrom(children, currentHeadline.ID)
			if previousHeadline.ParentID == currentHeadline.ParentID { // this means previous Headline has no children
				// we need to become first child of previous Headline
				insertSibling(&previousHeadline.Children, 0, currentHeadline)
				currentHeadline.ParentID = previousHeadline.ID
			} else { // we need to become the last child of previous sibling
				previousSibling := (*children)[idx-1]
				insertSibling(&previousSibling.Children, len(previousSibling.Children), currentHeadline)
				currentHeadline.ParentID = previousSibling.ID
			}
		}
	}
}

/*
  "Demote" a Headline back up the outline one level
*/
func (o *outline) backTabPressed() {
	if o.linePtr != 0 {
		currentHeadline := o.currentHeadline()
		//previousHeadline := o.previousHeadline(o.currentHeadlineID)
		if currentHeadline.ParentID != -1 { // it is possible to demote us
			// any siblings after us in my parent's chlidren list should be added to end of my list of children
			idx, children := o.childrenSliceFor(currentHeadline.ID)
			if len(*children) > 1 { // we have siblings after us, move them to be our children
				for c := idx + 1; c < len(*children); c++ {
					insertSibling(&currentHeadline.Children,
						len(currentHeadline.Children), (*children)[c]) // add to end of my children list
					(*children)[c].ParentID = currentHeadline.ID
					o.removeChildFrom(children, (*children)[c].ID) // remove from my parent
				}
			}
			// make me the next sibling of my parent
			parentHeadline := o.headlineIndex[currentHeadline.ParentID]
			idx, children = o.childrenSliceFor(parentHeadline.ID)
			o.removeChildFrom(&parentHeadline.Children, currentHeadline.ID)
			insertSibling(children, idx+1, currentHeadline)
			currentHeadline.ParentID = parentHeadline.ParentID
		}
	}
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
