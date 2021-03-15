package main

import (
	"fmt"
	"io/ioutil"
	"os"
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
	directory   string // where is Organizer looking for outline files?
	width       int    // width of the Organizer
	height      int    // height of the Organizer
	entries     []*entry
	currentLine int  // the current position within the list of outlines
	topLine     int  // index of the topmost outline of the Organizer
	inFocus     bool // Is the organizer currently in focus?
}

type entry struct {
	name     string
	filename string
	isDir    bool
}

var organizerWidth int = 15

func newEntry(n string, f string, d bool) *entry {
	return &entry{n, f, d}
}

func newOrganizer(directory string, height int) *organizer {
	// TODO: We should look in $GVHOME or $HOME/.gv/outlines
	return &organizer{directory, organizerWidth, height, nil, 0, 0, false}
}

func (org *organizer) refresh() {
	org.entries = []*entry{}
	org.entries = append(org.entries, newEntry("New Outline", "", false))
	org.entries = append(org.entries, newEntry("New Folder", "", false))
	org.readDirectory()
}

func (org *organizer) readDirectory() error {
	files, err := ioutil.ReadDir(org.directory)
	if err != nil {
		return err
	}
	for _, info := range files {
		if info.Mode().IsRegular() {
			if strings.HasSuffix(info.Name(), ".gv") {
				org.entries = append(org.entries, newEntry(info.Name(), info.Name(), false))
			}
		} else if info.Mode().IsDir() && !strings.HasPrefix(info.Name(), ".") {
			org.entries = append(org.entries, newEntry(info.Name(), info.Name(), true))
		}
	}
	// TODO: We should sort the entries array so that all directories are on top, in alphabetical order
	//   each.
	return nil
}

// draw the visible contents of the organizer
func (org *organizer) draw(s tcell.Screen, height int) {
	org.height = height - 2
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
				if x < org.width {
					s.SetContent(1+x, y, r, nil, style)
				}
			}
		}
		y++
	}
	s.Show()
}

// deal with whatever entry was selected
//  "Open" an outline or directory.  Or create a new outline or directory
func (org *organizer) entrySelected(s tcell.Screen) {
	if org.currentLine == 0 { // new outline

	} else if org.currentLine == 1 { // new directory

	} else { // open a file or directory
		entry := org.entries[org.currentLine]
		if entry.isDir {

		} else {
			ed.open(s, entry.filename)
		}
	}
}

func (org *organizer) dump() {
	out := fmt.Sprintf("%v\ncurrentLine %d", org.entries, org.currentLine)
	ioutil.WriteFile("orgdump.txt", []byte(out), 0644)
}

func (org *organizer) handleEvents(s tcell.Screen, o *Outline) {
	done := false
	org.inFocus = true
	s.HideCursor()
	org.draw(s, org.height)
	for !done {
		_, height := s.Size()
		switch ev := s.PollEvent().(type) {
		case *tcell.EventResize:
			s.Sync()
			org.draw(s, height)
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyDown:
				if org.currentLine < len(org.entries)-1 {
					org.currentLine++
					org.draw(s, org.height)
				}
			case tcell.KeyUp:
				if org.currentLine != 0 {
					org.currentLine--
					org.draw(s, height)
				}
			case tcell.KeyEnter:
				org.entrySelected(s)
				org.draw(s, height)
				done = true
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
