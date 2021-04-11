package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

type config map[string]string

var cfg *config

var currentFilename string

var fileTitle []rune

var screenWidth int
var screenHeight int

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
const small_bullet = '\u2022'
const solid_bullet = '\u25CF'
const open_bullet = '\u25CB'
const tri_bullet = '\u2023'
const dash_bullet = '\u2043'
const shear_bullet = '\u25B0'
const box_bullet = '\u25A0'

var defStyle tcell.Style
var borderStyle tcell.Style
var buttonStyle tcell.Style
var fileStyle tcell.Style
var dirStyle tcell.Style
var selectedStyle tcell.Style

var org *organizer
var ed *editor

const defaultConfigFilename = "gv.conf"

var storageDirectory string

//go:embed help.txt
var helptext string

func drawFrame(s tcell.Screen, x, y, width, height int) {
	// Corners
	s.SetContent(x, y, tlcorner, nil, defStyle)
	s.SetContent(x+width-1, y, trcorner, nil, defStyle)
	s.SetContent(x, y+height-1, llcorner, nil, defStyle)
	s.SetContent(x+width-1, y+height-1, lrcorner, nil, defStyle)
	// Horizontal
	for bx := 0; bx < width-2; bx++ {
		s.SetContent(bx+x+1, y, hline, nil, defStyle)
		s.SetContent(bx+x+1, y+height-1, hline, nil, defStyle)
	}
	writeString(s, x+2, y, "[Help]")
	// Vertical
	for by := y + 1; by < y+height-1; by++ {
		s.SetContent(x, by, vline, nil, defStyle)
		for bx := 1; bx < width-1; bx++ {
			s.SetContent(x+bx, by, ' ', nil, defStyle)
		}
		s.SetContent(x+width-1, by, vline, nil, defStyle)
	}
}

func writeString(s tcell.Screen, x int, y int, text string) {
	for c, r := range []rune(text) {
		s.SetContent(x+c, y, r, nil, borderStyle)
	}
}

func showHelp(s tcell.Screen) {
	lines := strings.Split(helptext, "\n")
	width := 0
	for _, l := range lines {
		if len(l) > width {
			width = len(l)
		}
	}
	width += 4
	height := len(lines) + 2
	x := (screenWidth - width) / 2
	y := (screenHeight - height) / 2
	drawFrame(s, x, y, width, height)
	for c, l := range lines {
		writeString(s, x+2, y+c+1, l)
	}

}

func drawBorder(s tcell.Screen, x, y, width, height int) {
	// Corners
	s.SetContent(x, y, tlcorner, nil, borderStyle)
	s.SetContent(x+width-1, y, trcorner, nil, borderStyle)
	s.SetContent(x, y+height-1, llcorner, nil, borderStyle)
	s.SetContent(x+width-1, y+height-1, lrcorner, nil, borderStyle)
	tb := renderTopBorder()
	bb := renderBottomBorder(width - 2)
	if len(*tb) != len(*bb) {
		fmt.Printf("Border Error! Screen is %d and org is %d and TB is %d and BB is %d\n",
			screenWidth, org.width, len(*tb), len(*bb))
		os.Exit(1)
	}
	// Horizontal
	for bx := 0; bx < len(*tb); bx++ {
		s.SetContent(bx+x+1, y, (*tb)[bx], nil, borderStyle)
		s.SetContent(bx+x+1, y+height-1, (*bb)[bx], nil, borderStyle)
	}
	// Vertical
	for by := y + 1; by < y+height-1; by++ {
		s.SetContent(x, by, vline, nil, borderStyle)
		s.SetContent(org.width+1, by, vline, nil, borderStyle)
		s.SetContent(x+width-1, by, vline, nil, borderStyle)
	}
}

// Render and re-draw the top border
func drawTopBorder(s tcell.Screen) {
	tb := renderTopBorder()
	for bx := 1; bx < len(*tb); bx++ {
		s.SetContent(bx+1, 0, (*tb)[bx], nil, borderStyle)
	}
	s.Show()
}

func renderTopBorder() *[]rune {
	var row []rune
	maxTitleWidth := int(float64(ed.editorWidth) * 0.8) // Set maximum title size so we don't run over

	// Organizer
	foldername := []rune(filepath.Base(org.currentDirectory)) // TODO: Ensure this is < org.width-3
	if len(foldername) > org.width-3 {
		foldername = foldername[:org.width-4]
		foldername = append(foldername, ellipsis)
	}
	row = append(row, hline)
	row = append(row, '[')
	row = append(row, foldername...)
	row = append(row, ']')
	for p := len(row); p < org.width; p++ {
		row = append(row, hline)
	}
	row = append(row, tdown)

	// Editor
	titleRunes := []rune(ed.out.Title)
	if len(titleRunes) > maxTitleWidth { // Is title too long?  Do we need to truncate & add ellipsis?
		titleRunes = titleRunes[:maxTitleWidth-1]
		titleRunes = append(titleRunes, ellipsis)
	}
	row = append(row, hline)
	row = append(row, '[')
	row = append(row, titleRunes...)
	if ed.dirty {
		row = append(row, '*')
	}
	row = append(row, ']')
	for p := len(row); p < screenWidth-2; p++ {
		row = append(row, hline)
	}

	return &row
}

func renderBottomBorder(width int) *[]rune {
	var row []rune
	for p := 1; p < org.width+1; p++ {
		row = append(row, hline)
	}
	row = append(row, tup)
	for p := len(row); p < screenWidth-2; p++ {
		row = append(row, hline)
	}
	return &row
}

func layoutOutline(s tcell.Screen) {
	y := 1

	// clear out lineIndex (avoid a re-allocation)
	for l := range ed.lineIndex {
		ed.lineIndex[l] = nil
	}
	ed.lineIndex = ed.lineIndex[:0]

	// Layout each Headline
	for _, h := range ed.out.Headlines {
		y = layoutHeadline(s, ed, ed.out, h, 1, y)
	}
}

// Format headline text according to indent and word-wrap.  Layout all of its children.
func layoutHeadline(s tcell.Screen, e *editor, o *Outline, h *Headline, level int, y int) int {
	var bullet rune
	endY := y
	indent := e.org.width + (level * 3)
	var hangingIndent int
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
			endY = layoutHeadline(s, e, o, h, level+1, endY)
		}
	}

	return endY
}

// Walk thru the lineIndex and render each logical line that is within the window's boundaries
func renderOutline(s tcell.Screen) {
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

func genTestOutline(s tcell.Screen, e *editor) *Outline {
	o := newOutline("Sample")
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

func drawScreen(s tcell.Screen) {
	ed.setScreenSize(s)
	s.Clear()
	drawBorder(s, 0, 0, screenWidth, screenHeight-1)
	org.draw(s)
	layoutOutline(s)
	renderOutline(s)
	s.ShowCursor(cursX, cursY) // TODO: This should only be done if editor is in focus
	s.Show()
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
func prompt(s tcell.Screen, msg string) string {
	var response []rune
	var cursX int = len(msg) + 1
	for {
		renderPrompt(s, cursX, msg, string(response))
		switch ev := s.PollEvent().(type) {
		case *tcell.EventResize:
			s.Sync()
			screenWidth, screenHeight = s.Size()
			drawScreen(s)
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

// Confirm that storage is set up; default to $HOME/.gv/outlines or use $GVHOME
//  return the base application directory and location of outline files
func setupStorage() (string, string, error) {
	dir, found := os.LookupEnv("GVHOME")
	if !found {
		var err error
		dir, err = os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		dir = filepath.Join(dir, "/.gv")
	}
	storageDirectory := filepath.Join(dir, "/outlines")
	err := os.MkdirAll(storageDirectory, 0700)
	if err != nil {
		return "", "", err
	}
	return dir, storageDirectory, nil
}

// Try to load the configuration.  If it does not exist, first initialize it
func loadConfig(dir string) (*config, error) {
	filePath := filepath.Join(dir, defaultConfigFilename)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		err = saveConfig(filePath, initConfig())
		if err != nil {
			return nil, err
		}
	}
	buf, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	// Extract the config JSON
	var c config
	err = json.Unmarshal(buf, &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func initConfig() *config {
	c := config{
		"backgroundColor":  "black",
		"borderColor":      "white",
		"defaultTextColor": "powderblue",
		"linkColor":        "blue",
		"listColor":        "yellow",
	}
	return &c
}

func saveConfig(filePath string, c *config) error {
	buf, err := json.MarshalIndent(c, "", "   ")
	if err != nil {
		return err
	}
	ioutil.WriteFile(filePath, buf, 0644)
	return nil
}

func colorFor(name string) tcell.Color {
	color, found := tcell.ColorNames[(*cfg)[name]]
	if !found {
		return tcell.ColorWhite
	} else {
		return color
	}
}

func setColors() {
	defStyle = tcell.StyleDefault.
		Background(colorFor("backgroundColor")).
		Foreground(colorFor("defaultTextColor"))

	borderStyle = tcell.StyleDefault.
		Background(colorFor("backgroundColor")).
		Foreground(colorFor("borderColor"))

	buttonStyle = tcell.StyleDefault.
		Background(colorFor("backgroundColor")).
		Foreground(colorFor("borderColor"))

	fileStyle = tcell.StyleDefault.
		Background(colorFor("backgroundColor")).
		Foreground(colorFor("listColor"))

	dirStyle = tcell.StyleDefault.
		Background(colorFor("backgroundColor")).
		Foreground(colorFor("linkColor")).
		Underline(true)

	selectedStyle = tcell.StyleDefault.
		Background(colorFor("defaultTextColor")).
		Foreground(colorFor("backgroundColor"))
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

	directory, storageDirectory, err := setupStorage()
	if err != nil {
		s.Fini()
		fmt.Printf("Unable to set up storage: %v\n", err)
		os.Exit(1)
	}

	// Load the application config
	cfg, err = loadConfig(directory)
	if err != nil {
		s.Fini()
		fmt.Printf("Error trying to load config from %s\n", directory)
		os.Exit(1)
	}

	setColors()

	screenWidth, screenHeight = s.Size()

	s.SetStyle(defStyle)

	_, height := s.Size()

	org = newOrganizer(directory, storageDirectory, height)
	org.refresh(s)
	ed = newEditor(s, org)

	drawScreen(s)

	ed.handleEvents(s)

}
