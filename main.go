package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

// Get stream count from metadata
func streamCount(path string) (int, error) {
	var stdout bytes.Buffer
	metadataCmd := exec.Command("vgmstream-cli", "-m", path)
	metadataCmd.Stdout = &stdout
	metadataCmd.Stderr = os.Stderr
	if err := metadataCmd.Run(); err != nil {
		return -1, err
	}

	var streamCount int
	for _, line := range strings.Split(stdout.String(), "\n") {
		if strings.HasPrefix(line, "stream count:") {
			fmt.Sscanf(strings.TrimSpace(line), "stream count: %d", &streamCount)
			break
		}
	}

	return streamCount, nil
}

// Get channel count from metadata
func channels(path string, streamIndex int) (int, error) {
	var stdout bytes.Buffer
	metadataCmd := exec.Command("vgmstream-cli", "-m", path, "-s", strconv.Itoa(streamIndex))
	metadataCmd.Stdout = &stdout
	metadataCmd.Stderr = os.Stderr
	if err := metadataCmd.Run(); err != nil {
		return -1, err
	}

	var channelCount int
	for _, line := range strings.Split(stdout.String(), "\n") {
		if strings.HasPrefix(line, "channels:") {
			fmt.Sscanf(strings.TrimSpace(line), "channels: %d", &channelCount)
			break
		}
	}

	return channelCount, nil
}

func convertSubstreamStereoPair(path string, streamIndex int, channelIndex int) error {
	metadataCmd := exec.Command("vgmstream-cli", path,
		"-s", strconv.Itoa(streamIndex),
		"-2", strconv.Itoa(channelIndex),
		"-o", fmt.Sprintf("?f#?s_?n-%02d.wav", channelIndex),
	)
	return metadataCmd.Run()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <path>\n", os.Args[0])
		fmt.Println("Extracts audio streams from the given file")
		os.Exit(1)
	}
	path := path.Clean(os.Args[1])

	streamCount, err := streamCount(path)
	if err != nil {
		panic(err)
	}

	fmt.Println("Stream count:", streamCount)

	for streamIndex := 1; streamIndex <= streamCount; streamIndex++ {

		channels, err := channels(path, streamIndex)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Stream %d: %d channels\n", streamIndex, channels)

		// Convert each substream by of its pair of stereo channels
		// This assumes everything has at least two channels. Since I'm targeting music, this is acceptable.
		for channelIndex := 0; channelIndex <= channels - 2; channelIndex += 2 {
			err := convertSubstreamStereoPair(path, streamIndex, channelIndex)
			if err != nil {
				panic(err)
			}
		}
	}
}
