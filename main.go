package main

import (
	"bytes"
	"encoding/json"
	"flag"
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

func convertSubstreamStereoStem(path string, streamIndex int, channelIndex int) error {
	metadataCmd := exec.Command("vgmstream-cli", path,
		"-s", strconv.Itoa(streamIndex),
		"-2", strconv.Itoa(channelIndex),
		"-o", fmt.Sprintf("?s_%02d_aio.wav", channelIndex),
	)
	return metadataCmd.Run()
}

func convertSubstreamStereoStemIntro(path string, streamIndex int, channelIndex int) error {
	metadataCmd := exec.Command("vgmstream-cli", path,
		"-s", strconv.Itoa(streamIndex),
		"-2", strconv.Itoa(channelIndex),
		"-w", // Convert in the original sample format
		"-l", "-1", // Remove looping section
		"-f", "0", // Remove fade out
		"-o", fmt.Sprintf("?s_%02d_intro.wav", channelIndex),
	)
	return metadataCmd.Run()
}

func convertSubstreamStereoStemLoop(path string, streamIndex int, channelIndex int) error {
	metadataCmd := exec.Command("vgmstream-cli", path,
		"-s", strconv.Itoa(streamIndex),
		"-2", strconv.Itoa(channelIndex),
		"-w", // Convert in the original sample format
		"-o", fmt.Sprintf("?s_%02d_loop.wav", channelIndex),
		"-k", "-2", // Remove intro section
		"-l", "1", // One loop
		"-f", "0", // Remove fade out
	)
	return metadataCmd.Run()
}

func main() {
	var (
		aioEnabled bool
		introEnabled bool
		loopEnabled bool
	)
	flag.BoolVar(&aioEnabled, "aio", false, "Extract all-in-one song loop stems")
	flag.BoolVar(&introEnabled, "intro", true, "Extract intro stems")
	flag.BoolVar(&loopEnabled, "loop", true, "Extract loop stems")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Printf("Usage: %s <path>\n", os.Args[0])
		fmt.Println("Extracts audio streams from the given file, e.g. 0xeda82cc0.srsa")
		flag.PrintDefaults()
		os.Exit(1)
	}
	path := path.Clean(flag.Arg(0))

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
			if aioEnabled {
				fmt.Printf("Exporting AIO stem for channel pair %d\n", channelIndex)
				err := convertSubstreamStereoStem(path, streamIndex, channelIndex)
				if err != nil {
					panic(err)
				}
			}
			if introEnabled {
				fmt.Printf("Exporting intro stem for channel pair %d\n", channelIndex)
				err := convertSubstreamStereoStemIntro(path, streamIndex, channelIndex)
				if err != nil {
					fmt.Printf("Failed to export intro stem for channel pair %d: %v\n", channelIndex, err)
				}
			}
			if loopEnabled {
				fmt.Printf("Exporting loop stem for channel pair %d\n", channelIndex)
				err := convertSubstreamStereoStemLoop(path, streamIndex, channelIndex)
				if err != nil {
					fmt.Printf("Failed to export loop stem for channel pair %d: %v\n", channelIndex, err)
				}
			}
		}
	}
}
