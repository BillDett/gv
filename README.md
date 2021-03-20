# GV - Grand View

A DIY take on the old DOS GrandView app that was very good at managing outlines.

GV
* Remember last opened outline and re-open when we launch.  If first time launching, just create an empty outline.
* Save defaults into the storage directory (e.g. color schemes, last opened, enable background saves, etc).  Use a generic name/value pair structure.

Editor
* Support Outline titles when saving (generate filename)
* More navigation (PgUp/PgDown, Home, End)
* Selecting text (shift-navigate)
* Copy/Cut/Paste Headlines
* Collapse/Expand Headlines (use bullets to indicate status)
* Background saves (set up a semaphore so that edits don't conflict with an in-progress save happening via goroutine)

Organizer
* Scrolling of organizer contents
* Need a better way to indicate what folder we're currently in- set as title of organizer?
* The New Outline and New Folder "buttons" are kind of clunky...is there a better way to do this?
* Organizer entries should render outline titles, not filenames
* Support DELETE To remove outlines and folders (with confirmation)

Bugs
* When saving/loading multiple times it seems like the editor gets out of sync & starts adding siblings as children.  Something is not being updated correctly.