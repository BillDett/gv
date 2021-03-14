package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/gdamore/tcell/v2"
)

/*

We have conflated the concepts of the outline and the editor into a single struct...really need to break that
apart so we are saving just the outline to disk, but passing the editor around as well.

*/

type outline struct {
	title         string            // describes an outline in the Organizer
	headlines     []*Headline       // list of top level headlines (this denotes the structure of the outline)
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

func newOutline(s tcell.Screen, title string) *outline {
	o := &outline{title, []*Headline{}, make(map[int]*Headline)}
	return o
}

// initialize a new outline to be used as a blank outline for editing
func (o *outline) init(e *editor) error {
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

func (o *outline) dump(e *editor) {
	text := (*o.headlineIndex[e.currentHeadlineID].Buf.Runes())
	out := "Headline and children\n"
	//i, c := o.childrenSliceFor(13)
	for _, h := range o.headlines {
		out += h.toString(0) + "\n"
	}
	out += fmt.Sprintf("\nlinePtr %d, currentHeadline %d, currentPosition %d, current Rune (%#U) num Headlines %d, dbg %d, dbg2 %d\n",
		e.linePtr, e.currentHeadlineID, e.currentPosition, text[e.currentPosition], len(o.headlineIndex), dbg, dbg2)
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
func (o *outline) load(filename string, e *editor) error {
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
	e.currentHeadlineID = o.headlines[0].ID
	e.currentPosition = 0
	return nil
}

// Add a Headline (and all of its children) into the o.headlineIndex
func (o *outline) addHeadlineToIndex(h *Headline) {
	o.headlineIndex[h.ID] = h
	for _, c := range h.Children {
		o.addHeadlineToIndex(c)
	}
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

// Return the IDs of the Headlines just before and after the Headline at given ID.  Return -1 for either if at beginning or end of outline.
// We leverage the fact that o.lineIndex is really a 'flattened' DFS list of Headlines, so it has the ordered list of Headlines
//  TODO: THIS FORCES A COUPLING BETWEEN THE editor AND THE outline THAT IS ARTIFICIAL...
func (o *outline) prevNextFrom(ID int, e *editor) (int, int) {
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
func (o *outline) nextHeadline(ID int, e *editor) *Headline {
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
func (o *outline) previousHeadline(ID int, e *editor) *Headline {
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
func (o *outline) currentHeadline(e *editor) *Headline {
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
