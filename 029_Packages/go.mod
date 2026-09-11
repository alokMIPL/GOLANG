module github.com/alok/podcast

go 1.26.5

require github.com/fatih/color v1.19.0

require (
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.42.0 // indirect
)

// Now see we use /fatih/color in main but after that it show //indirect here.
// So to improve this use command `go mod tidy`


// After that this `require github.com/fatih/color v1.19.0` move from indirect block.