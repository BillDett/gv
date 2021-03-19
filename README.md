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
* Clear organizer region when contents change (e.g. change folder)
* Scrolling of organizer contents
* Organizer entries should render outline titles, not filenames
* Support DELETE To remove outlines and folders (with confirmation)

Bugs
* Editor flickers a lot during cursor movement or scrolling.  Layout/Render algorithm could use some optimizations 
    * When we move the cursor and aren't scrolling, we don't have to redraw the whole screen...
    * Just layout one Headline at a time (without children) when we know we're just changing text?
    * It looks like tcell is optimizing and only redrawing changed cells.  So it might make sense to just update the
      cursor unless we are scrolling the page or actually editing text.