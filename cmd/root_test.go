package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	// Get the root command
	rootCmd := RootCmd

	// Capture the output
	output := new(bytes.Buffer)
	rootCmd.SetOut(output)

	// Execute the greet command
	rootCmd.SetArgs([]string{})

	err := Execute()
	assert.NoError(t, err)
}
