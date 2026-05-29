package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/SocktDev/CLI/internal/api"
	"github.com/spf13/cobra"
)

var tiersCmd = &cobra.Command{
	Use:   "tiers",
	Short: "List available sandbox tiers and pricing",
	Run: func(cmd *cobra.Command, args []string) {
		client := newClient()
		body, err := client.Get("/sandboxes/tiers", nil)
		if err != nil {
			exitError("%v", err)
		}

		var wrapped struct {
			Tiers []api.Tier `json:"tiers"`
		}
		if err := json.Unmarshal(body, &wrapped); err != nil {
			exitError("parse response: %v", err)
		}
		tiers := wrapped.Tiers

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TIER\tSATS/SEC\tUSD/HOUR\tSATS/HOUR\tSATS/DAY")
		for _, t := range tiers {
			name := t.Name
			if name == "" {
				name = t.Tier
			}
			fmt.Fprintf(w, "%s\t%.2f\t$%.4f\t%.0f\t%.0f\n",
				name, t.SatsPerSecond, t.CostPerHourUSD, t.CostPerHourSats, t.CostPerDaySats)
		}
		w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(tiersCmd)
}
