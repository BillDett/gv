package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
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

var organizerWidth int = 20

func newEntry(n string, f string, d bool) *entry {
	return &entry{n, f, d}
}

func newOrganizer(directory string, height int) *organizer {
	// TODO: We should look in $GVHOME or $HOME/.gv/outlines
	return &organizer{directory, directory, organizerWidth, height, nil, 0, 0, false}
}

func (org *organizer) refresh(s tcell.Screen) {
	org.entries = []*entry{}
	org.entries = append(org.entries, newEntry("New Outline", "", false))
	org.entries = append(org.entries, newEntry("New Folder", "", false))
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
				outlines = append(outlines, newEntry(info.Name(), info.Name(), false))
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

// Clear out the contents of the organizer's window
//  We depend on the caller to eventually do s.Show()
func (org *organizer) clear(s tcell.Screen) {
	for y := 1; y < ed.editorHeight; y++ {
		for x := 1; x < org.width; x++ {
			s.SetContent(x, y, ' ', nil, defStyle)
		}
	}
}

// draw the visible contents of the organizer
func (org *organizer) draw(s tcell.Screen) {
	org.height = screenHeight - 2
	width := org.width - 1
	y := 1
	for c := org.topLine; c < len(org.entries); c++ {
		if y < org.height {
			style := fileStyle
			if org.inFocus && c == org.currentLine {
				style = selectedStyle
			} else if c < 2 {
				style = buttonStyle
			} else if org.entries[c].isDir {
				style = dirStyle
			}
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
//  "Open" an outline or folder.  Or create a new outline or folder
// Return whether or not we should release Organizer focus
func (org *organizer) entrySelected(s tcell.Screen) bool {
	if org.currentLine == 0 { // new outline
		ed.newOutline(s)
	} else if org.currentLine == 1 { // new folder
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
	} else { // open a file or folder
		entry := org.entries[org.currentLine]
		if entry.isDir {
			org.currentDirectory = filepath.Join(org.currentDirectory, entry.filename)
			org.clear(s)
			org.refresh(s)
			drawTopBorder(s)
			return false
		} else {
			ed.open(s, entry.filename)
		}
	}
	return true
}

func (org *organizer) dump() {
	out := fmt.Sprintf("%v\ncurrentLine %d", org.entries, org.currentLine)
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
					org.draw(s)
				}
			case tcell.KeyUp:
				if org.currentLine != 0 {
					org.currentLine--
					org.draw(s)
				}
			case tcell.KeyEnter:
				done = org.entrySelected(s)
				org.draw(s)
			case tcell.KeyCtrlQ:
				s.Fini()
				os.Exit(0)
			case tcell.KeyCtrlF:
				org.dump()
			case tcell.KeyEscape:
				done = true
			}
		}
	}
	org.inFocus = false
}
