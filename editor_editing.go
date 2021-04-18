package main

import (
	"github.com/gdamore/tcell/v2"
)

/*

Editing methods for the editor.


*/

func (e *editor) setDirty(s tcell.Screen, dirty bool) {
	e.dirty = dirty
	drawTopBorder(s)
}

func (e *editor) insertRuneAtCurrentPosition(o *Outline, r rune) {
	h := o.currentHeadline(e)
	h.Buf.InsertRunes(e.currentPosition, []rune{r})
	e.moveRight(false)
}

// Remove the previous character.  Join this Headline to the previous Headline if on first character
// BUG: backspace on first line, but have scrolled, no bullet, navigation panics
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
				e.moveUp()
				//e.currentHeadlineID = previousHeadline.ID
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
	currentHeadline := o.currentHeadline(e)

	// If the Headlne has children and is collapsed, just move cursor down to next line instead, we can't add children now
	if !currentHeadline.Expanded && len(currentHeadline.Children) != 0 {
		e.moveEnd(false)
		e.moveDown()
		return
	}

	// "Split" current Headline at cursor position and create a new Headline with remaining text
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
