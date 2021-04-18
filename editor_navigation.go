package main

/*

Navigation methods for the editor.

*/

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

func (e *editor) pageDown() {
	if (e.topLine + e.editorHeight) < len(e.lineIndex) { // Make sure we have at least a "page" beneath us
		e.linePtr += e.editorHeight
		if e.linePtr >= len(e.lineIndex) {
			e.linePtr = len(e.lineIndex) - 1
		}
		e.currentPosition = e.lineIndex[e.linePtr].position
		e.currentHeadlineID = e.lineIndex[e.linePtr].headlineID
		e.topLine += e.editorHeight
		if e.topLine >= len(e.lineIndex) {
			e.topLine = len(e.lineIndex) - e.editorHeight
		}
	}
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

func (e *editor) pageUp() {
	e.linePtr -= e.editorHeight
	if e.linePtr < 0 {
		e.linePtr = 0
	}
	e.currentPosition = e.lineIndex[e.linePtr].position
	e.currentHeadlineID = e.lineIndex[e.linePtr].headlineID
	e.topLine -= e.editorHeight
	if e.topLine < 0 {
		e.topLine = 0
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
