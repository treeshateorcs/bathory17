# lydia

lydia is a dead simple, yet full-featured terminal rss reader

## installation

the only dependency is go (i don't know which minimal version is required (how
do i find that out?), i've tested on arch linux, go 1.14)

    $ go get git.sr.ht/~tho/lydia

before first run create a directory in your config dir ($HOME/.config/ on
linux), name it `lydia`, add a few feed urls in the file called `urls`,
one per line

## how to use
j, k; down, up - down, up

d - mark as read, don't open

o - open url in browser

r - refresh feeds and remove read articles from the screen

q, esc - quit



it will take some time to fetch all feeds on launch

you can comment out single feeds by prepending their urls with a pound sign ("#")

change the `TIMER` variable to set how often to auto refresh feeds

currently it's not possible to scroll past the bottom of the window, so the
number of feeds is limited by it

## to do

~~only show unread articles~~ DONE

~~scroll past the bottom of the screen~~ WONTFIX
