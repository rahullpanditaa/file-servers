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
		"-show_streams",
		filePath)

	b := bytes.Buffer{}
	cmd.Stdout = &b
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("unable to run command: %v", err)
	}

	var result struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	err = json.Unmarshal(b.Bytes(), &result)
	if err != nil {
		return "", fmt.Errorf("unable to unmarshal stdout to a struct")
	}

	aspectRatio := float64(result.Streams[0].Width) / float64(result.Streams[0].Height)

	switch {
	case aspectRatio >= 1.7 && aspectRatio <= 1.8:
		return "16:9", nil
	case aspectRatio >= 0.55 && aspectRatio <= 0.57:
		return "9:16", nil
	default:
		return "other", nil
	}

}
