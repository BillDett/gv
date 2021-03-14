package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
)

/*

Editor is the main part of the screen, responsible for editing the outline

*/

type editor struct {
	o                 *outline // Current outline being edited
	lineIndex         []line   // Text Position index for each "line" after editor has been laid out.
	linePtr           int      // index of the line currently beneath the cursor
	editorWidth       int      // width of an editor column
	editorHeight      int      // height of the editor window
	currentHeadlineID int      // ID of headline cursor is on
	currentPosition   int      // the current position within the currentHeadline.Buf
	topLine           int      // index of the topmost "line" of the window in lineIndex
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

var currentLine int // which "line" we are currently on
var cursX int       // X coordinate of the cursor
var cursY int       // Y coordinate of the cursor

// TODO: MAKE THIS A MEMBER OF editor ITSELF
var dirty bool = false // Is the outliine buffer modified since last save?

func (e *editor) setScreenSize(s tcell.Screen) {
	var width int
	width, height := s.Size()
	e.editorWidth = int(float64(width) * 0.7)
	e.editorHeight = height - 3 // 2 rows for border, 1 row for interaction
}

func newEditor() *editor {
	return &editor{nil, nil, 0, 0, 0, 0, 0, 0}
}

// Store a 'logical' line- this is a rendered line of text on the screen. We use this index
// to figure out where in the outline buffer to move to when we navigate visually
func (e *editor) recordLogicalLine(id int, bullet rune, indent int, hangingIndent int, position int, length int) {
	e.lineIndex = append(e.lineIndex, line{id, bullet, indent, hangingIndent, position, length})
}

func (e *editor) moveRight(o *outline) {
	previousHeadlineID := e.currentHeadlineID
	if e.currentPosition < o.headlineIndex[e.currentHeadlineID].Buf.lastpos-1 { // are we within the text of current Headline?
		e.currentPosition++
	} else { // move to the first character of next Headline (if one exists)
		h := o.nextHeadline(e.currentHeadlineID, e)
		if h != nil {
			e.currentHeadlineID = h.ID
			e.currentPosition = 0
		} else { // no more Headlines
			return
		}
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

func (e *editor) moveLeft(o *outline) {
	if e.currentPosition == 0 && e.linePtr == 0 { // Do nothing if on first character of first headline
		return
	} else {
		previousHeadlineID := e.currentHeadlineID
		if e.currentPosition > 0 { // Just move to previous character in this headline
			e.currentPosition--
		} else { // at first character of current headline, move to end of previous headline
			p := o.previousHeadline(e.currentHeadlineID, e)
			if p != nil {
				e.currentHeadlineID = p.ID
				e.currentPosition = p.Buf.lastpos - 1
			}
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

func (e *editor) moveDown(o *outline) {
	if e.linePtr != len(e.lineIndex)-1 { // Make sure we're not on last line
		offset := e.currentPosition - e.lineIndex[e.linePtr].position // how far 'in' are we on the logical line?
		dbg = offset
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
		}
	}
}

func (e *editor) moveUp(o *outline) {
	if e.linePtr != 0 { // Do nothing if on first logical line
		offset := e.currentPosition - e.lineIndex[e.linePtr].position // how far 'in' are we on the logical line?
		newLinePtr := e.linePtr - 1
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

// =============== Editing Methods ================================

func (e *editor) insertRuneAtCurrentPosition(o *outline, r rune) {
	h := o.currentHeadline(e)
	h.Buf.InsertRunes(e.currentPosition, []rune{r})
	e.moveRight(o)
}

// Remove the previous character.  Join this Headline to the previous Headline if on first character
func (e *editor) backspace(o *outline) {
	if e.currentPosition == 0 && e.linePtr == 0 { // Do nothing if on first character of first headline
		return
	} else {
		currentHeadline := o.currentHeadline(e)
		if e.currentPosition > 0 { // Remove previous character
			posToRemove := e.currentPosition - 1
			currentHeadline.Buf.Delete(posToRemove, 1)
			e.moveLeft(o)
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
func (e *editor) delete(o *outline) {
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
func (e *editor) enterPressed(o *outline) {

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

	// Update the o.headlinesIndex
	o.headlineIndex[newHeadline.ID] = newHeadline
	e.currentHeadlineID = newHeadline.ID
	e.currentPosition = 0

	// Scroll?
	if e.linePtr-e.topLine+1 >= e.editorHeight {
		e.topLine++
	}

}

//  Promote a Headline further down the outline one level
func (e *editor) tabPressed(o *outline) {
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
func (e *editor) backTabPressed(o *outline) {
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
func (e *editor) deleteHeadline(o *outline) {
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

// Collapse the current headline and all children
//  (this just marks each headline as invisible)
func (e *editor) collapse() {

}

// Expand the current headline (if necessary) and all children
//  (this just marks each headline as visible)
func (e *editor) expand() {

}

func (e *editor) handleEvents(s tcell.Screen, o *outline) {
	for {
		switch ev := s.PollEvent().(type) {
		case *tcell.EventResize:
			s.Sync()
			drawScreen(s, e, o)
		case *tcell.EventKey:
			mod := ev.Modifiers()
			//fmt.Printf("EventKey Modifiers: %d Key: %d Rune: %v", mod, key, ch)
			switch ev.Key() {
			case tcell.KeyDown:
				if mod == tcell.ModCtrl {
					e.expand()
				} else {
					e.moveDown(o)
				}
				drawScreen(s, e, o)
			case tcell.KeyUp:
				if mod == tcell.ModCtrl {
					e.collapse()
				} else {
					e.moveUp(o)
				}
				drawScreen(s, e, o)
			case tcell.KeyRight:
				e.moveRight(o)
				drawScreen(s, e, o)
			case tcell.KeyLeft:
				e.moveLeft(o)
				drawScreen(s, e, o)
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				dirty = true
				e.backspace(o)
				drawScreen(s, e, o)
			case tcell.KeyDelete:
				dirty = true
				e.delete(o)
				drawScreen(s, e, o)
			case tcell.KeyEnter:
				dirty = true
				e.enterPressed(o)
				drawScreen(s, e, o)
			case tcell.KeyTab:
				e.tabPressed(o)
				drawScreen(s, e, o)
			case tcell.KeyBacktab:
				e.backTabPressed(o)
				drawScreen(s, e, o)
			case tcell.KeyRune:
				dirty = true
				e.insertRuneAtCurrentPosition(o, ev.Rune())
				drawScreen(s, e, o)
			case tcell.KeyCtrlD:
				e.deleteHeadline(o)
				drawScreen(s, e, o)
			case tcell.KeyCtrlF: // for debugging
				o.dump(e)
			case tcell.KeyCtrlS:
				if currentFilename == "" {
					f := prompt(s, e, o, "Filename: ")
					if f != "" {
						currentFilename = f
						err := o.save(currentFilename)
						if err == nil {
							dirty = false
							setFileTitle(currentFilename)
						} else {
							msg := fmt.Sprintf("Error saving file: %v", err)
							prompt(s, e, o, msg)
						}
					}
				} else {
					dirty = false
					o.save(currentFilename)
				}
				drawScreen(s, e, o)
			case tcell.KeyCtrlQ:
				save := prompt(s, e, o, "Outline modified, save [Y|N]? ")
				if save == "" {
					clearPrompt(s)
					drawScreen(s, e, o)
					break
				}
				if strings.ToUpper(save) != "N" {
					if currentFilename == "" {
						f := prompt(s, e, o, "Filename: ")
						if f != "" {
							o.save(f)
						} else { // skipped setting filename, cancel quit request
							clearPrompt(s)
							drawScreen(s, e, o)
							break
						}
					} else {
						o.save(currentFilename)
						drawScreen(s, e, o)
					}
				}
				s.Fini()
				os.Exit(0)
			}
		}
	}
}
