//go:build unix

package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
)

func notifyShellSignals(ch chan os.Signal) {
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGWINCH)
}

func handleShellSignal(sig os.Signal, sendResize func(), conn *websocket.Conn) bool {
	switch sig {
	case syscall.SIGWINCH:
		sendResize()
		return false
	case syscall.SIGINT, syscall.SIGTERM:
		conn.Close()
		return true
	default:
		return false
	}
}
