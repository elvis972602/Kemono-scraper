module github.com/elvis972602/kemono-scraper

go 1.18

require (
	github.com/mattn/go-colorable v0.1.13
	github.com/mattn/go-sqlite3 v1.14.16
	github.com/zalando/go-keyring v0.2.2
	golang.org/x/crypto v0.5.0
	golang.org/x/net v0.8.0
	golang.org/x/sys v0.6.0
	golang.org/x/term v0.6.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
)

replace github.com/mattn/go-colorable => github.com/elvis972602/go-colorable v0.0.0-20230322143039-2b733b5d5ca7
