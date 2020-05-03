![Screenshot](https://i.imgur.com/iTe1Rpw.png)

# lydia

lydia is a dead simple, yet full-featured terminal rss reader

## installation

the only dependency is go >=1.13, tested on arch linux, go 1.14

    $ go get git.sr.ht/~tho/lydia

before the first run create a directory in your config dir ($HOME/.config/ on
linux), name it `lydia`, add a few feed urls in the file called `urls`,
one per line

## how to use
j, k; down, up - down, up

d - mark as read, don't open

A - mark everything as read

o - open url in your default browser

R - refresh feeds and remove read articles from the screen

r - remover read articles, don't refresh

q, esc - quit



it will take some time to fetch all feeds on the first launch

you can comment out single feeds by prepending their urls with a pound sign ("#")

change the `TIMER` variable to set how often to auto refresh feeds

## to do

~~only show unread articles~~ DONE

~~scroll past the bottom of the screen~~ WONTFIX

~~there may be issues when too few articles are on the screen, i'm working on it~~ DONE


## contributing

send bugreports [to this email](mailto:~tho/lydia@lists.sr.ht)
