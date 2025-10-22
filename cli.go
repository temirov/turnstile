package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"

	"github.com/spf13/cobra"
)

func newRootCommand() *cobra.Command {
	rootCommand := &cobra.Command{
		Use:   "ets",
		Short: "Ephemeral Token Service gateway",
		RunE:  runServeCommand,
	}
	rootCommand.SilenceUsage = true
	rootCommand.AddCommand(newServeCommand())
	rootCommand.AddCommand(newGenerateJwtKeyCommand())
	return rootCommand
}

func newServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the ETS HTTP server",
		RunE:  runServeCommand,
	}
}

func runServeCommand(cmd *cobra.Command, args []string) error {
	gatewayConfig, loadConfigError := loadConfig()
	if loadConfigError != nil {
		return fmt.Errorf("config error: %w", loadConfigError)
	}

	httpServer := newHTTPServer(gatewayConfig)
	log.Printf("ets listening on %s", gatewayConfig.ListenAddress)
	serveError := httpServer.ListenAndServe()
	if serveError != nil && serveError != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", serveError)
	}
	return nil
}

const secretByteLength = 32

func newGenerateJwtKeyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "generate-jwt-key",
		Short: "Generate a HS256 signing key for ETS token issuance",
		RunE: func(cmd *cobra.Command, args []string) error {
			tokenSecret, tokenSecretError := generateRandomHex(secretByteLength)
			if tokenSecretError != nil {
				return fmt.Errorf("generate TVM_JWT_HS256_KEY: %w", tokenSecretError)
			}
			if _, writeError := fmt.Fprintf(cmd.OutOrStdout(), "TVM_JWT_HS256_KEY=%s\n", tokenSecret); writeError != nil {
				return fmt.Errorf("write TVM_JWT_HS256_KEY: %w", writeError)
			}
			return nil
		},
	}
}

var randomRead = rand.Read

func generateRandomHex(byteLength int) (string, error) {
	randomBytes := make([]byte, byteLength)
	if _, readError := randomRead(randomBytes); readError != nil {
		return "", fmt.Errorf("read random bytes: %w", readError)
	}
	return hex.EncodeToString(randomBytes), nil
}
