# GV - Grand View

A DIY take on the old DOS GrandView app that was very good at managing outlines.


Rethink: Should I ditch the single-stream format for a proper tree structure again?  Seems like I'm halfway between two
models now for no good reason.  Will I get more stability if I just move to a proper tree?


Bugs
* Backspacing on first character of a child headline when parent level more than 1 less (due to join/split activity)
* Backspacing on first character of last empty headline (hitting twice) causes crash
* (Seems like backspacing on first char of a headline is mostly busted)