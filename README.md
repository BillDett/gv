# GV - Grand View

A DIY take on the old DOS GrandView app that was very good at managing outlines.


Bugs
* Indenting past 9 levels causes parsing issues with the level integers
* Indenting a headline with children causes the first non-child headline to indent as well
* When more than 1 headline, indenting the last headline causes a crash
* Backspacing on first character of last empty headline (hitting twice) causes crash