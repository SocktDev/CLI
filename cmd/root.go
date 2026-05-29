package cmd

import (
	"fmt"
	"os"

	"github.com/SocktDev/CLI/internal/api"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sockt",
	Short: "Sockt CLI — manage cloud sandboxes",
	Long:  "Create, manage, and interact with Sockt sandboxes from the command line.",
}

func Execute() error {
	return rootCmd.Execute()
}

func newClient() *api.Client {
	c := api.NewClient()
	if tok, _ := rootCmd.Flags().GetString("token"); tok != "" {
		c.SandboxTkn = tok
	}
	return c
}

func exitError(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+msg+"\n", args...)
	os.Exit(1)
}

func init() {
	rootCmd.PersistentFlags().String("token", "", "Sandbox token (overrides SOCKT_SANDBOX_TOKEN)")
}
