# TODO in gv

README
* Get a screenshot in there

GV
* Add a little API to this so we can push new outlines to it and/or pull outlines if desired (start API only via cmdline flag).
* Better visual cue whether the Organizer or the Editor is currently in focus (maybe dim out the colors of the contents or titlebar?)
* Add some Outline statistics on bottom right of screen (# lines, # headlines, # words, etc)

Editor
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
* Show a visual indicator in right border when Editor contents extend above or beyond current view

Organizer
* Put last accessed outilne at top of Organizer?
* Cross-outline searches in the Organizer (like ripgrep).  Show the search results in the Organizer.  ESC to clear.  (https://gobyexample.com/line-filters would get us started on a simple 'grep')
* Show a visual indicator in right border when Organizer contents extend above or beyond current view
* Ability to copy/cut/paste Outlines into folders using CTRL-C/CTRL-X/CTRL-V
  * Use a different color for Outline in Organizer when it's been copied, remove it when it's cut

Bugs
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
