package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func TestGenerateSecretsCommandOutputsHexAssignments(t *testing.T) {
	rootCommand := newRootCommand()
	var commandOutput bytes.Buffer
	rootCommand.SetOut(&commandOutput)
	rootCommand.SetErr(&commandOutput)
	rootCommand.SetArgs([]string{"generate-secrets"})

	if executeError := rootCommand.Execute(); executeError != nil {
		t.Fatalf("unexpected error: %v", executeError)
	}

	outputLines := strings.Split(strings.TrimSpace(commandOutput.String()), "\n")
	if len(outputLines) != 2 {
		t.Fatalf("expected 2 output lines, got %d", len(outputLines))
	}

	expectedKeys := []string{"TVM_JWT_HS256_KEY", "UPSTREAM_SERVICE_SECRET"}
	for index, expectedKey := range expectedKeys {
		line := outputLines[index]
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			t.Fatalf("expected key=value format on line %d: %q", index, line)
		}
		key := parts[0]
		value := parts[1]
		if key != expectedKey {
			t.Fatalf("expected key %q, got %q on line %d", expectedKey, key, index)
		}
		if len(value) != secretByteLength*2 {
			t.Fatalf("expected %d hex characters for %s, got %d", secretByteLength*2, expectedKey, len(value))
		}
		if _, decodeError := hex.DecodeString(value); decodeError != nil {
			t.Fatalf("value for %s is not valid hex: %v", expectedKey, decodeError)
		}
	}
}

func TestGenerateSecretsCommandPropagatesEntropyError(t *testing.T) {
	originalRandomRead := randomRead
	t.Cleanup(func() {
		randomRead = originalRandomRead
	})

	randomRead = func(destination []byte) (int, error) {
		return 0, errors.New("entropy source unavailable")
	}

	rootCommand := newRootCommand()
	rootCommand.SetArgs([]string{"generate-secrets"})

	if executeError := rootCommand.Execute(); executeError == nil {
		t.Fatalf("expected error when entropy source fails")
	}
}
