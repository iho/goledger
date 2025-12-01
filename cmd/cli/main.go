package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	baseURL string
	timeout time.Duration
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "goledger-cli",
		Short: "GoLedger CLI tool",
		Long:  `A command line interface for interacting with the GoLedger API.`,
	}

	rootCmd.PersistentFlags().StringVar(&baseURL, "url", "http://localhost:8080", "Base URL of the GoLedger API")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 10*time.Second, "Request timeout")

	// Ledger commands
	ledgerCmd := &cobra.Command{
		Use:   "ledger",
		Short: "Ledger operations",
	}

	consistencyCmd := &cobra.Command{
		Use:   "consistency",
		Short: "Check ledger consistency",
		Run: func(cmd *cobra.Command, args []string) {
			checkConsistency()
		},
	}

	ledgerCmd.AddCommand(consistencyCmd)
	rootCmd.AddCommand(ledgerCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func checkConsistency() {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(baseURL + "/api/v1/ledger/consistency")
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Consistency check FAILED (Status: %d)\nResponse: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Consistency check PASSED\n")
	if consistent, ok := result["consistent"].(bool); ok {
		fmt.Printf("Consistent: %v\n", consistent)
	}
	fmt.Printf("Status: %s\n", result["status"])
}
