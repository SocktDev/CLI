package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"github.com/SocktDev/CLI/internal/api"
	"github.com/spf13/cobra"
)

var sandboxCmd = &cobra.Command{
	Use:   "sandbox",
	Short: "Manage sandboxes",
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new sandbox",
	Run: func(cmd *cobra.Command, args []string) {
		tier, _ := cmd.Flags().GetString("tier")
		billing, _ := cmd.Flags().GetString("billing")
		prepaidSats, _ := cmd.Flags().GetInt64("prepaid-sats")
		credits, _ := cmd.Flags().GetInt("initial-credits")
		label, _ := cmd.Flags().GetString("label")
		wait, _ := cmd.Flags().GetBool("wait")

		client := newClient()
		req := api.CreateSandboxRequest{
			Tier:              tier,
			BillingMethod:     billing,
			PrepaidSats:       prepaidSats,
			InitialCreditsCents: credits,
			Label:             label,
		}

		body, err := client.Post("/sandboxes", req)
		if err != nil {
			exitError("%v", err)
		}

		var status api.SandboxStatus
		if err := json.Unmarshal(body, &status); err != nil {
			exitError("parse response: %v", err)
		}

		sandboxID := status.IDOrSandboxID()
		if status.SandboxToken != "" {
			fmt.Fprintf(os.Stderr, "Sandbox token: %s\n", status.SandboxToken)
		}

		bolt11 := status.Invoice
		if status.PendingInvoice != nil && status.PendingInvoice.Bolt11 != "" {
			bolt11 = status.PendingInvoice.Bolt11
		}
		if bolt11 != "" {
			fmt.Fprintf(os.Stderr, "Payment required. Invoice:\n%s\n", bolt11)
		}

		if wait {
			waitSandboxReady(client, sandboxID, status.Status == "running")
		}

		fmt.Printf("%s\n", sandboxID)
	},
}

var statusCmd = &cobra.Command{
	Use:   "status [sandbox-id]",
	Short: "Get sandbox status",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()
		body, err := client.Get("/sandboxes/"+args[0], nil)
		if err != nil {
			exitError("%v", err)
		}

		var status api.SandboxStatus
		if err := json.Unmarshal(body, &status); err != nil {
			exitError("parse response: %v", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID:\t%s\n", status.IDOrSandboxID())
		fmt.Fprintf(w, "Status:\t%s\n", status.Status)
		fmt.Fprintf(w, "Tier:\t%s\n", status.Tier)
		fmt.Fprintf(w, "Runtime:\t%s\n", status.Runtime)
		fmt.Fprintf(w, "Billing:\t%s\n", status.BillingMethod)
		if status.SatsPerSecond > 0 {
			fmt.Fprintf(w, "Rate:\t%.2f sats/sec\n", status.SatsPerSecond)
		}
		if status.SecondsRemaining > 0 {
			fmt.Fprintf(w, "Time remaining:\t%ds\n", status.SecondsRemaining)
		}
		if status.WarningLevel != "" {
			fmt.Fprintf(w, "Warning:\t%s\n", status.WarningLevel)
		}
		w.Flush()
	},
}

var pauseCmd = &cobra.Command{
	Use:   "pause [sandbox-id]",
	Short: "Pause a running sandbox",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()
		_, err := client.Post("/sandboxes/"+args[0]+"/pause", nil)
		if err != nil {
			exitError("%v", err)
		}
		fmt.Printf("Sandbox %s paused\n", args[0])
	},
}

var resumeCmd = &cobra.Command{
	Use:   "resume [sandbox-id]",
	Short: "Resume a paused sandbox",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()
		_, err := client.Post("/sandboxes/"+args[0]+"/resume", nil)
		if err != nil {
			exitError("%v", err)
		}
		fmt.Printf("Sandbox %s resumed\n", args[0])
	},
}

var terminateCmd = &cobra.Command{
	Use:   "terminate [sandbox-id]",
	Short: "Terminate a sandbox",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()
		_, err := client.Delete("/sandboxes/" + args[0])
		if err != nil {
			exitError("%v", err)
		}
		fmt.Printf("Sandbox %s terminated\n", args[0])
	},
}

func init() {
	createCmd.Flags().StringP("tier", "t", "nano", "Sandbox tier")
	createCmd.Flags().StringP("billing", "b", "credits", "Billing method (credits or lightning)")
	createCmd.Flags().Int64("prepaid-sats", 0, "Prepaid sats for lightning billing")
	createCmd.Flags().Int("initial-credits", 0, "Initial credits in cents")
	createCmd.Flags().StringP("label", "l", "", "Sandbox label")
	createCmd.Flags().BoolP("wait", "w", false, "Wait until status is running and files/exec are reachable")

	sandboxCmd.AddCommand(createCmd)
	sandboxCmd.AddCommand(statusCmd)
	sandboxCmd.AddCommand(pauseCmd)
	sandboxCmd.AddCommand(resumeCmd)
	sandboxCmd.AddCommand(terminateCmd)
	rootCmd.AddCommand(sandboxCmd)
}
