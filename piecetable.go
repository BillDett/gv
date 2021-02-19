package main

import (
	"fmt"
	"strings"
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
	// save in the add buffer and create the necessary piece instance
	fragrunes := []rune(fragment)
	start := len(p.add)
	length := len(fragrunes)
	p.add = append(p.add, fragrunes...)
	newadd := piece{&(p.add), start, length}

	fmt.Printf("Inserting at position %d\n", position)

	if position == 0 {
		// insert newadd to front of pieces list
		p.pieces = append([]piece{newadd}, p.pieces...)
	} else if position == p.lastpos {
		// append newadd to end of pieces list
		// update p.lastpos accordingly
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

// Delete removes length characters starting at position
func (p *PieceTable) Delete(position int, length int) {
	/* If delete is contained within a single piece:
		if delete is not at beginning or end of list:
		  Split the piece and adjust the lengths "around" the deleted span
		else:
		  Adjust the start/length accordingly to remove from beginning or end of piece
	  else:
	    For each piece in the span:
		  If first piece, adjust the length to just before where span starts
		  If last piece, adjust the start to just after the span ends
		  otherwise, remove the piece
	*/
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
	if i == 0 {
		// This is the first piece- just adjust the length
	}
}

// Text returns the string being managed by the PieceTable with all edits applied
func (p *PieceTable) Text() string {
	var sb strings.Builder
	// Walk thru the pieces and reconstruct the string
	for _, piece := range p.pieces {
		span := (*piece.source)[piece.start : piece.start+piece.length]
		sb.WriteString(string(span))
	}
	return sb.String()
}

func insertPiece(slice []piece, newpiece piece, index int) []piece {
	s := append(slice, piece{})  // Making space for the new element
	copy(s[index+1:], s[index:]) // Shifting elements
	s[index] = newpiece          // Copying/inserting the value
	return s
}

func main() {
	pt := NewPieceTable("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	fmt.Println(pt.Text())

	pt.Dump()

	pt.Insert(0, "FOO")
	fmt.Println(pt.Text())

	pt.Dump()

	pt.Insert(16, "BAR")
	fmt.Println(pt.Text())

	pt.Dump()

	pt.Insert(9, "HI")

	pt.Dump()

	fmt.Println(pt.Text())

	//pt.Dump()
}
