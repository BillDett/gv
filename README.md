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
* Organizer entries should render outline titles, not filenames
* When in a sub-folder, add a ".." entry so we can navigate upwards
* Sort folders separately from outlines
* Create folder
* Create Outline
* Support DELETE To remove outlines and folders (with confirmation)

Bugs
* CTRL-Q in Editor to quit always prompts to save (even when buffer isn't dirty) 