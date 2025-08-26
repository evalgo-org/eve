package common

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func ShellExecute(cmdToRun string) {
	cmd := exec.Command("bash", "-c", cmdToRun)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		Logger.Fatal("Error:", err, "\nStderr:", stderr.String(), "\n")
	}
	Logger.Info("Output:\n", out.String(), "\n")
}

func ShellSudoExecute(password, cmdToRun string) {
	// Prepare the command: echo password | sudo -S <command>
	ShellExecute(fmt.Sprintf("echo %s | sudo -S %s", password, cmdToRun))
}

func URLToFilePath(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	return strings.ReplaceAll(url, "/", "_")
}
