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

	"github.com/gdamore/tcell/v2"
)

type config map[string]string

var cfg config

var currentFilename string

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

var configFilePath string

const lastOpenedOutlineCfgKey = "lastOpenedOutline"

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

func drawScreen(s tcell.Screen) {
	screenWidth, screenHeight = s.Size()
	ed.setScreenSize(s)
	org.setScreenSize(s)
	s.Clear()
	drawBorder(s, 0, 0, screenWidth, screenHeight-1)
	org.draw(s)
	ed.draw(s)
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
//  Set the configuration filePath
func loadConfig(dir string) error {
	configFilePath = filepath.Join(dir, defaultConfigFilename)
	_, err := os.Stat(configFilePath)
	if os.IsNotExist(err) {
		initConfig()
		err = saveConfig()
		if err != nil {
			return err
		}
	}
	buf, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return err
	}
	// Extract the config JSON
	err = json.Unmarshal(buf, &cfg)
	if err != nil {
		return err
	}
	return nil
}

// Default configuration if none is available
func initConfig() {
	cfg = make(config)
	cfg["backgroundColor"] = "black"
	cfg["borderColor"] = "white"
	cfg["defaultTextColor"] = "powderblue"
	cfg["linkColor"] = "blue"
	cfg["listColor"] = "yellow"
	cfg["orgWidthPercent"] = "0.20"
}

func saveConfig() error {
	buf, err := json.MarshalIndent(cfg, "", "   ")
	if err != nil {
		return err
	}
	ioutil.WriteFile(configFilePath, buf, 0644)
	return nil
}

func colorFor(name string) tcell.Color {
	color, found := tcell.ColorNames[cfg[name]]
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
	err = loadConfig(directory)
	if err != nil {
		s.Fini()
		fmt.Printf("Error trying to load config from %s\n", directory)
		os.Exit(1)
	}

	setColors()

	screenWidth, screenHeight = s.Size()

	s.SetStyle(defStyle)

	org = newOrganizer(directory, storageDirectory)
	ed = newEditor(s, org)
	org.refresh(s)

	drawScreen(s)

	ed.handleEvents(s)

}
