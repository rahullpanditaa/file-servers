package main

import (
	"fmt"
	"os/exec"
)

func processVideoForFastStart(filePath string) (string, error) {
	outputFilePath := filePath + ".processing"

	cmd := exec.Command(
		"ffmpeg",
		"-i",
		filePath,
		"-c",
		"copy",
		"-movflags",
		"faststart", "-f", "mp4", outputFilePath)

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("unable to run command: %v", cmd.Args[0])
	}

	return outputFilePath, nil
}
