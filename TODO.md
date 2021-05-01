# TODO in gv

README
* Replace screenshot with asciicinema once colors/UX stabilized

GV
* Move our data model on top of boltdb instead of raw filesystem.  Will give us some more flexibility to add features over time.  Store everything into a single boltdb database file (organizer metadata, outlines, search index, etc..).  Add a `gv --export` flag that dumps out all of the outlines and organizer metadata into plain files in case we want to move to a different tool.
* Add a little API to this so we can push new outlines to it and/or pull outlines if desired (start API only via cmdline flag).
* Better visual cue whether the Organizer or the Editor is currently in focus (maybe dim out the colors of the contents or titlebar?)
* Add some Outline statistics on bottom right of screen (# lines, # headlines, # words, etc)

Editor
* Cut/Paste selected text
* Copy/Cut/Paste Headlines [should be entire Headline, not portion thereof]
* Background saves (set up a semaphore so that edits don't conflict with an in-progress save happening via goroutine)
* Add text searching (CTRL-F)- scan forward/backward through matches
* "Splitting" a headline with enter key at first character should not make the Headline a child of an empty Headline- it should just make the Headline a sibling (looks weird when it turns into a child underneath a blank line)
* SHIFT-CTRL-UP/DOWN - collapse or expand all Headlines
* Better use of color
    * identify collapsed Headlines (maybe a bit dimmer?)
    * make bullets a different color than text?
    * soften the highlight on selected text
    * change color of a copied Headline (and its children)
* Show a visual indicator in right border when Editor contents extend above or beyond current view

Organizer
* Put last accessed outilne at top of Organizer?
* Cross-outline searches in the Organizer (like ripgrep).  Show the search results in the Organizer.  ESC to clear.  (https://gobyexample.com/line-filters would get us started on a simple 'grep') [SEE ABOVE ON boltdb- MAKES THIS SIMPLER]
* Show a visual indicator in right border when Organizer contents extend above or beyond current view
* Ability to copy/cut/paste Outlines into folders using CTRL-C/CTRL-X/CTRL-V
  * Use a different color for Outline in Organizer when it's been copied, remove it when it's cut

Bugs
* Changing the title of a "New Outline" doesnt update in the Organizer
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
