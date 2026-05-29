package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/SocktDev/CLI/internal/api"
	"github.com/spf13/cobra"
)

var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "File operations on a sandbox",
}

var filesWriteCmd = &cobra.Command{
	Use:   "write [sandbox-id] [remote-path]",
	Short: "Write a file to the sandbox",
	Long:  "Write content from stdin or a local file to the sandbox filesystem.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]
		remotePath := args[1]
		localFile, _ := cmd.Flags().GetString("file")
		encoding, _ := cmd.Flags().GetString("encoding")

		var content []byte
		var err error
		if localFile != "" {
			content, err = os.ReadFile(localFile)
			if err != nil {
				exitError("read local file: %v", err)
			}
		} else {
			content, err = os.ReadFile("/dev/stdin")
			if err != nil {
				exitError("read stdin: %v", err)
			}
		}

		client := newClient()
		requireSandboxOperational(client, sandboxID)
		req := api.WriteFileRequest{
			Path:       remotePath,
			Content:    string(content),
			Encoding:   encoding,
			CreateDirs: true,
		}

		err = withHostRetry(func() error {
			_, err := client.Post("/sandboxes/"+sandboxID+"/files/write", req)
			return err
		})
		if err != nil {
			exitError("%v", err)
		}

		fmt.Fprintf(os.Stderr, "Written %s (%d bytes)\n", remotePath, len(content))
	},
}

var filesReadCmd = &cobra.Command{
	Use:   "read [sandbox-id] [remote-path]",
	Short: "Read a file from the sandbox",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]
		remotePath := args[1]
		encoding, _ := cmd.Flags().GetString("encoding")
		maxBytes, _ := cmd.Flags().GetInt("max-bytes")

		client := newClient()
		requireSandboxOperational(client, sandboxID)
		req := api.ReadFileRequest{
			Path:     remotePath,
			Encoding: encoding,
			MaxBytes: maxBytes,
		}

		var body []byte
		err := withHostRetry(func() error {
			var err error
			body, err = client.Post("/sandboxes/"+sandboxID+"/files/read", req)
			return err
		})
		if err != nil {
			exitError("%v", err)
		}

		var resp api.ReadFileResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			exitError("parse response: %v", err)
		}

		fmt.Print(resp.Content)
	},
}

var filesListCmd = &cobra.Command{
	Use:   "ls [sandbox-id] [path]",
	Short: "List files in a sandbox directory",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]
		path := "."
		if len(args) > 1 {
			path = args[1]
		}
		recursive, _ := cmd.Flags().GetBool("recursive")
		maxDepth, _ := cmd.Flags().GetInt("max-depth")

		client := newClient()
		requireSandboxOperational(client, sandboxID)
		query := map[string]string{"path": path}
		if recursive {
			query["recursive"] = "true"
		}
		if maxDepth > 0 {
			query["max_depth"] = strconv.Itoa(maxDepth)
		}

		var body []byte
		err := withHostRetry(func() error {
			var err error
			body, err = client.Get("/sandboxes/"+sandboxID+"/files", query)
			return err
		})
		if err != nil {
			exitError("%v", err)
		}

		var wrapped struct {
			Entries []api.FileEntry `json:"entries"`
		}
		if err := json.Unmarshal(body, &wrapped); err != nil {
			exitError("parse response: %v", err)
		}
		entries := wrapped.Entries

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, e := range entries {
			kind := "-"
			if e.IsDir {
				kind = "d"
			}
			fmt.Fprintf(w, "%s\t%d\t%s\n", kind, e.Size, e.Name)
		}
		w.Flush()
	},
}

func init() {
	filesWriteCmd.Flags().StringP("file", "f", "", "Local file to upload (reads stdin if not provided)")
	filesWriteCmd.Flags().String("encoding", "utf8", "File encoding (utf8 or base64)")

	filesReadCmd.Flags().String("encoding", "utf8", "File encoding (utf8 or base64)")
	filesReadCmd.Flags().Int("max-bytes", 0, "Maximum bytes to read")

	filesListCmd.Flags().BoolP("recursive", "r", false, "List recursively")
	filesListCmd.Flags().Int("max-depth", 0, "Max directory depth for recursive listing")

	filesCmd.AddCommand(filesWriteCmd)
	filesCmd.AddCommand(filesReadCmd)
	filesCmd.AddCommand(filesListCmd)
	rootCmd.AddCommand(filesCmd)
}
