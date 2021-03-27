package main

import (
	"fmt"
	"io/ioutil"
)

/*

Outline is the heart of the application.  Provides all necessary methods to manage and manipulate Headlines as part of
an overall outline structure.  We also hold onto the headlineIndex which is the master 'database' of all Headlines keyed
by ID.  The headlineIndex makes it fast to find any Headline that has been created without having to traverse the outline
each time.
*/

type Outline struct {
	Title         string            // describes an outline in the Organizer
	Headlines     []*Headline       // list of top level headlines (this denotes the structure of the outline)
	headlineIndex map[int]*Headline // index to all Headlines (keyed by ID- this makes serialization easier than using pointers)
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

const nodeDelim = '\ufeff'

const emptyHeadlineText = string(nodeDelim) // every Headline's text ends with a nonprinting rune so we can append to it easily

var dbg int
var dbg2 int

func newOutline(title string) *Outline {
	o := &Outline{title, []*Headline{}, make(map[int]*Headline)}
	return o
}

// initialize a new outline to be used as a blank outline for editing
func (o *Outline) init(e *editor) error {
	id, _ := o.addHeadline("", -1)
	e.currentHeadlineID = id
	e.currentPosition = 0
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

func (o *Outline) dump(e *editor) {
	text := (*o.headlineIndex[e.currentHeadlineID].Buf.Runes())
	out := "Headline and children\n"
	//i, c := o.childrenSliceFor(13)
	//for _, h := range o.Headlines {
	//	out += h.toString(0) + "\n"
	//}
	out += fmt.Sprintf("\nscreen width %d, org width %d, editor width %d\n", screenWidth, e.org.width, e.editorWidth)
	out += fmt.Sprintf("\nlinePtr %d, currentHeadline %d, currentPosition %d, current Rune (%#U) num Headlines %d, dbg %d, dbg2 %d\n",
		e.linePtr, e.currentHeadlineID, e.currentPosition, text[e.currentPosition], len(o.headlineIndex), dbg, dbg2)
	ioutil.WriteFile("dump.txt", []byte(out), 0644)
}

// Add a Headline (and all of its children) into the o.headlineIndex
func (o *Outline) addHeadlineToIndex(h *Headline) {
	o.headlineIndex[h.ID] = h
	for _, c := range h.Children {
		o.addHeadlineToIndex(c)
	}
}

func (o *Outline) newHeadline(text string, parent int) *Headline {
	id := nextHeadlineID(o.headlineIndex)
	return &Headline{id, parent, true, *NewPieceTable(text + emptyHeadlineText), []*Headline{}} // Note we're adding extra non-printing char to end of text
}

// appends a new headline onto the outline under the parent
func (o *Outline) addHeadline(text string, parent int) (int, error) {
	h := o.newHeadline(text, parent)
	if parent == -1 { // Is this a top-level headline?
		o.Headlines = append(o.Headlines, h)
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

// Return the IDs of the Headlines just before and after the Headline at given ID.  Return -1 for either if at beginning or end of outline.
// We leverage the fact that e.lineIndex is really a 'flattened' DFS list of Headlines, so it has the ordered list of Headlines
//   (ideally we would not need a reference to the editor here, but it saves us from having to maintain our own structure)
func (o *Outline) prevNextFrom(ID int, e *editor) (int, int) {
	previous := -1
	next := -1
	// generate list of ordered, unique headline IDs from e.lineIndex
	var headlines []int
	for _, l := range e.lineIndex {
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

// Get the 'parent' slice where the Headline with ID is held.  Also return ID's index in that slice.
func (o *Outline) childrenSliceFor(ID int) (int, *[]*Headline) {
	index := -1
	var children *[]*Headline
	h := o.headlineIndex[ID]
	if h != nil {
		if h.ParentID == -1 {
			children = &o.Headlines
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
func (o *Outline) nextHeadline(ID int, e *editor) *Headline {
	if ID == -1 {
		return nil
	}
	_, n := o.prevNextFrom(ID, e)
	if n == -1 {
		return nil
	}
	return o.headlineIndex[n]
}

// Find the "previous" Headline befoire the Headline with ID.  Return nil if no more Headlines are prior.
func (o *Outline) previousHeadline(ID int, e *editor) *Headline {
	if ID == -1 {
		return nil
	}
	p, _ := o.prevNextFrom(ID, e)
	if p == -1 {
		return nil
	}
	return o.headlineIndex[p]
}

// Find the "current" Headline
func (o *Outline) currentHeadline(e *editor) *Headline {
	return o.headlineIndex[e.currentHeadlineID]
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
func (o *Outline) removeChildFrom(children *[]*Headline, childID int) {
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
