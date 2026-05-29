//go:build windows

package cmd

import (
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
)

func notifyShellSignals(ch chan os.Signal) {
	signal.Notify(ch, os.Interrupt)
}

func handleShellSignal(sig os.Signal, sendResize func(), conn *websocket.Conn) bool {
	if sig == os.Interrupt {
		conn.Close()
		return true
	}
	return false
}
