# lydia

lydia is a dead simple, yet full-featured terminal rss reader

## installation

the only dependency is go

    $ go get git.sr.ht/~tho/lydia

before first run create a directory in your config dir ($HOME/.config/ on
linux), name it `lydia`, there touch `urls` and add a few feed urls in it,
one per line

## how to use
j, k, down, up - down, up

o - open url in browser

r - refresh feeds

q, esc - quit



it will take some time to fetch all feeds on launch

you can comment out single feeds by prepending their urls with a pound sign ("#")

change the `TIMER` variable to set how often to auto refresh feeds

currently it's not possible to scroll past the bottom of the window, so the
number of feeds is limited by it
