package main

import "os"

func main() {
	rootCommand := newRootCommand()
	if executeError := rootCommand.Execute(); executeError != nil {
		os.Exit(1)
	}
}
