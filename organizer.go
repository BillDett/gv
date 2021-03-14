package main

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

type Organizer struct {
	Directory   string // where is Organizer looking for outline files?
	Width       int    // width of the Organizer
	Height      int    // height of the Organizer
	Outlines    []Entry
	CurrentLine int // the current position within the list of outlines
	TopLine     int // index of the topmost outline of the Organizer
}

type Entry struct {
	Name     string
	Filename string
}
