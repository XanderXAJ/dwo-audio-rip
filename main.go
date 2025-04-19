package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
)

// This code assumes you've already extracted the data files,
// e.g. 0xeda82cc0.srsa.

type VgmStreamInfo struct {
	Channels   int       `json:"channels"`
	StreamInfo StreamInfo `json:"streamInfo"`
}

type StreamInfo struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	Total int    `json:"total"`
}

func infoFromStream(path string, streamIndex int) (*VgmStreamInfo, error) {
	// Gather JSON stream information
	var stdout bytes.Buffer
	metadataCmd := exec.Command("vgmstream-cli", "-I", "-m", path, "-s", strconv.Itoa(streamIndex))
	metadataCmd.Stdout = &stdout
	metadataCmd.Stderr = os.Stderr
	if err := metadataCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to extract stream info: %w", err)
	}

	// Parse JSON stream information
	var si VgmStreamInfo
	if err := json.Unmarshal(stdout.Bytes(), &si); err != nil {
		return nil, fmt.Errorf("failed to unmarshall stream JSON: %w", err)
	}

	return &si, nil
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
		fmt.Println("Extracts audio streams from the given file, e.g. 0xeda82cc0.srsa")
		os.Exit(1)
	}
	path := path.Clean(os.Args[1])

	containerInfo, err := infoFromStream(path, 0)
	if err != nil {
		panic(err)
	}

	fmt.Println("Stream total:", containerInfo.StreamInfo.Total)

	for streamIndex := 1; streamIndex <= containerInfo.StreamInfo.Total; streamIndex++ {

		streamInfo, err := infoFromStream(path, streamIndex)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Stream %d: %d channels\n", streamIndex, streamInfo.Channels)

		// Convert each substream by of its pair of stereo channels
		// This assumes everything has at least two channels. Since I'm targeting music, this is acceptable.
		for channelIndex := 0; channelIndex <= streamInfo.Channels-2; channelIndex += 2 {
			err := convertSubstreamStereoPair(path, streamIndex, channelIndex)
			if err != nil {
				panic(err)
			}
		}
	}
}
