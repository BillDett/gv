package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
)

/*

Editor is the main part of the screen, responsible for editing the outline.  It also contains the lineIndex
which acts like a 'render buffer' of logical lines of text that have been laid out within the boundary of
the editor's window.


*/

type editor struct {
	org                *organizer // pointer to the organizer
	out                *Outline   // current outline being edited
	lineIndex          []*line    // Text Position index for each "line" after editor has been laid out.
	linePtr            int        // index of the line currently beneath the cursor
	editorWidth        int        // width of an editor column
	editorHeight       int        // height of the editor window
	currentHeadlineID  int        // ID of headline cursor is on
	currentPosition    int        // the current position within the currentHeadline.Buf
	topLine            int        // index of the topmost "line" of the window in lineIndex
	dirty              bool       // Is the outliine buffer modified since last save?
	sel                *selection // pointer to the current selection (nil means we are not selecting any text)
	headlineClipboard  *Headline  // pointer to the currently copied/cut Headline (nil if nothing being copied/cut)
	selectionClipboard *[]rune    // pointer to a slice of runes containing copied/cut selecton text (nil if nothing copied/cut)
}

// a line is a logical representation of a line that is rendered in the window
type line struct {
	headlineID    int  // Which headline's text are we representing?
	bullet        rune // What bullet should precede this line (if any)?
	indent        int  // Initial indent before a bullet
	hangingIndent int  // Indent for text without a bullet
	position      int  // Text position in o.lineIndex[headlineID].Buf.Runes()
	length        int  // How many runes in this "line"
}

// A selection indicates the start and end positions of contiguous Headline text that is selected
// We do not support selecting text *across* Headlines as that is really difficult to do right
type selection struct {
	headlineID    int // ID of headline where selection occurs
	startPosition int // outline buf position where selection starts
	endPosition   int // outline buf position where selection occurs
}

var currentLine int // which "line" we are currently on
var cursX int       // X coordinate of the cursor
var cursY int       // Y coordinate of the cursor

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func (e *editor) setScreenSize(s tcell.Screen) {
	//var width int
	//width, height := s.Size()
	e.editorWidth = screenWidth - e.org.width - 3
	e.editorHeight = screenHeight - 3 // 2 rows for border, 1 row for interaction
}

func newEditor(s tcell.Screen, org *organizer) *editor {
	ed := &editor{org, nil, nil, 0, 0, 0, 0, 0, 0, false, nil, nil, nil}
	lastOutlineFilePath, found := cfg[lastOpenedOutlineCfgKey]
	if found {
		ed.open(s, lastOutlineFilePath)
	} else {
		ed.newOutline(s, "New Outline")
	}
	return ed
}

func (e *editor) isSelecting() bool { return e.sel != nil }

// save the outline buffer to a file
func (e *editor) save(filename string) error {
	buf, err := json.Marshal(e.out)
	if err != nil {
		return err
	}
	ioutil.WriteFile(filename, buf, 0644)
	return nil
}

// User is about to change editor contents or quit, see if they want to save current editor first.
func (e *editor) saveFirst(s tcell.Screen) bool {
	response := prompt(s, "Save first (Y/N)?")
	if response != "" {
		if strings.ToUpper(response) == "Y" {
			if currentFilename != "" {
				e.save(filepath.Join(org.currentDirectory, currentFilename))
			} else {
				f := prompt(s, "Filename: ")
				if f != "" {
					currentFilename = f
					err := e.save(filepath.Join(org.currentDirectory, currentFilename))
					if err == nil {
						e.dirty = false
						org.refresh(s)
					} else {
						msg := fmt.Sprintf("Error saving file: %v", err)
						prompt(s, msg)
					}
				}
			}
		}
		return true
	}
	return false
}

// store this filePath as last opened Outline
func (e *editor) rememberOutline(filePath string) {
	cfg[lastOpenedOutlineCfgKey] = filePath
	saveConfig()
}

// user wants to create a new outline, save an existing, dirty one first
func (e *editor) newOutline(s tcell.Screen, title string) error {
	proceed := true
	if e.dirty {
		// Prompt to save current outline first
		proceed = e.saveFirst(s)
	}
	if proceed {
		if title == "" {
			title = prompt(s, "Enter new outline title:")
		}
		if title != "" {
			e.out = newOutline(title)
			e.out.init(e)
			e.linePtr = 0
			e.topLine = 0
			e.dirty = true
			currentFilename = e.generateFilename()
			e.sel = nil
			filePath := filepath.Join(org.currentDirectory, currentFilename)
			e.save(filePath)
			e.rememberOutline(filePath)
		}
	}
	return nil
}

func (e *editor) generateFilename() string {
	filename := strings.ToLower(e.out.Title)
	filename = strings.Replace(filename, " ", "_", -1)
	maxLen := 10 // Cap prefix at this length
	if maxLen > len(e.out.Title) {
		maxLen = len(e.out.Title)
	}
	filename = filename[0:maxLen]
	filename = fmt.Sprintf("%s%s.gv", filename, randSeq(5))
	return filename
}

func randSeq(n int) string {
	b := make([]rune, n)
	rand.Seed(time.Now().Unix())
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// user wants to open this outline, save an existing, dirty one first
func (e *editor) open(s tcell.Screen, filePath string) error {
	proceed := true
	if e.dirty {
		// Prompt to save current outline first
		proceed = e.saveFirst(s)
	}
	if proceed {
		err := e.load(filePath)
		if err == nil {
			currentFilename = filepath.Base(filePath)
		} else {
			msg := fmt.Sprintf("Error opening file: %v", err)
			prompt(s, msg)
		}
		e.rememberOutline(filePath)
	}
	return nil
}

// load a .gv file and use it to populate the outline's buffer
func (e *editor) load(filename string) error {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	// Extract the outline JSON
	e.out = nil
	err = json.Unmarshal(buf, &(e.out))
	if err != nil {
		return err
	}
	if len(e.out.Headlines) == 0 {
		return fmt.Errorf("Error: did not read any headlines from the input file")
	}
	// (Re)build the headlineIndex
	e.out.headlineIndex = make(map[int]*Headline)
	for _, h := range e.out.Headlines {
		e.out.addHeadlineToIndex(h)
	}
	e.currentHeadlineID = e.out.Headlines[0].ID
	e.currentPosition = 0
	e.linePtr = 0
	e.topLine = 0
	e.dirty = false
	e.sel = nil
	return nil
}

// Store a 'logical' line- this is a rendered line of text on the screen. We use this index
// to figure out where in the outline buffer to move to when we navigate visually
func (e *editor) recordLogicalLine(id int, bullet rune, indent int, hangingIndent int, position int, length int) {
	e.lineIndex = append(e.lineIndex, &line{id, bullet, indent, hangingIndent, position, length})
}

// Clear out the contents of the organizer's window
//  We depend on the caller to eventually do s.Show()
func (e *editor) clear(s tcell.Screen) {
	offset := e.org.width + 2
	for y := 1; y < screenHeight-2; y++ {
		for x := offset; x < screenWidth-1; x++ {
			s.SetContent(x, y, ' ', nil, defStyle)
		}
	}
}

func (e *editor) draw(s tcell.Screen) {
	layoutOutline(s)
	e.clear(s)
	renderOutline(s)
	s.ShowCursor(cursX, cursY)
	s.Show()
}

func (e *editor) handleEvents(s tcell.Screen) {
	for {
		switch ev := s.PollEvent().(type) {
		case *tcell.EventResize:
			s.Sync()
			screenWidth, screenHeight = s.Size()
			drawScreen(s)
		case *tcell.EventKey:
			mod := ev.Modifiers()
			switch ev.Key() {
			case tcell.KeyDown:
				if mod == tcell.ModCtrl {
					e.out.currentHeadline(e).Expanded = true
					e.setDirty(s, true)
				} else if mod == tcell.ModShift {
					e.selectDown()
				} else {
					e.moveDown()
				}
				e.draw(s)
			case tcell.KeyUp:
				if mod == tcell.ModCtrl {
					e.out.currentHeadline(e).Expanded = false
					e.setDirty(s, true)
				} else if mod == tcell.ModShift {
					e.selectUp()
				} else {
					e.moveUp()
				}
				e.draw(s)
			case tcell.KeyPgUp:
				e.pageUp()
				e.draw(s)
			case tcell.KeyPgDn:
				e.pageDown()
				e.draw(s)
			case tcell.KeyRight:
				e.moveRight(mod == tcell.ModShift)
				e.draw(s)
			case tcell.KeyLeft:
				e.moveLeft(mod == tcell.ModShift)
				e.draw(s)
			case tcell.KeyHome:
				e.moveHome(mod == tcell.ModShift)
				e.draw(s)
			case tcell.KeyEnd:
				e.moveEnd(mod == tcell.ModShift)
				e.draw(s)
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				e.backspace(e.out)
				e.draw(s)
				e.setDirty(s, true)
			case tcell.KeyDelete:
				if mod == tcell.ModCtrl {
					e.deleteHeadline(e.out)
				} else {
					e.delete(e.out)
				}
				e.draw(s)
				e.setDirty(s, true)
			case tcell.KeyEnter:
				e.enterPressed(e.out)
				e.draw(s)
				e.setDirty(s, true)
			case tcell.KeyTab:
				e.tabPressed(e.out)
				e.draw(s)
				e.setDirty(s, true)
			case tcell.KeyBacktab:
				e.backTabPressed(e.out)
				e.draw(s)
				e.setDirty(s, true)
			case tcell.KeyRune:
				e.insertRuneAtCurrentPosition(e.out, ev.Rune())
				e.draw(s)
				e.setDirty(s, true)
			case tcell.KeyCtrlB:
				if e.out.Bullets == glyphBullet { // TODO: This will need to change when we support more bullet types
					e.out.Bullets = noBullet
				} else {
					e.out.Bullets = glyphBullet
				}
				e.draw(s)
				e.setDirty(s, true)
			case tcell.KeyCtrlC:
				if e.isSelecting() {
					e.copySelection()
				} else {
					e.copyHeadline()
				}
			case tcell.KeyCtrlF: // for debugging
				e.out.dump(e)
			case tcell.KeyCtrlL:
				e.out.MultiList = !e.out.MultiList
				e.draw(s)
				e.setDirty(s, true)
			case tcell.KeyCtrlV:
				if e.isSelecting() {
					e.pasteSelection()
				} else {
					e.pasteHeadline()
				}
				e.draw(s)
				e.setDirty(s, true)
			case tcell.KeyCtrlX:
				if e.isSelecting() {
					e.cutSelection()
				} else {
					e.cutHeadline()
				}
				e.draw(s)
				e.setDirty(s, true)
			case tcell.KeyCtrlS:
				if currentFilename == "" {
					f := prompt(s, "Filename: ")
					if f != "" {
						currentFilename = f
						err := e.save(filepath.Join(org.currentDirectory, currentFilename))
						if err == nil {
							e.setDirty(s, false)
						} else {
							msg := fmt.Sprintf("Error saving file: %v", err)
							prompt(s, msg)
						}
						org.refresh(s)
						drawScreen(s)
					}
				} else {
					e.save(filepath.Join(org.currentDirectory, currentFilename))
					e.setDirty(s, false)
				}
			case tcell.KeyCtrlT:
				e.editOutlineTitle(s, e.out)
				e.draw(s)
				e.setDirty(s, true)
			case tcell.KeyEscape:
				if e.isSelecting() { // Clear any selection
					e.sel = nil
					e.draw(s)
				}
				org.handleEvents(s, e.out)
				drawScreen(s)
			case tcell.KeyF1:
				showHelp(s)
				prompt(s, "")
				drawScreen(s)
			case tcell.KeyCtrlQ:
				proceed := true
				if e.dirty {
					proceed = e.saveFirst(s)
				}
				if proceed {
					s.Fini()
					os.Exit(0)
				}
			}
		}
	}
}
