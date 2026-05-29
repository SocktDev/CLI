package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/SocktDev/CLI/internal/api"
)

const (
	waitPollInterval       = 2 * time.Second
	waitStatusMaxAttempts  = 60
	waitDataPlaneAttempts  = 30
	hostRetryAttempts      = 3
	hostRetryDelay         = 2 * time.Second
)

func fetchSandboxStatus(client *api.Client, sandboxID string) (api.SandboxStatus, error) {
	body, err := client.Get("/sandboxes/"+sandboxID, nil)
	if err != nil {
		return api.SandboxStatus{}, err
	}
	var status api.SandboxStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return api.SandboxStatus{}, fmt.Errorf("parse status: %w", err)
	}
	return status, nil
}

func requireSandboxOperational(client *api.Client, sandboxID string) {
	status, err := fetchSandboxStatus(client, sandboxID)
	if err != nil {
		exitError("%v", err)
	}
	requireStatusOperational(status, sandboxID)
}

func requireStatusOperational(status api.SandboxStatus, sandboxID string) {
	switch status.Status {
	case "running":
		return
	case "paused":
		exitError("sandbox %s is paused; run: sockt sandbox resume %s", sandboxID, sandboxID)
	case "terminated":
		exitError("sandbox %s is terminated", sandboxID)
	case "awaiting_payment":
		exitError("sandbox %s is awaiting payment; pay the invoice from create output, then poll status", sandboxID)
	case "failed":
		exitError("sandbox %s failed", sandboxID)
	default:
		exitError("sandbox %s is not runnable (status: %s)", sandboxID, status.Status)
	}
}

func probeDataPlane(client *api.Client, sandboxID string) error {
	_, err := client.Get("/sandboxes/"+sandboxID+"/files", map[string]string{"path": "."})
	return err
}

func isRetryableHostError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "host_error") ||
		strings.Contains(msg, "pod_starting") ||
		strings.Contains(msg, "pod unreachable") ||
		strings.Contains(msg, "503")
}

func withHostRetry(fn func() error) error {
	var last error
	for attempt := 0; attempt < hostRetryAttempts; attempt++ {
		last = fn()
		if last == nil || !isRetryableHostError(last) {
			return last
		}
		time.Sleep(hostRetryDelay)
	}
	return last
}

// waitSandboxReady blocks until status is running and the file API responds (data plane up).
func waitSandboxReady(client *api.Client, sandboxID string, alreadyRunning bool) {
	if !alreadyRunning {
		fmt.Fprintf(os.Stderr, "Waiting for sandbox to start...")
		for i := 0; i < waitStatusMaxAttempts; i++ {
			status, err := fetchSandboxStatus(client, sandboxID)
			if err != nil {
				exitError("%v", err)
			}
			if status.Status == "running" {
				fmt.Fprintf(os.Stderr, " running.")
				alreadyRunning = true
				break
			}
			if status.Status == "failed" || status.Status == "terminated" {
				exitError("sandbox %s", status.Status)
			}
			time.Sleep(waitPollInterval)
			fmt.Fprintf(os.Stderr, ".")
		}
		if !alreadyRunning {
			exitError("timeout waiting for sandbox to reach running state")
		}
	}

	fmt.Fprintf(os.Stderr, " Waiting for data plane...")
	for i := 0; i < waitDataPlaneAttempts; i++ {
		if err := probeDataPlane(client, sandboxID); err == nil {
			fmt.Fprintf(os.Stderr, " ready!\n")
			return
		}
		time.Sleep(waitPollInterval)
		fmt.Fprintf(os.Stderr, ".")
	}
	exitError("timeout waiting for sandbox data plane (files/exec not reachable yet)")
}
