package main

import (
	"fmt"
	"os"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

var currentFilename string

var fileTitle []rune

// Standard line drawing characters
const tlcorner = '\u250c'
const trcorner = '\u2510'
const llcorner = '\u2514'
const lrcorner = '\u2518'
const hline = '\u2500'
const vline = '\u2502'
const tdown = '\u252c'
const tup = '\u2534'

const dirtyFlag = '*'

const ellipsis = '\u2026'

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

var defStyle tcell.Style
var fileStyle tcell.Style
var dirStyle tcell.Style

var org *organizer
var editorX int = 15 // Column at which editor starts

func drawBorder(s tcell.Screen, x, y, width, height int) {
	// Corners
	s.SetContent(x, y, tlcorner, nil, defStyle)
	s.SetContent(x+width-1, y, trcorner, nil, defStyle)
	s.SetContent(x, y+height-1, llcorner, nil, defStyle)
	s.SetContent(x+width-1, y+height-1, lrcorner, nil, defStyle)
	tb := renderTopBorder(width - 2)
	bb := renderBottomBorder(width - 2)
	// Horizontal
	for bx := 0; bx < len(*tb); bx++ {
		s.SetContent(bx+x+1, y, (*tb)[bx], nil, defStyle)
		s.SetContent(bx+x+1, y+height-1, (*bb)[bx], nil, defStyle)
	}
	// Vertical
	for by := y + 1; by < y+height-1; by++ {
		s.SetContent(x, by, vline, nil, defStyle)
		s.SetContent(editorX, by, vline, nil, defStyle)
		s.SetContent(x+width-1, by, vline, nil, defStyle)
	}
}

func renderTopBorder(width int) *[]rune {
	var row []rune
	titlePos := (width - len(fileTitle)) - 3
	for p := 0; p < titlePos; p++ {
		if p == editorX-1 {
			row = append(row, tdown)
		} else {
			row = append(row, hline)
		}
	}
	row = append(row, fileTitle...)
	row = append(row, hline)
	if dirty {
		row = append(row, dirtyFlag)
	} else {
		row = append(row, hline)
	}
	row = append(row, hline)
	return &row
}

func renderBottomBorder(width int) *[]rune {
	var row []rune
	for p := 0; p < width; p++ {
		if p == editorX-1 {
			row = append(row, tup)
		} else {
			row = append(row, hline)
		}
	}
	return &row
}

func setFileTitle(filename string) {
	fileTitle = append(fileTitle, []rune("Filename: ")...)
	fileTitle = append(fileTitle, []rune(filename)...)
}

func layoutOutline(s tcell.Screen, e *editor, o *outline) {
	y := 1
	e.lineIndex = []line{}
	for _, h := range o.headlines {
		y = layoutHeadline(s, e, o, h, 1, y)
	}
}

// Format headline text according to indent and word-wrap.  Layout all of its children.
func layoutHeadline(s tcell.Screen, e *editor, o *outline, h *Headline, level int, y int) int {
	var bullet rune
	endY := y
	if h.Expanded {
		bullet = vtriangle
	} else {
		bullet = solid_bullet
	}
	indent := editorX + (level * 3)
	hangingIndent := indent + 3
	text := h.Buf.Runes()
	pos := 0
	end := len(*text)
	firstLine := true
	for pos < end {
		endPos := pos + e.editorWidth
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

	for _, h := range h.Children {
		endY = layoutHeadline(s, e, o, h, level+1, endY)
	}

	return endY
}

// Walk thru the lineIndex and render each logical line that is within the window's boundaries
func renderOutline(s tcell.Screen, e *editor, o *outline) {
	y := 1
	lastLine := e.topLine + e.editorHeight - 1
	for l := e.topLine; l <= lastLine && l < len(e.lineIndex); l++ {
		x := 0
		line := e.lineIndex[l]
		runes := (*o.headlineIndex[line.headlineID].Buf.Runes())
		s.SetContent(x+line.indent, y, line.bullet, nil, defStyle)
		for p := line.position; p < line.position+line.length; p++ {
			// If we're rendering the current position, place cursor here, remember this is current logical line
			if line.headlineID == e.currentHeadlineID && e.currentPosition == p {
				cursX = line.hangingIndent + x
				cursY = y
				e.linePtr = l
			}
			s.SetContent(x+line.hangingIndent, y, runes[p], nil, defStyle)
			x++
		}
		y++
	}
}

func genTestOutline(s tcell.Screen, e *editor) *outline {
	o := newOutline(s, "Sample")
	o.addHeadline("What is this odd beast GrandView?", -1)                                                                                                                                                                                                                                                                                                             // 1
	o.addHeadline("In a single-pane outliner, all the components of your outline and its accompanying information are visible in one window.", 1)                                                                                                                                                                                                                      // 2
	o.addHeadline("Project and task manager", 2)                                                                                                                                                                                                                                                                                                                       // 3
	o.addHeadline("Information manager", 2)                                                                                                                                                                                                                                                                                                                            // 4
	o.addHeadline("Here's a headline that has children hidden", -1)                                                                                                                                                                                                                                                                                                    // 5
	o.addHeadline("Here's a headline that has no children", -1)                                                                                                                                                                                                                                                                                                        // 6
	o.addHeadline("What makes GrandView so unique even today?  How is it possible that such a product like this could exist?", 6)                                                                                                                                                                                                                                      // 7
	o.addHeadline("Multiple Views", 7)                                                                                                                                                                                                                                                                                                                                 // 8
	o.addHeadline("ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890", 7)                                                                                                                             // 9
	o.addHeadline("Outline View", 7)                                                                                                                                                                                                                                                                                                                                   // 10
	o.addHeadline("You can associate any headline (node) with a document. Document view is essentially a hoist that removes all the other elements of your outline from the screen so you can focus on writing the one document. When you are done writing this document (or section of your outline), you can return to outline view, where your document text.", 10) // 11
	o.addHeadline("Category & Calendar Views", 7)                                                                                                                                                                                                                                                                                                                      // 12
	o.addHeadline("Way over the top.", 2)                                                                                                                                                                                                                                                                                                                              // 13
	o.addHeadline("ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVW XYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ123456 7890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890", 13)                                                                                                                          // 14
	o.addHeadline("Fully customizable meta-data", 14)
	e.currentHeadlineID = 1
	e.currentPosition = 0
	return o
}

func drawScreen(s tcell.Screen, e *editor, o *outline) {
	width, height := s.Size()
	s.Clear()
	drawBorder(s, 0, 0, width, height-1)
	drawOrganizer(s, height)
	e.setScreenSize(s)
	layoutOutline(s, e, o)
	renderOutline(s, e, o)
	s.ShowCursor(cursX, cursY)
	s.Show()
}

func drawOrganizer(s tcell.Screen, height int) {
	org.height = height - 2
	y := 1
	for c := org.topLine; c < len(org.entries); c++ {
		if y < org.height {
			style := fileStyle
			if org.entries[c].isDir {
				style = dirStyle
			}
			for x, r := range org.entries[c].name {
				if x < org.width {
					s.SetContent(1+x, y, r, nil, style)
				}
			}
		}
		y++
	}
}

func clearPrompt(s tcell.Screen) {
	width, height := s.Size()
	// Clear the row
	for x := 0; x < width; x++ {
		s.SetContent(x, height-1, ' ', nil, defStyle)
	}
}

func renderPrompt(s tcell.Screen, cx int, msg string, response string) {
	width, height := s.Size()
	y := height - 1
	var x, x2 int
	var r rune
	clearPrompt(s)
	// Brackets
	s.SetContent(0, y, '[', nil, defStyle)
	s.SetContent(width-1, y, ']', nil, defStyle)
	// Write the content
	for x, r = range msg {
		s.SetContent(1+x, y, r, nil, defStyle)
	}
	for x2, r = range response {
		s.SetContent(2+x+x2, y, r, nil, defStyle)
	}
	s.ShowCursor(cx, y)
	s.Show()
}

// Prompt the user for some input- blocking main event loop
func prompt(s tcell.Screen, e *editor, o *outline, msg string) string {
	var response []rune
	var cursX int = len(msg) + 1
	for {
		renderPrompt(s, cursX, msg, string(response))
		switch ev := s.PollEvent().(type) {
		case *tcell.EventResize:
			s.Sync()
			drawScreen(s, e, o)
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyRune:
				response = append(response, ev.Rune())
				cursX++
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if len(response) > 0 {
					response = response[:len(response)-1]
					cursX--
				}
			case tcell.KeyEnter:
				clearPrompt(s)
				return string(response)
			case tcell.KeyEscape:
				clearPrompt(s)
				return ""
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
		Foreground(tcell.ColorPowderBlue)

	fileStyle = tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorGreen)

	dirStyle = tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorBlue)

	s.SetStyle(defStyle)

	_, height := s.Size()

	o := newOutline(s, "Example")
	org = newOrganizer(height)
	org.refresh()
	ed := newEditor()
	if len(os.Args) > 1 {
		currentFilename = os.Args[1]
		err := o.load(currentFilename, ed)
		if err == nil {
			setFileTitle(currentFilename)
		} else {
			msg := fmt.Sprintf("Error opening file: %v", err)
			prompt(s, ed, o, msg)
		}
	} else {
		err := o.init(ed)
		if err != nil {
			msg := fmt.Sprintf("Error initalizing outline: %v", err)
			prompt(s, ed, o, msg)
		}
	}

	drawScreen(s, ed, o)

	ed.handleEvents(s, o)

}
