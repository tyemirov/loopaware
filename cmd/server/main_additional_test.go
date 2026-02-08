package main

import (
	"os"
	"testing"
)

const testHelpFlag = "--help"

func TestMainRunsHelpCommand(testingT *testing.T) {
	originalArguments := os.Args
	testingT.Cleanup(func() {
		os.Args = originalArguments
	})

	os.Args = []string{commandUseName, testHelpFlag}
	main()
}
