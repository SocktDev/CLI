package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/SocktDev/CLI/internal/api"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec [sandbox-id] [command...]",
	Short: "Execute a command in a sandbox",
	Long:  "Run a command in the sandbox and wait for the result.",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]
		command := strings.Join(args[1:], " ")
		workDir, _ := cmd.Flags().GetString("workdir")
		timeoutMs, _ := cmd.Flags().GetInt("timeout")
		pollMs, _ := cmd.Flags().GetInt("poll")

		client := newClient()
		requireSandboxOperational(client, sandboxID)

		req := api.ExecRequest{
			Command:    command,
			WorkingDir: workDir,
			TimeoutMs:  timeoutMs,
		}

		var exec api.Execution
		err := withHostRetry(func() error {
			body, err := client.Post("/sandboxes/"+sandboxID+"/exec", req)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(body, &exec); err != nil {
				return fmt.Errorf("parse response: %w", err)
			}
			return nil
		})
		if err != nil {
			exitError("%v", err)
		}
		if exec.ExecutionID == "" {
			exitError("exec returned no execution_id")
		}

		pollInterval := time.Duration(pollMs) * time.Millisecond
		emptyDoneRetries := 0
		for {
			time.Sleep(pollInterval)

			b, err := client.Get("/executions/"+exec.ExecutionID, nil)
			if err != nil {
				exitError("%v", err)
			}

			var result api.ExecResult
			if err := json.Unmarshal(b, &result); err != nil {
				exitError("parse result: %v", err)
			}

			if result.Status == "running" {
				emptyDoneRetries = 0
				continue
			}

			if len(result.Output) == 0 && result.Error == "" && result.ExitCode == 0 && emptyDoneRetries < 5 {
				emptyDoneRetries++
				continue
			}

			for _, chunk := range result.Output {
				if chunk.Stream == "stdout" {
					fmt.Fprint(os.Stdout, chunk.Chunk)
				} else {
					fmt.Fprint(os.Stderr, chunk.Chunk)
				}
			}

			if result.Error != "" {
				fmt.Fprintf(os.Stderr, "Error: %s\n", result.Error)
			}

			if result.PendingInvoice != nil {
				fmt.Fprintf(os.Stderr, "\nPayment required. Invoice:\n%s\n", result.PendingInvoice.Bolt11)
			}

			exitCode := result.ExitCode
			if result.Status == "failed" && exitCode == 0 {
				exitCode = 1
			}
			if result.Status == "cancelled" && exitCode == 0 {
				exitCode = 1
			}
			os.Exit(exitCode)
		}
	},
}

var execCancelCmd = &cobra.Command{
	Use:   "exec-cancel [execution-id]",
	Short: "Cancel a running execution",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()
		_, err := client.Delete("/executions/" + args[0])
		if err != nil {
			exitError("%v", err)
		}
		fmt.Printf("Execution %s cancelled\n", args[0])
	},
}

func init() {
	execCmd.Flags().SetInterspersed(false)
	execCmd.Flags().StringP("workdir", "d", "", "Working directory inside sandbox")
	execCmd.Flags().Int("timeout", 0, "Execution timeout in milliseconds")
	execCmd.Flags().Int("poll", 500, "Poll interval in milliseconds")

	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(execCancelCmd)
}
