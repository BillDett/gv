# TODO in gv

README
* Replace screenshot with asciicinema once colors/UX stabilized

GV
* Look at moving to a sqlite based datastore for everything (https://pkg.go.dev/modernc.org/sqlite)
* Add a little API to this so we can push new outlines to it and/or pull outlines if desired (start API only via cmdline flag).
* Better visual cue whether the Organizer or the Editor is currently in focus (maybe dim out the colors of the contents or titlebar?)
* Add some Outline statistics on bottom right of screen (# lines, # headlines, # words, etc)

* Mouse support
  * Keyboard should be our preferred method, but it's necessary to have some rudimentary support
  * Editor
    * Set the cursor position
    * Use scroll wheel to move cursor up and down a line
    * Select text within a single Headline (constrain the mouse Y range while selecting)
  * Organizer
    * Hover over entries highlights them
    * Scroll wheel to scroll entries list up and down
    * Click on entries to open them

Editor
* 'Raise' (CTRL-J) or 'Lower' (CTRL-K) Headlines.
  * moves a Headline (an all of its children) up or down the screen, respecting hierarchies
  * within current parent, Headline moves within siblings
  * Raising when first child makes last child of previous Headline (or no-op if first Headline)
  * Lowering when last child makes first child or next Headline (or no-op if last Headline)
  * Preerve the Headline's expansion flag
* Copy/Cut/Paste
  * CTRL-C/CTRL-X/CTRL-V
  * Applies to any currently selected text 
  * Applies to current Headline and it's children recursively (if no text selected)
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
* Support custom keymappings.  Allow overrides on certain CTRL combos within the gv config file

Organizer
* Finish adding the FolderIndex stuff
  * BUG: When navigating 'up' the tree of folders, the Organizer top border label is set to ".." and not the actual parent's name.
  * Implement a way to 'rename' a folder w/out changing the directory name
* Put last accessed outilne at top of Organizer?
* Cross-outline searches in the Organizer (like ripgrep).  Show the search results in the Organizer.  ESC to clear.  (https://gobyexample.com/line-filters would get us started on a simple 'grep')
* Show a visual indicator in right border when Organizer contents extend above or beyond current view
* Ability to copy/cut/paste Outlines into folders using CTRL-C/CTRL-X/CTRL-V
  * Use a different color for Outline in Organizer when it's been copied, remove it when it's cut
* BUG: Cursor does not get set to first item in the list when drilling into a sub-folder.

Bugs
* Changing the title of a "New Outline" doesnt update in the Organizer
* Resizing window really small causes an exception (top and bottom borders not same size anymore)
* When organizer sub-folder is empty, the up/down navigation highlight can get 'lost'- you need to poke up and down a bit to find it again.
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
