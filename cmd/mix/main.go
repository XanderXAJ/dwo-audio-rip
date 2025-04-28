package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"
)

type CliConfig struct {
	InputDirectory string
	OutputDirectory string
}

func main() {
	cliConfig := CliConfig{}

	flag.StringVar(&cliConfig.InputDirectory, "i", "", "Path to the directory containing the input files")
	flag.StringVar(&cliConfig.OutputDirectory, "o", "", "Path to the directory to save the output files")
	flag.Parse()

	if cliConfig.InputDirectory == "" {
		flag.Usage()
		return
	}

	// Process the input directory
	processAll(&cliConfig)
}

type TrackFiles struct {
	TrackNo      int
	AllFiles     TrackFileList
	FilesByTrait map[TrackTrait]*TrackFile
	FilesByType  map[string]TrackFileList
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

func (t *TrackFiles) SortedFiles() TrackFileList {
	sort.Sort(t.AllFiles)

	return t.AllFiles
}

type TrackFile struct {
	ChannelNo int
	Extension string
	FileName  string
	FilePath  string
	TrackNo   int
	Type      string
}

type TrackFileList []*TrackFile

func (t TrackFileList) Len() int {
	return len(t)
}

func (t TrackFileList) Less(i, j int) bool {
	// Sort first: lower track > lower channel > type (intro > loop > oneshot)
	return t[i].TrackNo < t[j].TrackNo || t[i].ChannelNo < t[j].ChannelNo || t[i].Type < t[j].Type
}

func (t TrackFileList) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
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
		FilesByType:  make(map[string]TrackFileList),
	}
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

func processAll(cliConfig *CliConfig) error {
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
		processTrack(cliConfig, trackFile)
	}

	return nil
}

func processTrack(cliConfig *CliConfig, track *TrackFiles) error {
	// Process the track based on the number of subchannels and file types available
	fmt.Printf("Processing track %d with %d channels: %v\n", track.TrackNo, track.NoOfChannels(), track.FilesByType)

	if track.NoOfChannels() == 12 {
		return mix12ChannelTrack(cliConfig, track)
	} else if track.NoOfChannels() == 2 {
		return mixStereoTrack(track)
	}
	return nil
}

func mix12ChannelTrack(cliConfig *CliConfig, track *TrackFiles) error {
	fmt.Printf("Mixing 12-channel track: %v\n", track.FilesByType)
	ffmpegArgs := make([]string, 0, 50)

	ffmpegArgs = append(ffmpegArgs, "-y")

	// Create ffmpeg input args for the stems as per test-mix-complete.ps1
	for _, file := range track.SortedFiles() {
		// Ignore 00 channels in 12-channel tracks as they are silent
		if file.ChannelNo == 0 {
			continue
		}
		ffmpegArgs = append(ffmpegArgs, "-i", file.FilePath)
	}

	// Add filter that mixes the tracks together
	// TODO: Consider the use of strings.Builder to build the complex filter for ease of maintenance
	ffmpegArgs = append(ffmpegArgs, "-filter_complex", fmt.Sprintf(`
		[0][1][2][3][4]amix=inputs=5[intro];
		[5][6][7][8][9]amix=inputs=5[loop];
		[loop]aloop=loop=%d:size=2e9[loops];
		[intro][loops]concat=v=0:a=1;
		`, 2))

	// Add output file name
	outputPath := path.Join(cliConfig.OutputDirectory, fmt.Sprintf("%d_mix.flac", track.TrackNo))
	ffmpegArgs = append(ffmpegArgs, outputPath)

	fmt.Printf("ffmpeg command: %v\n", ffmpegArgs)

	cmd := exec.Command("ffmpeg", ffmpegArgs...)
	return cmd.Run()
}

func mixStereoTrack(track *TrackFiles) error {
	fmt.Printf("Mixing stereo track: %v\n", track.FilesByType)
	return nil
}
