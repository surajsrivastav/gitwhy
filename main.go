package main

import (
	"github.com/surajsrivastav/gitwhy/cmd"
)

var (
	commit string
	date   string
)

func main() {
	cmd.SetVersion(commit, date)
	cmd.Execute()
}
