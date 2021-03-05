package main

import (
	"fmt"
	"os"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

var currentFilename string

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

func layoutOutline(s tcell.Screen, o *outline, width int, height int) {
	y := 1
	o.lineIndex = []line{}
	text := o.buf.Runes()
	var start, end int
	var delim *delimiter
	var err error
	delim, start, end = o.nextHeadline(start)
	//	for end < len(*text)-2 && delim != nil { // Scan thru all headlines (stop before you hit EOL Delim)
	for delim != nil { // Scan thru all headlines (stop before you hit EOL Delim)

		if err != nil {
			fmt.Printf("%v\n", err)
			break
		}
		y = layoutHeadline(s, o, text, start, end+1, y, delim.level, false) // we use end+1 so we render the <nodeDelim>- this gives us something at end of headline to start typing on when appending text to headline
		delim, start, end = o.nextHeadline(end)
	}
}

// Format headline according to indent and word-wrap the text.
func layoutHeadline(s tcell.Screen, o *outline, text *[]rune, start int, end int, y int, level int, collapsed bool) int {
	origX := 1
	level++
	var bullet rune
	endY := y
	if collapsed {
		bullet = solid_bullet
	} else {
		bullet = vtriangle
	}
	indent := origX + (level * 3)
	hangingIndent := indent + 3
	headlineLength := end - start + 1
	if headlineLength <= o.editorWidth-hangingIndent { // headline fits entirely within a single line
		o.recordLogicalLine(bullet, indent, hangingIndent, start, headlineLength-1)
		endY++
	} else { // going to have to wrap it
		pos := start
		firstLine := true
		for pos < end {
			endPos := pos + o.editorWidth
			if endPos > end { // overshot end of text, we're on the last fragment
				o.recordLogicalLine(0, indent, hangingIndent, pos, end-pos)
				endPos = end
				endY++
			} else { // on first or middle fragment
				var mybullet rune
				if firstLine { // if we're laying out first line of a multi-line headline, remember that we want to use a bullet
					mybullet = bullet
					firstLine = false
				}
				if !unicode.IsSpace((*text)[endPos]) {
					// Walk backwards until you see your first whitespace
					p := endPos
					for p > pos && !unicode.IsSpace((*text)[p]) {
						p--
					}
					if p != pos { // split at the space (hitting pos means beginning of text or last starting point)
						endPos = p + 1
					}
				}
				o.recordLogicalLine(mybullet, indent, hangingIndent, pos, endPos-pos)
				endY++
			}
			pos = endPos
		}
	}

	return endY
}

// Walk thru the lineIndex and render each logical line that is within the window's boundaries
func renderOutline(s tcell.Screen, o *outline) {
	runes := *(o.buf.Runes())
	y := 1
	lastLine := o.topLine + o.editorHeight - 1
	for l := o.topLine; l <= lastLine && l < len(o.lineIndex); l++ {
		x := 0
		line := o.lineIndex[l]
		s.SetContent(x+line.indent, y, line.bullet, nil, defStyle)
		for p := line.position; p < line.position+line.length; p++ {
			// If we're rendering the current position, place cursor here, remember this is current logical line
			if o.currentPosition == p {
				cursX = line.hangingIndent + x
				cursY = y
				o.linePtr = l
			}
			s.SetContent(x+line.hangingIndent, y, runes[p], nil, defStyle)
			x++
		}
		y++
	}
}

func genTestOutline(s tcell.Screen) *outline {
	o := newOutline(s)
	o.addHeadline("What is this odd beast GrandView?", 0)
	o.addHeadline("In a single-pane outliner, all the components of your outline and its accompanying information are visible in one window.", 1)
	o.addHeadline("Project and task manager", 2)
	o.addHeadline("Information manager", 2)
	o.addHeadline("Here's a headline that has children hidden", 0)
	o.addHeadline("Here's a headline that has no children", 0)
	o.addHeadline("What makes GrandView so unique even today?  How is it possible that such a product like this could exist?", 0)
	o.addHeadline("Multiple Views", 1)
	o.addHeadline("ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890", 1)
	o.addHeadline("Outline View", 1)
	o.addHeadline("You can associate any headline (node) with a document. Document view is essentially a hoist that removes all the other elements of your outline from the screen so you can focus on writing the one document. When you are done writing this document (or section of your outline), you can return to outline view, where your document text.", 1)
	o.addHeadline("Category & Calendar Views", 1)
	o.addHeadline("Way over the top.", 2)
	o.addHeadline("ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVW XYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890ABCEFGHIJKLMNOPQRSTUVWXYZ123456 7890ABCEFGHIJKLMNOPQRSTUVWXYZ1234567890", 2)
	o.addHeadline("Fully customizable meta-data", 1)
	return o
}

func drawScreen(s tcell.Screen, o *outline) {
	width, height := s.Size()
	s.Clear()
	drawBorder(s, 0, 0, width, height)
	layoutOutline(s, o, width, height)
	renderOutline(s, o)
	s.ShowCursor(cursX, cursY)
	s.Show()
}

func handleEvents(s tcell.Screen, o *outline) {
	for {
		switch ev := s.PollEvent().(type) {
		case *tcell.EventResize:
			s.Sync()
			drawScreen(s, o)
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape:
				s.Fini()
				os.Exit(0)
			case tcell.KeyDown:
				o.moveDown()
				drawScreen(s, o)
			case tcell.KeyUp:
				o.moveUp()
				drawScreen(s, o)
			case tcell.KeyRight:
				o.moveRight()
				drawScreen(s, o)
			case tcell.KeyLeft:
				o.moveLeft()
				drawScreen(s, o)
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				o.backspace()
				drawScreen(s, o)
			case tcell.KeyDelete:
				o.delete()
				drawScreen(s, o)
			case tcell.KeyEnter:
				o.enterPressed()
				drawScreen(s, o)
			case tcell.KeyTab:
				o.tabPressed(true)
				drawScreen(s, o)
			case tcell.KeyBacktab:
				o.tabPressed(false)
				drawScreen(s, o)
			case tcell.KeyRune:
				o.insertRuneAtCurrentPosition(ev.Rune())
				drawScreen(s, o)
			case tcell.KeyCtrlF: // for debugging
				o.dump()
			case tcell.KeyCtrlS:
				if currentFilename == "" {
					// TODO: Should prompt for a filename
					currentFilename = "outline.gv"
				}
				o.save(currentFilename)
			}
		}
	}
}

func main() {

	//s, _ := tcell.NewScreen()
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

	//o := genTestOutline(s)

	o := newOutline(s)
	if len(os.Args) > 1 {
		currentFilename = os.Args[1]
		o.load(currentFilename)
	} else {
		o.init()
	}

	//o.test()

	drawScreen(s, o)

	handleEvents(s, o)

}
