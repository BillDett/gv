package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
)

/*
Organizer is the place where we keep all of our outlines together.  Represents a list of outlines and folders
	based upon the *.gv files found in the current directory (or value of $HOME/.gv/outlines).

Outlines and Folders should be a different color to differentiate them.

First item in list is "New Outline" which is really a button that creates a new blank outline in the Editor

Arrow up and down navigate the list.

Enter on an outline opens it in the Editor.

Enter on a folder opens that folder to display the *.gv/folders inside.  Topmost entry is ".." to indicate you can go back
	to the parent directory.

Delete pressed on an outline prompts for its removal. Removing an outline means the file is deleted and the Organizer is refreshed.
Delete on a folder does nothing.

*/

type organizer struct {
	baseDir          string // base directory for gv's files
	directory        string // where is Organizer looking for outline files?
	currentDirectory string // what directory are we currently in?
	width            int    // width of the Organizer
	height           int    // height of the Organizer
	entries          []*entry
	currentLine      int  // the current position within the list of outlines
	topLine          int  // index of the topmost outline of the Organizer
	inFocus          bool // Is the organizer currently in focus?
}

type entry struct {
	name     string
	filename string
	isDir    bool
}

func newEntry(n string, f string, d bool) *entry {
	return &entry{n, f, d}
}

func newOrganizer(dir string, storageDir string) *organizer {
	return &organizer{dir, storageDir, storageDir, 0, 0, nil, 0, 0, false}
}

func (org *organizer) setScreenSize(s tcell.Screen) {
	var pct float64
	fallback := 0.50
	pctStr, ok := cfg["orgWidthPercent"]
	if !ok {
		pct = fallback // Safe default just in case it's not in config
	} else {
		var err error
		pct, err = strconv.ParseFloat(pctStr, 2)
		if err != nil {
			pct = fallback // Ignore parsing error, just default
		}
	}
	org.width = int(pct * float64(screenWidth-3))
	org.height = screenHeight - 2
}

func (org *organizer) refresh(s tcell.Screen) {
	org.entries = []*entry{}
	contents, err := org.readDirectory()
	if err != nil {
		msg := fmt.Sprintf("Error reading storage; %v", err)
		prompt(s, msg)
		return
	}
	org.entries = append(org.entries, contents...)
}

func (org *organizer) readDirectory() ([]*entry, error) {
	files, err := ioutil.ReadDir(org.currentDirectory)
	if err != nil {
		return nil, err
	}
	outlines := []*entry{}
	folders := []*entry{}
	for _, info := range files {
		if info.Mode().IsRegular() {
			if strings.HasSuffix(info.Name(), ".gv") {
				title, err := org.getTitleFrom(filepath.Join(org.currentDirectory, info.Name()))
				if err != nil {
					return nil, err
				}
				outlines = append(outlines, newEntry(title, info.Name(), false))
			}
		} else if info.Mode().IsDir() && !strings.HasPrefix(info.Name(), ".") {
			folders = append(folders, newEntry(info.Name(), info.Name(), true))
		}
	}
	// Sort everything nicely
	sort.Slice(folders, func(i, j int) bool { return folders[i].name < folders[j].name })
	sort.Slice(outlines, func(i, j int) bool { return outlines[i].name < outlines[j].name })
	// Put it together
	result := []*entry{}
	if filepath.Clean(org.currentDirectory) != filepath.Clean(org.directory) { // we are in a child folder
		result = append(result, newEntry("..", "..", true))
	}
	result = append(result, folders...)
	result = append(result, outlines...)
	return result, nil
}

// peek inside the outline file and return the Title field
// TODO: This could be very slow for large numbers of outlines...how can we do this w/out unmarshaling entire outline?
func (org *organizer) getTitleFrom(filename string) (string, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	// Extract the outline JSON
	var out Outline
	err = json.Unmarshal(buf, &out)
	if err != nil {
		return "", err
	}
	result := "no title"
	if out.Title != "" {
		result = out.Title
	}
	return result, nil
}

// Clear out the contents of the organizer's window
//  We depend on the caller to eventually do s.Show()
func (org *organizer) clear(s tcell.Screen) {
	for y := 1; y < org.height; y++ {
		for x := 1; x < org.width; x++ {
			s.SetContent(x, y, ' ', nil, defStyle)
		}
	}
}

// draw the visible contents of the organizer
func (org *organizer) draw(s tcell.Screen) {
	org.clear(s)
	org.height = screenHeight - 2 // TODO: TAKE THIS OUT & MOVE TO setScrenSize() ?
	width := org.width - 1
	y := 1
	for c := org.topLine; c < len(org.entries); c++ {
		if y < org.height {
			style := fileStyle
			if org.inFocus && c == org.currentLine {
				style = selectedStyle
			} else if org.entries[c].isDir {
				style = dirStyle
			}
			// Write out the entry name
			for x, r := range org.entries[c].name {
				if x < width {
					if x == width-1 { // probably will go over width of organizer, just use an ellipsis
						r = ellipsis
					}
					s.SetContent(1+x, y, r, nil, style)
				}
			}
		}
		y++
	}
	s.Show()
}

// deal with whatever entry was selected
// Return whether or not we should release Organizer focus
func (org *organizer) entrySelected(s tcell.Screen) bool {
	entry := org.entries[org.currentLine]
	if entry.isDir {
		org.currentDirectory = filepath.Join(org.currentDirectory, entry.filename)
		org.clear(s)
		org.refresh(s)
		drawTopBorder(s)
		return false
	} else {
		ed.open(s, filepath.Join(org.currentDirectory, entry.filename))
		return true
	}
}

func (org *organizer) newFolder(s tcell.Screen) {
	f := prompt(s, "Enter new Folder name: ")
	if f != "" {
		err := os.Mkdir(filepath.Join(org.currentDirectory, f), 0700)
		if err != nil {
			msg := fmt.Sprintf("Error creating directory %s; %v", f, err)
			prompt(s, msg)
		}
		org.clear(s)
		org.refresh(s)
	}
}

func (org *organizer) deleteSelected(s tcell.Screen) {
	entry := org.entries[org.currentLine]
	msg := fmt.Sprintf("Delete %s (Y/N)?", entry.name)
	proceed := false
	response := prompt(s, msg)
	if response != "" {
		if strings.ToUpper(response) == "Y" {
			if entry.isDir {
				msg := fmt.Sprintf("%s is a Folder- all contents will be removed! (Y/N)?", entry.name)
				response := prompt(s, msg)
				if response != "" {
					if strings.ToUpper(response) == "Y" {
						proceed = true
					}
				}
			} else {
				proceed = true
			}
		}
		if proceed {
			thefile := filepath.Join(org.currentDirectory, entry.filename)
			err := os.RemoveAll(thefile)
			if err != nil {
				msg := fmt.Sprintf("Error removing %s; %v", thefile, err)
				prompt(s, msg)
			}
			org.clear(s)
			org.refresh(s)
		}
	}
}

func (org *organizer) dump() {
	out := fmt.Sprintf("%v\ncurrentLine %d  currentDirectory %s   directory %s\n",
		org.entries, org.currentLine, org.currentDirectory, org.directory)
	ioutil.WriteFile("orgdump.txt", []byte(out), 0644)
}

func (org *organizer) handleEvents(s tcell.Screen, o *Outline) {
	done := false
	org.inFocus = true
	s.HideCursor()
	org.draw(s)
	for !done {
		switch ev := s.PollEvent().(type) {
		case *tcell.EventResize:
			s.Sync()
			screenWidth, screenHeight = s.Size()
			drawScreen(s)
			s.HideCursor()
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyDown:
				if org.currentLine < len(org.entries)-1 {
					org.currentLine++
					if org.currentLine-org.topLine+1 >= org.height { // Scroll?
						org.topLine++
					}
					org.draw(s)
				}
			case tcell.KeyUp:
				if org.currentLine != 0 {
					org.currentLine--
					if org.currentLine != 0 && org.currentLine-org.topLine+1 == 1 { // Scroll?
						org.topLine--
					}
					org.draw(s)
				}
			case tcell.KeyEnter:
				done = org.entrySelected(s)
				org.draw(s)
			case tcell.KeyCtrlQ:
				proceed := true
				if ed.dirty {
					proceed = ed.saveFirst(s)
				}
				if proceed {
					s.Fini()
					os.Exit(0)
				}
			case tcell.KeyCtrlO:
				ed.newOutline(s, "")
				org.refresh(s)
				org.draw(s)
				done = true
			case tcell.KeyCtrlF:
				org.newFolder(s)
				org.draw(s)
			case tcell.KeyCtrlD:
				org.deleteSelected(s)
				org.draw(s)
			case tcell.KeyCtrlP:
				org.dump()
			case tcell.KeyF1:
				showHelp(s)
				prompt(s, "")
				drawScreen(s)
			case tcell.KeyEscape:
				done = true
			}
		}
	}
	org.inFocus = false
}
