package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var shellCmd = &cobra.Command{
	Use:   "shell [sandbox-id]",
	Short: "Open an interactive shell to a sandbox",
	Long:  "Connect to the sandbox via WebSocket for a live terminal session.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]
		client := newClient()
		requireSandboxOperational(client, sandboxID)

		body, err := client.Get("/sandboxes/"+sandboxID+"/shell-url", nil)
		if err != nil {
			exitError("%v", err)
		}

		var resp struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			exitError("parse response: %v", err)
		}

		headers := http.Header{}
		if tok := client.APIKey; tok != "" {
			headers.Set("Authorization", "Bearer "+tok)
		}
		if tok := client.SandboxTkn; tok != "" {
			headers.Set("Authorization", "Bearer "+tok)
		}

		conn, _, err := websocket.DefaultDialer.Dial(resp.URL, headers)
		if err != nil {
			exitError("connect: %v", err)
		}
		defer conn.Close()

		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			exitError("terminal raw mode: %v", err)
		}
		defer term.Restore(int(os.Stdin.Fd()), oldState)

		sigCh := make(chan os.Signal, 1)
		notifyShellSignals(sigCh)

		sendResize := func() {
			w, h, err := term.GetSize(int(os.Stdin.Fd()))
			if err != nil {
				return
			}
			msg, _ := json.Marshal(map[string]interface{}{
				"type": "resize",
				"cols": w,
				"rows": h,
			})
			conn.WriteMessage(websocket.TextMessage, msg)
		}
		sendResize()

		done := make(chan struct{})

		go func() {
			defer close(done)
			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					return
				}

				var frame struct {
					Type string `json:"type"`
					Data string `json:"data"`
				}
				if json.Unmarshal(message, &frame) == nil {
					switch frame.Type {
					case "stdout", "stderr":
						os.Stdout.Write([]byte(frame.Data))
					case "error":
						fmt.Fprintf(os.Stderr, "\r\nShell error: %s\r\n", frame.Data)
					case "low_balance":
						fmt.Fprintf(os.Stderr, "\r\n[warning] Low balance\r\n")
					case "deposit_required":
						fmt.Fprintf(os.Stderr, "\r\n[warning] Deposit required\r\n")
					}
				} else {
					os.Stdout.Write(message)
				}
			}
		}()

		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := os.Stdin.Read(buf)
				if err != nil {
					return
				}
				msg, _ := json.Marshal(map[string]string{
					"type": "input",
					"data": string(buf[:n]),
				})
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			}
		}()

		go func() {
			for sig := range sigCh {
				if handleShellSignal(sig, sendResize, conn) {
					return
				}
			}
		}()

		<-done
		fmt.Fprintf(os.Stderr, "\r\nConnection closed.\r\n")
	},
}

func init() {
	rootCmd.AddCommand(shellCmd)
}
