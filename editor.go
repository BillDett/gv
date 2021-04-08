package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
)

/*

Editor is the main part of the screen, responsible for editing the outline.  It also contains the lineIndex
which acts like a 'render buffer' of logical lines of text that have been laid out within the boundary of
the editor's window.


*/

type editor struct {
	org                *organizer // pointer to the organizer
	out                *Outline   // current outline being edited
	lineIndex          []*line    // Text Position index for each "line" after editor has been laid out.
	linePtr            int        // index of the line currently beneath the cursor
	editorWidth        int        // width of an editor column
	editorHeight       int        // height of the editor window
	currentHeadlineID  int        // ID of headline cursor is on
	currentPosition    int        // the current position within the currentHeadline.Buf
	topLine            int        // index of the topmost "line" of the window in lineIndex
	dirty              bool       // Is the outliine buffer modified since last save?
	sel                *selection // pointer to the current selection (nil means we are not selecting any text)
	headlineClipboard  *Headline  // pointer to the currently copied/cut Headline (nil if nothing being copied/cut)
	selectionClipboard *[]rune    // pointer to a slice of runes containing copied/cut selecton text (nil if nothing copied/cut)
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

// A selection indicates the start and end positions of contiguous Headline text that is selected
// We do not support selecting text *across* Headlines as that is really difficult to do right
type selection struct {
	headlineID    int // ID of headline where selection occurs
	startPosition int // outline buf position where selection starts
	endPosition   int // outline buf position where selection occurs
}

var currentLine int // which "line" we are currently on
var cursX int       // X coordinate of the cursor
var cursY int       // Y coordinate of the cursor

func (e *editor) setScreenSize(s tcell.Screen) {
	var width int
	width, height := s.Size()
	//e.editorWidth = int(float64(width-e.org.width-3) * 0.9)
	e.editorWidth = width - e.org.width - 3
	e.editorHeight = height - 3 // 2 rows for border, 1 row for interaction
}

func newEditor(org *organizer) *editor {
	o := newOutline("")
	ed := &editor{org, o, nil, 0, 0, 0, 0, 0, 0, false, nil, nil, nil}
	o.init(ed)
	return ed
}

func (e *editor) isSelecting() bool { return e.sel != nil }

// save the outline buffer to a file
func (e *editor) save(filename string) error {
	buf, err := json.Marshal(ed.out)
	if err != nil {
		return err
	}
	ioutil.WriteFile(filename, buf, 0644)
	return nil
}

// User is about to change editor contents or quit, see if they want to save current editor first.
func (e *editor) saveFirst(s tcell.Screen) bool {
	response := prompt(s, "Save first (Y/N)?")
	if response != "" {
		if strings.ToUpper(response) == "Y" {
			if currentFilename != "" {
				e.save(filepath.Join(org.currentDirectory, currentFilename))
			} else {
				f := prompt(s, "Filename: ")
				if f != "" {
					currentFilename = f
					err := e.save(filepath.Join(org.currentDirectory, currentFilename))
					if err == nil {
						e.dirty = false
						setFileTitle(currentFilename)
						org.refresh(s)
					} else {
						msg := fmt.Sprintf("Error saving file: %v", err)
						prompt(s, msg)
					}
				}
			}
		}
		return true
	}
	return false
}

// user wants to create a new outline, save an existing, dirty one first
func (e *editor) newOutline(s tcell.Screen) error {
	proceed := true
	if e.dirty {
		// Prompt to save current outline first
		proceed = e.saveFirst(s)
	}
	if proceed {
		title := prompt(s, "Enter new outline title:")
		if title != "" {
			e.out = newOutline(title)
			e.out.init(e)
			e.linePtr = 0
			e.topLine = 0
			e.dirty = true
			currentFilename = ""
			e.sel = nil
			setFileTitle(currentFilename)
		}
	}
	return nil
}

// user wants to open this outline, save an existing, dirty one first
func (e *editor) open(s tcell.Screen, filename string) error {
	proceed := true
	if e.dirty {
		// Prompt to save current outline first
		proceed = e.saveFirst(s)
	}
	if proceed {
		err := ed.load(filename)
		if err == nil {
			currentFilename = filepath.Base(filename)
			setFileTitle(currentFilename)
		} else {
			msg := fmt.Sprintf("Error opening file: %v", err)
			prompt(s, msg)
		}
	}
	return nil
}

// load a .gv file and use it to populate the outline's buffer
func (e *editor) load(filename string) error {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	// Extract the outline JSON
	e.out = nil
	err = json.Unmarshal(buf, &(e.out))
	if err != nil {
		return err
	}
	if len(e.out.Headlines) == 0 {
		return fmt.Errorf("Error: did not read any headlines from the input file")
	}
	// (Re)build the headlineIndex
	e.out.headlineIndex = make(map[int]*Headline)
	for _, h := range e.out.Headlines {
		e.out.addHeadlineToIndex(h)
	}
	e.currentHeadlineID = e.out.Headlines[0].ID
	e.currentPosition = 0
	e.dirty = false
	e.sel = nil
	return nil
}

// Store a 'logical' line- this is a rendered line of text on the screen. We use this index
// to figure out where in the outline buffer to move to when we navigate visually
func (e *editor) recordLogicalLine(id int, bullet rune, indent int, hangingIndent int, position int, length int) {
	e.lineIndex = append(e.lineIndex, &line{id, bullet, indent, hangingIndent, position, length})
}

func (e *editor) moveRight(shiftPressed bool) {
	origPosition := e.currentPosition
	previousHeadlineID := e.currentHeadlineID
	if e.currentPosition < e.out.headlineIndex[e.currentHeadlineID].Buf.lastpos-1 { // are we within the text of current Headline?
		e.currentPosition++
	} else { // move to the first character of next Headline (if one exists and we are not selecting)
		if !shiftPressed {
			h := e.out.nextHeadline(e.currentHeadlineID, e)
			if h != nil {
				e.currentHeadlineID = h.ID
				e.currentPosition = 0
			} else { // no more Headlines
				return
			}
		} else { // selecting, but at end of Headline
			return
		}
	}
	// Make necessary updates to selection if necessary
	if shiftPressed {
		if !e.isSelecting() {
			e.sel = &selection{e.currentHeadlineID, origPosition, e.currentPosition}
		} else {
			// See if we are before or after the selection and update the beginning or end accordingly
			if e.currentPosition > e.sel.endPosition {
				e.sel.endPosition = e.currentPosition
			} else {
				e.sel.startPosition = e.currentPosition
			}
		}
	} else { // Cancel any existing selection
		e.sel = nil
	}
	// Do we need to scroll?
	newPtr := e.linePtr + 1
	if newPtr < len(e.lineIndex) { // we have additional logical lines beneath us
		if e.linePtr-e.topLine+1 >= e.editorHeight { // we are on last row of editor window
			if e.currentHeadlineID == previousHeadlineID { // we are on same Headline
				if e.currentPosition >= e.lineIndex[newPtr].position { // We've 'moved' to next logical line
					e.topLine++
				}
			} else { // We moved to a new headline, so is must be a new logical line
				e.topLine++
			}
		}
	}
}

func (e *editor) moveLeft(shiftPressed bool) {
	if e.currentPosition == 0 && e.linePtr == 0 { // Do nothing if on first character of first headline
		return
	} else {
		origPosition := e.currentPosition
		previousHeadlineID := e.currentHeadlineID
		if e.currentPosition > 0 { // Just move to previous character in this headline
			e.currentPosition--
		} else { // at first character of current headline, move to end of previous headline
			if !shiftPressed {
				p := e.out.previousHeadline(e.currentHeadlineID, e)
				if p != nil {
					e.currentHeadlineID = p.ID
					e.currentPosition = p.Buf.lastpos - 1
				}
			} else { // We are selecting, so do nothing
				return
			}
		}
		// Make necessary updates to selection if necessary
		if shiftPressed {
			if !e.isSelecting() {
				e.sel = &selection{e.currentHeadlineID, e.currentPosition, origPosition}
			} else {
				// Update the beginning or end of the selection depending on where cursor is relatively
				if e.currentPosition < e.sel.startPosition {
					e.sel.startPosition = e.currentPosition
				} else {
					e.sel.endPosition = e.currentPosition
				}
			}
		} else { // Cancel any existing selection
			e.sel = nil
		}
		// Do we need to scroll?
		newPtr := e.linePtr - 1
		if newPtr >= 0 {
			if e.linePtr-e.topLine+1 == 1 { // we are on first row of editor window
				if e.currentHeadlineID == previousHeadlineID { // we are on same Headline
					if e.currentPosition <= e.lineIndex[newPtr].position+e.lineIndex[newPtr].length { // We've 'moved' to previous logical line
						e.topLine--
					}
				} else { // We moved to a new headline, so is must be a new logical line
					e.topLine--
				}
			}
		}
	}
}

func (e *editor) moveDown() bool {
	if e.linePtr != len(e.lineIndex)-1 { // Make sure we're not on last line
		e.sel = nil
		offset := e.currentPosition - e.lineIndex[e.linePtr].position // how far 'in' are we on the logical line?
		newLinePtr := e.linePtr + 1
		if newLinePtr < len(e.lineIndex) { // There are more lines below us
			if offset >= e.lineIndex[newLinePtr].length { // Are we moving down to a smaller line with x too far right?
				e.currentPosition = e.lineIndex[newLinePtr].position + e.lineIndex[newLinePtr].length - 1
			} else {
				e.currentPosition = offset + e.lineIndex[newLinePtr].position
			}
			e.currentHeadlineID = e.lineIndex[newLinePtr].headlineID // pick up this logical line's headlineID just in case we move to a new Headline
		}
		// Scroll?
		if e.linePtr-e.topLine+1 >= e.editorHeight {
			e.topLine++
			return true
		}
	}
	return false
}

func (e *editor) selectDown() bool {
	if e.linePtr != len(e.lineIndex)-1 { // Make sure we're not on last line
		offset := e.currentPosition - e.lineIndex[e.linePtr].position // how far 'in' are we on the logical line?
		newLinePtr := e.linePtr + 1
		if !e.isSelecting() {
			e.sel = &selection{e.currentHeadlineID, e.currentPosition, 0}
		}
		if newLinePtr < len(e.lineIndex) { // There are more lines below us
			if e.sel.headlineID == e.lineIndex[newLinePtr].headlineID { // Make sure we're not moving to a new Headline
				if offset >= e.lineIndex[newLinePtr].length { // Are we moving down to a smaller line with x too far right?
					e.currentPosition = e.lineIndex[newLinePtr].position + e.lineIndex[newLinePtr].length - 1
				} else {
					e.currentPosition = offset + e.lineIndex[newLinePtr].position
				}
			} else { // moving down would put us on a new Headline, do nothing
				return false
			}
			e.sel.endPosition = e.currentPosition
		}
		// Scroll?
		if e.linePtr-e.topLine+1 >= e.editorHeight {
			e.topLine++
			return true
		}
	}
	return false
}

func (e *editor) moveUp() {
	if e.linePtr != 0 { // Do nothing if on first logical line
		offset := e.currentPosition - e.lineIndex[e.linePtr].position // how far 'in' are we on the logical line?
		newLinePtr := e.linePtr - 1
		e.sel = nil
		if newLinePtr >= 0 { // There are more lines above
			if offset >= e.lineIndex[newLinePtr].length { // Are we moving up to a smaller line with x too far right?
				e.currentPosition = e.lineIndex[newLinePtr].position + e.lineIndex[newLinePtr].length - 1
			} else {
				e.currentPosition = offset + e.lineIndex[newLinePtr].position
			}
			e.currentHeadlineID = e.lineIndex[newLinePtr].headlineID // pick up this logical line's headlineID just in case we move to a new Headline
		}
		// Scroll?
		if e.linePtr != 0 && e.linePtr-e.topLine+1 == 1 {
			e.topLine--
		}
	}
}

func (e *editor) selectUp() {
	if e.linePtr != 0 { // Do nothing if on first logical line
		offset := e.currentPosition - e.lineIndex[e.linePtr].position // how far 'in' are we on the logical line?
		newLinePtr := e.linePtr - 1
		if !e.isSelecting() {
			e.sel = &selection{e.currentHeadlineID, 0, e.currentPosition}
		}
		if newLinePtr >= 0 { // There are more lines above
			if e.sel.headlineID == e.lineIndex[newLinePtr].headlineID { // Make sure we're not moving to a new Headline
				if offset >= e.lineIndex[newLinePtr].length { // Are we moving up to a smaller line with x too far right?
					e.currentPosition = e.lineIndex[newLinePtr].position + e.lineIndex[newLinePtr].length - 1
				} else {
					e.currentPosition = offset + e.lineIndex[newLinePtr].position
				}
			} else { // moving down would put us on a new Headline, do nothing
				return
			}
			e.sel.startPosition = e.currentPosition
		}
		// Scroll?
		if e.linePtr != 0 && e.linePtr-e.topLine+1 == 1 {
			e.topLine--
		}
	}
}

func (e *editor) moveHome(shiftPressed bool) {
	origPosition := e.currentPosition
	e.currentPosition = 0
	if shiftPressed {
		if !e.isSelecting() {
			e.sel = &selection{e.currentHeadlineID, 0, origPosition}
		} else {
			e.sel.startPosition = 0
		}
	} else {
		e.sel = nil
	}
}

func (e *editor) moveEnd(shiftPressed bool) {
	origPosition := e.currentPosition
	e.currentPosition = e.out.headlineIndex[e.currentHeadlineID].Buf.lastpos - 1
	if shiftPressed {
		if !e.isSelecting() {
			e.sel = &selection{e.currentHeadlineID, origPosition, e.currentPosition - 1} // omit nodeDelim at end of Headline
		} else {
			e.sel.endPosition = e.currentPosition - 1
		}
	} else {
		e.sel = nil
	}
}

// =============== Editing Methods ================================

func (e *editor) insertRuneAtCurrentPosition(o *Outline, r rune) {
	h := o.currentHeadline(e)
	h.Buf.InsertRunes(e.currentPosition, []rune{r})
	e.moveRight(false)
}

// Remove the previous character.  Join this Headline to the previous Headline if on first character
func (e *editor) backspace(o *Outline) {
	if e.currentPosition == 0 && e.linePtr == 0 { // Do nothing if on first character of first headline
		return
	} else {
		currentHeadline := o.currentHeadline(e)
		if e.currentPosition > 0 { // Remove previous character
			posToRemove := e.currentPosition - 1
			currentHeadline.Buf.Delete(posToRemove, 1)
			e.moveLeft(false)
		} else { // Join this headline with previous one
			previousHeadline := o.previousHeadline(currentHeadline.ID, e)
			if previousHeadline != nil {
				// Add my text to the previous Headline, remove me from my parent's child list
				//  Don't bother removing the actual Headline itself from e.headlineIndex- in case we want to support undo
				e.currentPosition = previousHeadline.Buf.lastpos - 1
				previousHeadline.Buf.Delete(previousHeadline.Buf.lastpos-1, 1) // remove trailing nodeDelim
				previousHeadline.Buf.Append(currentHeadline.Buf.Text())
				// If I have children, add them as children of the previous Headline
				for i, c := range currentHeadline.Children {
					insertSibling(&previousHeadline.Children, i, c)
					c.ParentID = previousHeadline.ID
				}
				// Remove me from my parent and make previous Headline the current one
				_, children := o.childrenSliceFor(currentHeadline.ID)
				o.removeChildFrom(children, currentHeadline.ID)
				e.currentHeadlineID = previousHeadline.ID
			}
		}
	}
}

// delete the character underneath the cursor.  Join the next headline to this one if on last character of headline.
func (e *editor) delete(o *Outline) {
	currentHeadline := o.currentHeadline(e)
	if e.currentPosition != currentHeadline.Buf.lastpos-1 { // Just delete the current position
		currentHeadline.Buf.Delete(e.currentPosition, 1)
	} else { // Join the next Headline onto this one
		nextHeadline := o.nextHeadline(currentHeadline.ID, e)
		if nextHeadline != nil {
			// Add text from next Headline onto my own, add their children as mine
			currentHeadline.Buf.Delete(e.currentPosition, 1) // remove my trailing nodeDelim
			currentHeadline.Buf.Append(nextHeadline.Buf.Text())
			// If next Headline has children, make them my own
			for i, c := range nextHeadline.Children {
				insertSibling(&currentHeadline.Children, i, c)
				c.ParentID = currentHeadline.ID
			}
			// Remove next Headline from its parent
			_, children := o.childrenSliceFor(nextHeadline.ID)
			o.removeChildFrom(children, nextHeadline.ID)
		}
	}
}

/*
Enter always creates a new headline at current position at same level as current headline

Split the current Headline's text at the cursor point.  Put all text from cursor to end into a
new Headline that is the next sibling of current Headline.
*/
func (e *editor) enterPressed(o *Outline) {

	// "Split" current Headline at cursor position and create a new Headline with remaining text
	currentHeadline := o.currentHeadline(e)
	text := (*currentHeadline.Buf.Runes())
	newText := text[e.currentPosition : len(text)-1] // Extract remaining text (except trailing nodeDelim)
	currentHeadline.Buf.Delete(e.currentPosition, len(newText))
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

	// Update the o.HeadlinesIndex
	o.headlineIndex[newHeadline.ID] = newHeadline
	e.currentHeadlineID = newHeadline.ID
	e.currentPosition = 0

	// Scroll?
	if e.linePtr-e.topLine+1 >= e.editorHeight {
		e.topLine++
	}

}

//  Promote a Headline further down the outline one level
func (e *editor) tabPressed(o *Outline) {
	if e.linePtr != 0 {
		currentHeadline := o.currentHeadline(e)
		previousHeadline := o.previousHeadline(e.currentHeadlineID, e)
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

//  "Demote" a Headline back up the outline one level
func (e *editor) backTabPressed(o *Outline) {
	if e.linePtr != 0 {
		currentHeadline := o.currentHeadline(e)
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
func (e *editor) deleteHeadline(o *Outline) {
	h := o.currentHeadline(e)
	p := o.previousHeadline(h.ID, e)
	if e.linePtr != 0 && p != nil { // Make sure we're not removing first Headline and there are at least two in outline
		// Simply remove the Headline reference from our parent's children, keep in the index (so we can support Undo eventually)
		_, children := o.childrenSliceFor(h.ID)
		o.removeChildFrom(children, h.ID)
		e.currentHeadlineID = p.ID
		e.currentPosition = 0
	} else { // delete all text in first headline if it's the only one left
		h.Buf.Delete(0, h.Buf.lastpos-1)
		e.currentPosition = 0
	}
}

// copy the current Headline to the clipboard
func (e *editor) copyHeadline() {

}

// cut the current Headline and put in the clipboard
func (e *editor) cutHeadline() {

}

// copy the text of selection to the clipboard
func (e *editor) copySelection() {
	if e.isSelecting() {
		buf := []rune{}
		text := *(e.out.headlineIndex[e.currentHeadlineID].Buf.Runes())
		for c := e.sel.startPosition; c <= e.sel.endPosition; c++ {
			buf = append(buf, text[c])
		}
		e.selectionClipboard = &buf
	}
}

// cut the current selection from current position and put in clipboard
func (e *editor) cutSelection() {
	if e.isSelecting() {
		e.copySelection()
		// Remove the runes within the selection from current Headline
		// BUG: Sometimes this cuts the whole rest of the Headline...?  Also not setting e.currentPosition correctly
		span := e.sel.endPosition - e.sel.startPosition
		pieceTable := e.out.headlineIndex[e.currentHeadlineID].Buf
		pieceTable.Delete(e.sel.startPosition, span)
		if e.currentPosition == e.sel.endPosition {
			e.currentPosition -= span
		}
		e.sel = nil
	}
}

// paste the Headline in the clipboard as a child of current Headline
func (e *editor) pasteHeadline() {

}

// paste the selection in the clipboard at current cursor position in current Headline
func (e *editor) pasteSelection() {

}

// Edit the current outline's Title
func (e *editor) editOutlineTitle(s tcell.Screen, o *Outline) {
	newTitle := prompt(s, "Enter new title: ")
	if newTitle != "" {
		o.Title = newTitle
	}
}

// Collapse the current headline and all children
//  (this just marks each headline as invisible)
func (e *editor) collapse() {

}

// Expand the current headline (if necessary) and all children
//  (this just marks each headline as visible)
func (e *editor) expand() {

}

// Clear out the contents of the organizer's window
//  We depend on the caller to eventually do s.Show()
func (e *editor) clear(s tcell.Screen) {
	offset := organizerWidth + 2
	for y := 1; y < screenHeight-2; y++ {
		for x := offset; x < screenWidth-1; x++ {
			s.SetContent(x, y, ' ', nil, defStyle)
		}
	}
}

func (e *editor) draw(s tcell.Screen) {
	layoutOutline(s)
	e.clear(s)
	renderOutline(s)
	s.ShowCursor(cursX, cursY)
	s.Show()
}

func (e *editor) handleEvents(s tcell.Screen) {
	for {
		switch ev := s.PollEvent().(type) {
		case *tcell.EventResize:
			s.Sync()
			screenWidth, screenHeight = s.Size()
			drawScreen(s)
		case *tcell.EventKey:
			mod := ev.Modifiers()
			switch ev.Key() {
			case tcell.KeyDown:
				if mod == tcell.ModCtrl {
					e.expand()
				} else if mod == tcell.ModShift {
					e.selectDown()
				} else {
					e.moveDown()
				}
				e.draw(s)
			case tcell.KeyUp:
				if mod == tcell.ModCtrl {
					e.collapse()
				} else if mod == tcell.ModShift {
					e.selectUp()
				} else {
					e.moveUp()
				}
				e.draw(s)
			case tcell.KeyRight:
				e.moveRight(mod == tcell.ModShift)
				e.draw(s)
			case tcell.KeyLeft:
				e.moveLeft(mod == tcell.ModShift)
				e.draw(s)
			case tcell.KeyHome:
				e.moveHome(mod == tcell.ModShift)
				e.draw(s)
			case tcell.KeyEnd:
				e.moveEnd(mod == tcell.ModShift)
				e.draw(s)
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				e.dirty = true
				e.backspace(e.out)
				e.draw(s)
				drawTopBorder(s)
			case tcell.KeyDelete:
				e.dirty = true
				e.delete(e.out)
				e.draw(s)
				drawTopBorder(s)
			case tcell.KeyEnter:
				e.dirty = true
				e.enterPressed(e.out)
				e.draw(s)
				drawTopBorder(s)
			case tcell.KeyTab:
				e.tabPressed(e.out)
				e.draw(s)
			case tcell.KeyBacktab:
				e.dirty = true
				e.backTabPressed(e.out)
				e.draw(s)
				drawTopBorder(s)
			case tcell.KeyRune:
				e.dirty = true
				e.insertRuneAtCurrentPosition(e.out, ev.Rune())
				e.draw(s)
				drawTopBorder(s)
			case tcell.KeyCtrlC:
				if e.isSelecting() {
					e.copySelection()
				} else {
					e.copyHeadline()
				}
			case tcell.KeyCtrlD:
				e.deleteHeadline(e.out)
				e.draw(s)
			case tcell.KeyCtrlF: // for debugging
				e.out.dump(e)
			case tcell.KeyCtrlV:
				if e.isSelecting() {
					e.pasteSelection()
				} else {
					e.pasteHeadline()
				}
				e.dirty = true
				e.draw(s)
				drawTopBorder(s)
			case tcell.KeyCtrlX:
				if e.isSelecting() {
					e.cutSelection()
				} else {
					e.cutHeadline()
				}
				e.dirty = true
				e.draw(s)
				drawTopBorder(s)
			case tcell.KeyCtrlS:
				if currentFilename == "" {
					f := prompt(s, "Filename: ")
					if f != "" {
						currentFilename = f
						err := e.save(filepath.Join(org.currentDirectory, currentFilename))
						if err == nil {
							e.dirty = false
							setFileTitle(currentFilename)
						} else {
							msg := fmt.Sprintf("Error saving file: %v", err)
							prompt(s, msg)
						}
					}
				} else {
					e.dirty = false
					e.save(filepath.Join(org.currentDirectory, currentFilename))
				}
				org.refresh(s)
				drawScreen(s)
			case tcell.KeyCtrlT:
				e.editOutlineTitle(s, e.out)
				e.dirty = true
				drawTopBorder(s)
				e.draw(s)
			case tcell.KeyEscape:
				if e.isSelecting() { // Clear any selection
					e.sel = nil
					e.draw(s)
				}
				org.handleEvents(s, e.out)
				drawScreen(s)
			case tcell.KeyF1:
				showHelp(s)
				prompt(s, "")
				drawScreen(s)
			case tcell.KeyCtrlQ:
				proceed := true
				if e.dirty {
					// Prompt to save current outline first
					proceed = e.saveFirst(s)
				}
				if proceed {
					s.Fini()
					os.Exit(0)
				}
			}
		}
	}
}
