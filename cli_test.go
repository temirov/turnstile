package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func TestGenerateJwtKeyCommandOutputsHexAssignment(t *testing.T) {
	rootCommand := newRootCommand()
	var commandOutput bytes.Buffer
	rootCommand.SetOut(&commandOutput)
	rootCommand.SetErr(&commandOutput)
	rootCommand.SetArgs([]string{"generate-jwt-key"})

	if executeError := rootCommand.Execute(); executeError != nil {
		t.Fatalf("unexpected error: %v", executeError)
	}

	outputLines := strings.Split(strings.TrimSpace(commandOutput.String()), "\n")
	if len(outputLines) != 1 {
		t.Fatalf("expected 1 output line, got %d", len(outputLines))
	}

	line := outputLines[0]
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		t.Fatalf("expected key=value format: %q", line)
	}
	key := parts[0]
	value := parts[1]
	if key != "TVM_JWT_HS256_KEY" {
		t.Fatalf("expected key TVM_JWT_HS256_KEY, got %q", key)
	}
	if len(value) != secretByteLength*2 {
		t.Fatalf("expected %d hex characters, got %d", secretByteLength*2, len(value))
	}
	if _, decodeError := hex.DecodeString(value); decodeError != nil {
		t.Fatalf("value is not valid hex: %v", decodeError)
	}
}

func TestGenerateJwtKeyCommandPropagatesEntropyError(t *testing.T) {
	originalRandomRead := randomRead
	t.Cleanup(func() {
		randomRead = originalRandomRead
	})

	randomRead = func(destination []byte) (int, error) {
		return 0, errors.New("entropy source unavailable")
	}

	rootCommand := newRootCommand()
	rootCommand.SetArgs([]string{"generate-jwt-key"})

	if executeError := rootCommand.Execute(); executeError == nil {
		t.Fatalf("expected error when entropy source fails")
	}
}
