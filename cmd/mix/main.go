package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

type CliConfig struct {
	InputDirectory string
}

func main() {
	cliConfig := CliConfig{}

	flag.StringVar(&cliConfig.InputDirectory, "i", "", "Path to the directory containing the input files")
	flag.Parse()

	if cliConfig.InputDirectory == "" {
		flag.Usage()
		return
	}

	// Process the input directory
	processAll(cliConfig)
}

type TrackFiles struct {
	TrackNo      int
	AllFiles     []*TrackFile
	FilesByTrait map[TrackTrait]*TrackFile
	FilesByType  map[string][]*TrackFile
}

type TrackTrait struct {
	ChannelNo int
	Type      string
}

func NewTrackFiles(trackNo int) *TrackFiles {
	return &TrackFiles{
		TrackNo:      trackNo,
		AllFiles:     make([]*TrackFile, 0),
		FilesByTrait: make(map[TrackTrait]*TrackFile),
		FilesByType:  make(map[string][]*TrackFile),
	}
}

func (t *TrackFiles) String() string {
	return fmt.Sprintf("Track %d: %v", t.TrackNo, t.FilesByType)
}

func (t *TrackFiles) AddFile(file TrackFile) {
	t.AllFiles = append(t.AllFiles, &file)

	trait := TrackTrait{
		ChannelNo: file.ChannelNo,
		Type:      file.Type,
	}
	t.FilesByTrait[trait] = &file

	t.FilesByType[file.Type] = append(t.FilesByType[file.Type], &file)
}

func (t *TrackFiles) NoOfChannels() int {
	// Return the channel count of the first type that exists: loop, oneshot
	// We assume every file is stereo
	if len(t.FilesByType["loop"]) > 0 {
		return len(t.FilesByType["loop"]) * 2
	} else if len(t.FilesByType["oneshot"]) > 0 {
		return len(t.FilesByType["oneshot"]) * 2
	}
	return 0
}

func (t *TrackFiles) SortedFiles() []string {
	sortedFiles := make([]string, 0, len(t.AllFiles))

	sortChannels := func(i, j int) bool {
		return t.[i].ChannelNo < files[j].ChannelNo
	}

	append(sortedFiles, sort.Slice(t.FilesByType["intro"], sortChannels))
	append(sortedFiles, sort.Slice(t.FilesByType["loop"], sortChannels))
	append(sortedFiles, sort.Slice(t.FilesByType["oneshot"], sortChannels))

	return sortedFiles
}

type TrackFile struct {
	ChannelNo int
	Extension string
	FileName  string
	FilePath  string
	TrackNo   int
	Type      string
}

func (t *TrackFile) String() string {
	return t.FileName
}

func trackFileFromFileName(filePath string) (*TrackFile, error) {
	fileName := path.Base(filePath)
	parts := strings.Split(fileName, "_")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid file name format: %s", fileName)
	}

	trackNo, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("error converting track number: %w", err)
	}
	channelNo, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("error converting channel number: %w", err)
	}

	parts = strings.Split(parts[2], ".")
	fileType := parts[0]
	extension := parts[1]

	return &TrackFile{
		ChannelNo: channelNo,
		Extension: extension,
		FileName:  fileName,
		FilePath:  filePath,
		TrackNo:   trackNo,
		Type:      fileType,
	}, nil
}

func processAll(cliConfig CliConfig) error {
	fmt.Println("Processing input directory:", cliConfig.InputDirectory)

	/*
		Every file is listed in the following format:
		<trackNo>_<subChannelNo>_<intro|loop|oneshot>.wav

		How we process a track is determined by:
		- The number of subchannels files a track has;
		- The type of the files available.

		First, collate all of the files in InputDirectory and group them by track number.
		Then, for each track, check the number of subchannels and the types of files available.

		We'll assume that every file within a track and type is the same duration, sample rate, etc.
	*/
	// Map to group files by track number
	allTracks := make(map[int]*TrackFiles)

	// Read the directory and group files by track number
	files, err := os.ReadDir(cliConfig.InputDirectory)
	if err != nil {
		return fmt.Errorf("error reading input directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Parse the file name
		filePath := path.Join(cliConfig.InputDirectory, file.Name())
		trackFile, err := trackFileFromFileName(filePath)
		if err != nil {
			fmt.Printf("skipping invalid file %s: %v\n", filePath, err)
			continue
		}

		if allTracks[trackFile.TrackNo] == nil {
			allTracks[trackFile.TrackNo] = NewTrackFiles(trackFile.TrackNo)
		}
		allTracks[trackFile.TrackNo].AddFile(*trackFile)
	}

	// Process every track
	for _, trackFile := range allTracks {
		processTrack(trackFile)
	}

	return nil
}

func processTrack(track *TrackFiles) error {
	// Process the track based on the number of subchannels and file types available
	fmt.Printf("Processing track %d with %d channels: %v\n", track.TrackNo, track.NoOfChannels(), track.FilesByType)

	if track.NoOfChannels() == 12 {
		return mix12ChannelTrack(track)
	} else if track.NoOfChannels() == 2 {
		return mixStereoTrack(track)
	}
	return nil
}

func mix12ChannelTrack(track *TrackFiles) error {
	fmt.Printf("Mixing 12-channel track: %v\n", track.FilesByType)
	var ffmpegArgs []string
	// TODO: Get all stems, ensuring they're in the correct (predictable) order
	for _, file := range track.SortedFiles() {
		ffmpegArgs = append(ffmpegArgs, "-i", file)
	}
	fmt.Printf("FFmpeg args: %v\n", ffmpegArgs)

	// TODO: Create ffmpeg input args for the stems as per test-mix-complete.ps1
	// TODO: Append into slice to be able to pass to exec.Command()
	// TODO: Complete the rest of the ffmpeg command
	// TODO: Consider the use of strings.Builder to build the complex filter
	return nil
}

func mixStereoTrack(track *TrackFiles) error {
	fmt.Printf("Mixing stereo track: %v\n", track.FilesByType)
	return nil
}
