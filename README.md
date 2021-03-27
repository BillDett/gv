# GV - Grand View

A DIY take on the old DOS GrandView app that was very good at managing outlines.

GV
* Remember last opened outline and re-open when we launch.  If first time launching, just create an empty outline.
* Save defaults into the storage directory (e.g. color schemes, last opened, enable background saves, etc).  Use a generic name/value pair structure.
* Support F1 bringing up an overlay "window" that has all commands and help text
* Need a better way to show CTRL-keys that are available based on focus

Editor
* Support Outline titles when saving (generate filename)
* More navigation (PgUp/PgDown, Home, End)
* Selecting text (shift-navigate)
* Copy/Cut/Paste Headlines
* Collapse/Expand Headlines (use bullets to indicate status)
* Background saves (set up a semaphore so that edits don't conflict with an in-progress save happening via goroutine)

Organizer
* Scrolling of organizer contents
* Organizer width should be dynamic percentage of overall window (update on resize)
* Organizer entries should render outline titles, not filenames
* Support DELETE To remove outlines and folders (with confirmation)


Bugs
* Quitting while Organizer in focus does not prompt to save the current outline
* When saving/loading multiple times it seems like the editor gets out of sync & starts adding siblings as children.  Something is not being updated correctly.
* Resizing window vary small causes editor text to overrun right border some times...word wrap algorithm not exactly right
* Crash when GVHOME isn't writable when setting up storage for first time
* Crash when trying to load a zero-byte or malformed .gv file