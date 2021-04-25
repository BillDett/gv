package main

import (
	_ "embed"
	_ "net/http/pprof"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

/*

Layout and rendering methods for the editor.

*/

func (e *editor) layoutOutline(s tcell.Screen) {
	y := 1

	// clear out lineIndex (avoid a re-allocation)
	for l := range ed.lineIndex {
		ed.lineIndex[l] = nil
	}
	ed.lineIndex = ed.lineIndex[:0]

	// Layout each Headline
	for _, h := range ed.out.Headlines {
		y = e.layoutHeadline(s, h, 1, y)
	}
}

// Format headline text according to indent and word-wrap.  Layout all of its children.
func (e *editor) layoutHeadline(s tcell.Screen, h *Headline, level int, y int) int {
	var bullet rune
	o := e.out
	endY := y
	indent := e.org.width + (level * 3)
	var hangingIndent int
	if e.out.MultiList && level == 1 { // For multi-list, we don't render a bullet for top level headlines
		bullet = ' '
		hangingIndent = indent
	} else {
		switch o.Bullets {
		case glyphBullet:
			if len(h.Children) != 0 {
				if h.Expanded {
					bullet = small_vtriangle
				} else {
					bullet = small_htriangle
				}
			} else {
				bullet = small_bullet
			}
			hangingIndent = indent + 3
		case noBullet:
			bullet = ' '
			hangingIndent = indent
		}
	}
	text := h.Buf.Runes()
	pos := 0
	end := len(*text)
	firstLine := true
	for pos < end {
		endPos := pos + e.editorWidth - (level * 3) - 2
		if endPos > end { // overshot end of text, we're on the first or last fragment
			var mybullet rune
			if firstLine { // if we're laying out first line less than editor width, remember that we want to use a bullet
				mybullet = bullet
				firstLine = false
			}
			e.recordLogicalLine(h.ID, mybullet, indent, hangingIndent, pos, end-pos)
			endPos = end
			endY++
		} else { // on first or middle fragment
			var mybullet rune
			if firstLine { // if we're laying out first line of a multi-line headline, remember that we want to use a bullet
				mybullet = bullet
				firstLine = false
			}
			if endPos < len(*text) && !unicode.IsSpace((*text)[endPos]) {
				// Walk backwards until you see your first whitespace
				p := endPos
				for p > pos && !unicode.IsSpace((*text)[p]) {
					p--
				}
				if p != pos { // split at the space (hitting pos means beginning of text or last starting point)
					endPos = p + 1
				}
			}
			e.recordLogicalLine(h.ID, mybullet, indent, hangingIndent, pos, endPos-pos)
			endY++
		}
		pos = endPos
	}

	// Unless headline is collapsed, render its children
	if h.Expanded {
		for _, h := range h.Children {
			endY = e.layoutHeadline(s, h, level+1, endY)
		}
	}

	return endY
}

// Walk thru the lineIndex and render each logical line that is within the window's boundaries
func (e *editor) renderOutline(s tcell.Screen) {
	y := 1
	lastLine := ed.topLine + ed.editorHeight - 1
	for l := ed.topLine; l <= lastLine && l < len(ed.lineIndex); l++ {
		x := 0
		line := ed.lineIndex[l]
		h := ed.out.headlineIndex[line.headlineID]
		runes := (*h.Buf.Runes())
		s.SetContent(x+line.indent, y, line.bullet, nil, defStyle)
		for p := line.position; p < line.position+line.length; p++ {
			// If we're rendering the current position, place cursor here, remember this is current logical line
			if line.headlineID == ed.currentHeadlineID && ed.currentPosition == p {
				cursX = line.hangingIndent + x
				cursY = y
				ed.linePtr = l
			}
			// Set the style depending on whether we're selecting or not
			theStyle := defStyle
			if ed.isSelecting() && line.headlineID == ed.sel.headlineID && p >= ed.sel.startPosition && p <= ed.sel.endPosition {
				theStyle = selectedStyle
			}
			s.SetContent(x+line.hangingIndent, y, runes[p], nil, theStyle)
			x++
		}
		y++
	}
}
