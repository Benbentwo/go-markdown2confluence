package main

import (
	"github.com/Benbentwo/go-markdown2confluence/cmd"
)

// version of markdown2confluence. Overwritten during build
var version = "0.0.0"

func main() {
	cmd.Execute(version)
}
