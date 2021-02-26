package main

import (
	"fmt"
	"os"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

var defStyle tcell.Style

var editorWidth, editorHeight int

// Standard line drawing characters
const tlcorner = '\u250c'
const trcorner = '\u2510'
const llcorner = '\u2514'
const lrcorner = '\u2518'
const hline = '\u2500'
const vline = '\u2502'

const htriangle = '\u25B6'
const small_htriangle = '\u25B8'
const vtriangle = '\u25BC'
const small_vtriangle = '\u25BE'
const solid_bullet = '\u25CF'
const open_bullet = '\u25CB'
const tri_bullet = '\u2023'
const dash_bullet = '\u2043'
const shear_bullet = '\u25B0'
const box_bullet = '\u25A0'

func drawBorder(s tcell.Screen, x, y, width, height int) {
	// Corners
	s.SetContent(x, y, tlcorner, nil, defStyle)
	s.SetContent(x+width-1, y, trcorner, nil, defStyle)
	s.SetContent(x, y+height-1, llcorner, nil, defStyle)
	s.SetContent(x+width-1, y+height-1, lrcorner, nil, defStyle)
	// Horizontal
	for bx := x + 1; bx < x+width-1; bx++ {
		s.SetContent(bx, y, hline, nil, defStyle)
		s.SetContent(bx, y+height-1, hline, nil, defStyle)
	}
	// Vertical
	for by := y + 1; by < y+height-1; by++ {
		s.SetContent(x, by, vline, nil, defStyle)
		s.SetContent(x+width-1, by, vline, nil, defStyle)
	}
}

func writeString(s tcell.Screen, word string, x int, y int) {
	for n, c := range word {
		s.SetContent(x+n, y, c, nil, defStyle)
	}
}

// Write a headline into the window, format according to indent and word-wrap the text.
func headline(s tcell.Screen, text string, y int, level int, collapsed bool) int {
	origX := 1
	var bullet rune
	endY := y
	if collapsed {
		bullet = solid_bullet
	} else {
		bullet = vtriangle
	}
	indent := origX + (level * 3)
	hangingIndent := indent + 3
	s.SetContent(indent, y, bullet, nil, defStyle)
	if len(text) <= editorWidth-hangingIndent { // headline fits entirely within a single line
		writeString(s, text, hangingIndent, endY)
		endY++
	} else { // going to have to wrap it
		pos := 0
		for pos < len(text) {
			endPos := pos + editorWidth - hangingIndent
			if endPos > len(text)-1 { // overshot end of text, we're on the last fragment
				writeString(s, text[pos:len(text)-1], hangingIndent, endY)
				endY++
			} else { // on first or middle fragment
				if !unicode.IsSpace(rune(text[endPos])) {
					// Walk backwards until you see your first whitespace
					p := endPos
					for p > pos && !unicode.IsSpace(rune(text[p])) {
						p--
					}
					if p != pos { // split at the space (hitting pos means beginning of text or last starting point)
						endPos = p + 1
					}
				}
				writeString(s, text[pos:endPos], hangingIndent, endY)
				endY++
			}
			pos = endPos
		}
	}

	return endY
}

func drawEditorText(s tcell.Screen) {
	y := headline(s, "What is GrandView?", 2, 1, false)
	y = headline(s, "In a single-pane outliner, all the components of your outline and its accompanying information are visible in one window.", y, 2, false)
	y = headline(s, "Project and task manager ", y, 3, true)
	y = headline(s, "Information manager ", y, 3, true)
	y = headline(s, "Here's a headline that has children hidden", y, 1, true)
	y = headline(s, "Here's a headline that has no children", y, 1, true)
	y = headline(s, "What makes GrandView so unique even today?  How is it possible that such a product like this could exist?", y, 1, false)
	y = headline(s, "Multiple Views", y, 2, false)
	y = headline(s, "ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890", y, 3, true)
	y = headline(s, "Outline View", y, 3, true)
	y = headline(s, "You can associate any headline (node) with a document. Document view is essentially a hoist that removes all the other elements of your outline from the screen so you can focus on writing the one document. When you are done writing this document (or section of your outline), you can return to outline view, where your document text.", y, 3, true)
	y = headline(s, "Category & Calendar Views", y, 3, true)
	y = headline(s, "Way over the top.", y, 4, true)
	y = headline(s, "ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVW XYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ123456 7890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890", y, 4, true)
	y = headline(s, "Fully customizable meta-data", y, 3, true)
}

func drawScreen(s tcell.Screen) {
	width, height := s.Size()
	editorWidth = int(float64(width) * 0.7)
	editorHeight = height
	s.Clear()
	drawBorder(s, 0, 0, width, height)
	drawEditorText(s)
	s.Show()
}

func handleEvents(s tcell.Screen) {
	for {
		switch ev := s.PollEvent().(type) {
		case *tcell.EventResize:
			s.Sync()
			drawScreen(s)
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape:
				s.Fini()
				os.Exit(0)
			}
		}
	}
}

func main() {

	s, e := tcell.NewScreen()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	if e := s.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}

	defStyle = tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorGreen)
	s.SetStyle(defStyle)

	drawScreen(s)

	handleEvents(s)

}
