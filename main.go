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

const VgmstreamCLI = "vgmstream-cli"

// This code assumes you've already extracted the data files,
// e.g. 0xeda82cc0.srsa.

type VgmStreamInfo struct {
	Channels   int        `json:"channels"`
	StreamInfo StreamInfo `json:"streamInfo"`
}

type StreamInfo struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	Total int    `json:"total"`
}

// TODO: Create bad_streams list to ignore bad streams for specific filenames
// e.g. streams 1 and 48 in 0xeda82cc0.srsa

func infoFromStream(config *CliConfig, streamIndex int) (*VgmStreamInfo, error) {
	// Gather JSON stream information
	var stdout bytes.Buffer
	metadataCmd := exec.Command(VgmstreamCLI, "-I", "-m", config.InputPath, "-s", strconv.Itoa(streamIndex))
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

func convertSubstreamStereoStemAIO(config *CliConfig, streamIndex int, channelIndex int) error {
	metadataCmd := exec.Command(VgmstreamCLI, config.InputPath,
		"-s", strconv.Itoa(streamIndex),
		"-2", strconv.Itoa(channelIndex),
		"-o", path.Join(config.OutputPath, fmt.Sprintf("?s_%02d_aio.wav", channelIndex)),
	)
	return metadataCmd.Run()
}

func convertSubstreamStereoStemIntro(config *CliConfig, streamIndex int, channelIndex int) error {
	metadataCmd := exec.Command(VgmstreamCLI, config.InputPath,
		"-s", strconv.Itoa(streamIndex),
		"-2", strconv.Itoa(channelIndex),
		"-w",       // Convert in the original sample format
		"-l", "-1", // Remove looping section
		"-f", "0", // Remove fade out
		"-o", path.Join(config.OutputPath, fmt.Sprintf("?s_%02d_intro.wav", channelIndex)),
	)
	return metadataCmd.Run()
}

func convertSubstreamStereoStemLoop(config *CliConfig, streamIndex int, channelIndex int) error {
	metadataCmd := exec.Command(VgmstreamCLI, config.InputPath,
		"-s", strconv.Itoa(streamIndex),
		"-2", strconv.Itoa(channelIndex),
		"-w", // Convert in the original sample format
		"-o", path.Join(config.OutputPath, fmt.Sprintf("?s_%02d_loop.wav", channelIndex)),
		"-k", "-2", // Remove intro section
		"-l", "1", // One loop
		"-f", "0", // Remove fade out
	)
	return metadataCmd.Run()
}

type CliConfig struct {
	AioEnabled   bool
	IntroEnabled bool
	LoopEnabled  bool

	InputPath  string
	OutputPath  string
}

// TODO: Detect one-shot tracks and export them with a specific file name
func main() {
	cliConfig := &CliConfig{}

	flag.BoolVar(&cliConfig.AioEnabled, "aio", false, "Output all-in-one song loop stems")
	flag.BoolVar(&cliConfig.IntroEnabled, "intro", true, "Output intro stems")
	flag.BoolVar(&cliConfig.LoopEnabled, "loop", true, "Output loop stems")

	flag.StringVar(&cliConfig.InputPath, "i", "", "Input path for the container file to extract, e.g. 0xeda82cc0.srsa")
	flag.StringVar(&cliConfig.OutputPath, "o", "", "Output path for the extracted files")

	flag.Parse()

	if cliConfig.InputPath == "" {
		fmt.Printf("Usage: %s -i <path>\n", os.Args[0])
		fmt.Println("Extracts audio streams from the given file, e.g. 0xeda82cc0.srsa")
		flag.PrintDefaults()
		os.Exit(1)
	}

	containerInfo, err := infoFromStream(cliConfig, 0)
	if err != nil {
		panic(err)
	}

	fmt.Println("Stream total:", containerInfo.StreamInfo.Total)

	for streamIndex := 1; streamIndex <= containerInfo.StreamInfo.Total; streamIndex++ {

		streamInfo, err := infoFromStream(cliConfig, streamIndex)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Stream %d: %d channels\n", streamIndex, streamInfo.Channels)

		// Convert each substream by of its pair of stereo channels
		// This assumes everything has at least two channels. Since I'm targeting music, this is acceptable.
		for channelIndex := 0; channelIndex <= streamInfo.Channels-2; channelIndex += 2 {
			if cliConfig.AioEnabled {
				fmt.Printf("Exporting AIO stem for channel pair %d\n", channelIndex)
				err := convertSubstreamStereoStemAIO(cliConfig, streamIndex, channelIndex)
				if err != nil {
					panic(err)
				}
			}
			if cliConfig.IntroEnabled {
				fmt.Printf("Exporting intro stem for channel pair %d\n", channelIndex)
				err := convertSubstreamStereoStemIntro(cliConfig, streamIndex, channelIndex)
				if err != nil {
					fmt.Printf("Failed to export intro stem for channel pair %d: %v\n", channelIndex, err)
				}
			}
			if cliConfig.LoopEnabled {
				fmt.Printf("Exporting loop stem for channel pair %d\n", channelIndex)
				err := convertSubstreamStereoStemLoop(cliConfig, streamIndex, channelIndex)
				if err != nil {
					fmt.Printf("Failed to export loop stem for channel pair %d: %v\n", channelIndex, err)
				}
			}
		}
	}
}
