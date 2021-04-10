# GV - Grand View

A DIY take on the old DOS GrandView app that was very good at managing outlines.

GV
* Remember last opened outline and re-open when we launch.  If first time launching, just create an empty outline.
* Save defaults into the storage directory (e.g. color schemes, last opened, enable background saves, etc).  Use a generic name/value pair structure.

Editor
* More navigation (PgUp/PgDown)
* Cut/Paste selected text
* Copy/Cut/Paste Headlines [should be entire Headline, not portion thereof]
* Background saves (set up a semaphore so that edits don't conflict with an in-progress save happening via goroutine)
* Add text searching (CTRL-F)- scan forward/backward through matches
* SHIFT-CTRL-UP/DOWN - collapse or expand all Headlines
* Better use of color
    * identify collapsed Headlines (maybe a bit dimmer?)
    * make bullets a different color than text?
    * soften the highlight on selected text
    * change color of a copied Headline (and its children)

Organizer
* Scrolling of organizer contents
* Organizer width should be dynamic percentage of overall window (update on resize)
* Put last accessed outilne at top of Organizer?

Bugs
* When saving/loading multiple times it seems like the editor gets out of sync & starts adding siblings as children.  Something is not being updated correctly.
* Resizing window really small causes an exception (top and bottom borders not same size anymore)
* Crash when trying to load a zero-byte or malformed .gv file
* drawScreen should only showCursor if editor is handling events
* Kind of weird behavior when selecting up then down right afterwards..is that what users would expect?
* Bug when un-indenting first of several children.  Unindented child 'keeps' its subseqeuent siblings as its own children.
    * Start with:
        - One
          - A
          - B
          - C
    * Unindent A.  Result is:
        - One
          - B
        - A
          - C
    
    Should be:
        - One
        - A
          - B
          - C       
