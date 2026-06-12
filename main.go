package main

import (
	"github.com/anomalyco/gitwhy/cmd"
)

var (
	commit string
	date   string
)

func main() {
	cmd.SetVersion(commit, date)
	cmd.Execute()
}
