package main

import (
	"io/ioutil"
	"strings"
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
	currentLine int // the current position within the list of outlines
	topLine     int // index of the topmost outline of the Organizer
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

func newOrganizer(height int) *organizer {
	// TODO: We should look in $GVHOME or $HOME/.gv/outlines
	return &organizer{".", organizerWidth, height, nil, 0, 0}
}

func (org *organizer) refresh() {
	org.entries = []*entry{}
	org.entries = append(org.entries, newEntry("New Outline", "", false))
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
