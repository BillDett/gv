# GV - Grand View

A DIY take on the old DOS GrandView app that was very good at managing outlines.

GV
* Remember last opened outline and re-open when we launch.  If first time launching, just create an empty outline.
* Save defaults into the storage directory (e.g. color schemes, last opened, enable background saves, etc).  Use a generic name/value pair structure.

Editor
* Support Outline titles when saving (generate filename, no need to prompt when doing CTRL-S)
* More navigation (PgUp/PgDown)
* Selecting text (shift-navigate)  [LEFT/RIGHT WORKING, FINISH UP AND DOWN]
* Copy/Cut/Paste selected text
* Copy/Cut/Paste Headlines [should be entire Headline, not portion thereof]
* Collapse/Expand Headlines (use bullets to indicate status)
* Background saves (set up a semaphore so that edits don't conflict with an in-progress save happening via goroutine)

Organizer
* Scrolling of organizer contents
* Organizer width should be dynamic percentage of overall window (update on resize)
* Organizer entries should render outline titles, not filenames

Bugs
* When saving/loading multiple times it seems like the editor gets out of sync & starts adding siblings as children.  Something is not being updated correctly.
* Resizing window really small causes an exception (top and bottom borders not same size anymore)
* Crash when trying to load a zero-byte or malformed .gv file
* drawScreen should only showCursor if editor is handling events