package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command(
		"ffprobe",
		"-v",
		"error",
		"-print_format",
		"json",
		"-show_strems",
		filePath)

	b := bytes.Buffer{}
	cmd.Stdout = &b
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("unable to run command: %v", err)
	}

	s := struct {
		width  int
		height int
	}{}

	err = json.Unmarshal(b.Bytes(), &s)
	if err != nil {
		return "", fmt.Errorf("unable to unmarshal stdout to a struct")
	}

	aspectRatio := float64(s.width) / float64(s.height)

	switch {
	case aspectRatio >= 1.7 && aspectRatio <= 1.8:
		return "16:9", nil
	case aspectRatio >= 0.55 && aspectRatio <= 0.57:
		return "9:16", nil
	default:
		return "other", nil
	}

}
