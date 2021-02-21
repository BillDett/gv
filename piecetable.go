package main

import (
	"fmt"
)

/*
	Piece Table implementation for the text editor- allows for efficient insert/delete activity on a sequence of runes
*/

type piece struct {
	source *[]rune // should point to original or add slices in piecetable
	start  int
	length int // Sum of all piece lengths == length of final edited text
}

// PieceTable manages efficient edits to a string of text
type PieceTable struct {
	original []rune
	add      []rune
	pieces   []piece
	lastpos  int
}

// NewPieceTable creates a piecetable instance
func NewPieceTable(orig string) *PieceTable {
	origrunes := []rune(orig)
	length := len(origrunes)
	pt := PieceTable{
		origrunes,
		[]rune{},
		[]piece{},
		length,
	}
	p := piece{&(pt.original), 0, len(pt.original)}
	pt.pieces = append(pt.pieces, p)
	return &pt
}

// Dump generates a debug view of the PieceTable for troubleshooting
func (p *PieceTable) Dump() {
	fmt.Printf("Original buffer:\n\t(%p), %s\n", &p.original, string(p.original))
	fmt.Printf("Add buffer:\n\t(%p) %s\nPieces:", &p.add, string(p.add))
	fmt.Printf("Lastpos: %d\n", p.lastpos)
	fmt.Printf("Pieces:\n\tSource\t\t\tStart\tLength\n\t------\t\t\t-----\t------\n")
	for _, piece := range p.pieces {
		fmt.Printf("\t%p\t\t\t%d\t%d\n", piece.source, piece.start, piece.length)
	}
	fmt.Println()
}

// Insert puts fragment into the string at given position
func (p *PieceTable) Insert(position int, fragment string) {
	fragrunes := []rune(fragment)
	p.InsertRunes(position, fragrunes)
}

// InsertRunes puts a slice of runes into the string at given position
func (p *PieceTable) InsertRunes(position int, runes []rune) {
	// save in the add buffer and create the necessary piece instance
	start := len(p.add)
	length := len(runes)
	p.add = append(p.add, runes...)
	newadd := piece{&(p.add), start, length}

	//fmt.Printf("Inserting at position %d\n", position)

	if position == 0 {
		// insert newadd to front of pieces list
		p.pieces = append([]piece{newadd}, p.pieces...)
	} else if position == p.lastpos {
		// append newadd to end of pieces list
		p.pieces = append(p.pieces, newadd)
	} else {
		// We have to look for the right piece now to split so we can insert mid-string
		totalLength := 0
		i := 0
		for i < len(p.pieces) {
			totalLength += p.pieces[i].length
			if totalLength >= position {
				break
			}
			i++
		}
		// We're on the piece that needs to be split
		newRemainder := (totalLength - position)      // What is length of the remainder after we split this piece?
		p.pieces[i].length -= newRemainder            // "shrink" this piece where we split it
		p.pieces = insertPiece(p.pieces, newadd, i+1) // Insert a piece for the thing we're inserting after split
		p.pieces = insertPiece(p.pieces,
			piece{p.pieces[i].source, p.pieces[i].start + p.pieces[i].length, newRemainder}, i+2) // Insert new piece (which we split from p.pieces[i]) for remainder
	}

	p.lastpos += length

}

// AppendRune will add a single rune to the end of the PieceTable
func (p *PieceTable) AppendRune(r rune) {

	p.InsertRunes(p.lastpos, []rune{r})
}

// Append will add the characters to the end of the PieceTable
func (p *PieceTable) Append(fragment string) {
	p.Insert(p.lastpos, fragment)
}

// Delete removes length characters starting at position
func (p *PieceTable) Delete(position int, spanLength int) {
	//fmt.Printf("Deleting at position %d for %d runes\n", position, spanLength)
	// First step is to figure out which piece begins the span
	totalLength := 0
	i := 0
	for i < len(p.pieces) {
		totalLength += p.pieces[i].length
		if totalLength > position {
			break
		}
		i++
	}

	// Now process the delete depending on how many pieces the span crosses
	newRemainder := (totalLength - position)
	//fmt.Printf("Found span start in piece %d, TL=%d, NR=%d\n", i, totalLength, newRemainder)
	if p.pieces[i].length >= spanLength {
		//fmt.Printf("Delete entirely within this piece #%d\n", i)
		if totalLength-p.pieces[i].length == position {
			// span starts at beginning if piece, just adjust start and length
			//fmt.Printf("Delete from beginning of text\n")
			p.pieces[i].start += spanLength
			p.pieces[i].length -= spanLength
		} else if position+spanLength == totalLength {
			// span ends at end off piece, just adjust length
			//fmt.Printf("Delete from end of text\n")
			p.pieces[i].length -= spanLength
		} else {
			// span is in middle of piece, split the piece 'around' it
			//fmt.Printf("Delete from middle of text\n")
			origLength := p.pieces[i].length
			p.pieces[i].length -= totalLength - position // 'truncate' leftmost piece before span
			rmStart := p.pieces[i].start + p.pieces[i].length + spanLength
			rmLength := origLength - p.pieces[i].length - spanLength
			//fmt.Printf("Rightmost start is %d, length is %d\n", rmStart, rmLength)
			p.pieces = insertPiece(p.pieces, piece{p.pieces[i].source, rmStart, rmLength}, i+1) // rightmost section of original piece after the span
		}
	} else {
		//fmt.Printf("Delete spans multiple pieces\n")
		spanRemaining := spanLength - newRemainder
		p.pieces[i].length -= newRemainder // Shorten the leftmost piece
		for spanRemaining > 0 {            // 'delete' the remaining pieces if necessary
			//fmt.Printf("Span remaining: %d\n", spanRemaining)
			i++
			//fmt.Printf("\tNow scanning pieces- on %d\n", i)
			if p.pieces[i].length >= spanRemaining {
				// Last piece for the deleted span, just adjust accordingly
				//fmt.Printf("\tDeleting from piece %d\n", i)
				p.pieces[i].start += spanRemaining
				p.pieces[i].length -= spanRemaining
				spanRemaining = 0
			} else {
				spanRemaining -= p.pieces[i].length
				// "Remove" this piece by setting length to 0
				p.pieces[i].length = 0
				//fmt.Printf("\tRemoving piece at %d\n", i)
			}
		}
	}
	p.lastpos -= spanLength
}

// Text returns the string being managed by the PieceTable with all edits applied
func (p *PieceTable) Text() string {
	runes := p.Runes()
	return string(*runes)
}

// Runes returns the runes being managed by the PieceTable with all edits applied
func (p *PieceTable) Runes() *[]rune {
	var runes []rune
	for _, piece := range p.pieces {
		if piece.length != 0 {
			span := (*piece.source)[piece.start : piece.start+piece.length]
			runes = append(runes, span...)
		}
	}
	return &runes
}

func insertPiece(slice []piece, newpiece piece, index int) []piece {
	s := append(slice, piece{})  // Making space for the new element
	copy(s[index+1:], s[index:]) // Shifting elements
	s[index] = newpiece          // Copying/inserting the value
	return s
}
