package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
)

// OutputMode represents the output mode for mixing
type OutputMode int

const (
	ModeFull OutputMode = iota
	ModeIntroOnly
	ModeLoopOnly
)

// String returns the string representation of the OutputMode
func (m OutputMode) String() string {
	switch m {
	case ModeFull:
		return "full"
	case ModeIntroOnly:
		return "intro-only"
	case ModeLoopOnly:
		return "loop-only"
	default:
		return "unknown"
	}
}

// ParseOutputMode converts a string to an OutputMode
func ParseOutputMode(s string) (OutputMode, error) {
	switch s {
	case "full":
		return ModeFull, nil
	case "intro-only":
		return ModeIntroOnly, nil
	case "loop-only":
		return ModeLoopOnly, nil
	default:
		return ModeFull, fmt.Errorf("invalid mode: %s", s)
	}
}

type CliConfig struct {
	Duration        float64
	FadeDuration    float64
	InputDirectory  string
	OutputDirectory string
	Loops           int
	Mode            OutputMode
	Workers         int
	Verbose         bool
}

func (c *CliConfig) Validate() error {
	if c.InputDirectory == "" {
		return fmt.Errorf("input directory is required")
	}

	// For intro-only and loop-only, duration/loops are not required
	switch c.Mode {
	case ModeFull:
		// Require either Duration or Loops to be set for full mode
		if c.Duration <= 0 && c.Loops <= 0 {
			return fmt.Errorf("either -duration or -loops must be set to positive numbers")
		}
	}

	return nil
}

func main() {
	cliConfig := CliConfig{}

	const (
		inputDefault   = ""
		inputUsage     = "Path to the directory containing the input files"
		outputDefault  = ""
		outputUsage    = "Path to the directory to save the output files"
		loopsDefault   = 2
		loopsUsage     = "Number of loops for the intro"
		verboseDefault = false
		verboseUsage   = "Enable verbose logging"
		workersDefault = -1
		workersUsage   = "Number of workers to use for processing"
	)

	flag.Float64Var(&cliConfig.Duration, "duration", -1, "Minimum duration (in seconds) for mixed tracks. Takes precedence over -loops. Only used in 'full' mode.")
	flag.Float64Var(&cliConfig.FadeDuration, "fade", 15.0, "Fade out duration at the end of tracks")
	flag.StringVar(&cliConfig.InputDirectory, "i", inputDefault, inputUsage)
	flag.StringVar(&cliConfig.InputDirectory, "input", inputDefault, inputUsage)
	flag.StringVar(&cliConfig.OutputDirectory, "o", outputDefault, outputUsage)
	flag.StringVar(&cliConfig.OutputDirectory, "output", outputDefault, outputUsage)
	flag.IntVar(&cliConfig.Loops, "l", loopsDefault, loopsUsage)
	flag.IntVar(&cliConfig.Loops, "loops", loopsDefault, loopsUsage)
	mode := flag.String("mode", "full", "Output mode: 'full' (intro+loops), 'intro-only', or 'loop-only'")
	flag.IntVar(&cliConfig.Workers, "w", -1, workersUsage)
	flag.IntVar(&cliConfig.Workers, "workers", -1, workersUsage)
	flag.BoolVar(&cliConfig.Verbose, "v", verboseDefault, verboseUsage)
	flag.BoolVar(&cliConfig.Verbose, "verbose", verboseDefault, verboseUsage)
	flag.Parse()

	// Parse the mode string into the enum
	var err error
	cliConfig.Mode, err = ParseOutputMode(*mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		return
	}

	if err := cliConfig.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		return
	}

	// Process the input directory
	processAll(&cliConfig)
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

	// Verbosely print all processed TrackFiles
	if cliConfig.Verbose {
		fmt.Println("All processed TrackFiles:")
		for _, track := range allTracks {
			fmt.Printf("Track %d:\n", track.TrackNo)
			for _, file := range track.SortedFiles() {
				fmt.Printf("  %+v\n", *file)
			}
		}
	}

	// Create workers for processing tracks
	numWorkers := cliConfig.Workers
	if numWorkers < 0 {
		numWorkers = runtime.NumCPU()
	}
	trackChan := make(chan *TrackFiles, len(allTracks))
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(cliConfig, trackChan, &wg)
	}

	// Enqueue processing for every track
	for _, trackFile := range allTracks {
		trackChan <- trackFile
	}
	close(trackChan)

	// Wait for all workers to finish
	wg.Wait()

	return nil
}

func worker(cliConfig *CliConfig, trackChan chan *TrackFiles, wg *sync.WaitGroup) {
	defer wg.Done()
	for track := range trackChan {
		err := processTrack(cliConfig, track)
		if err != nil {
			fmt.Printf("Error processing track %d: %v\n", track.TrackNo, err)
		}
	}
}

func processTrack(cliConfig *CliConfig, track *TrackFiles) error {
	// Process the track based on the number of subchannels and file types available
	fmt.Printf("Processing track %d with %d channels: %v\n", track.TrackNo, track.NoOfChannels(), track.FilesByType)

	if track.NoOfChannels() == 12 {
		return mix12ChannelTrack(cliConfig, track)
	} else if track.NoOfChannels() == 2 {
		return mixStereoTrack(cliConfig, track)
	}
	return nil
}

func mix12ChannelTrack(cliConfig *CliConfig, track *TrackFiles) error {
	fmt.Printf("Mixing 12-channel track: %v\n", track.FilesByType)

	// Ensure output directory exists
	if err := os.MkdirAll(cliConfig.OutputDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	ffmpegArgs := make([]string, 0, 50)
	ffmpegArgs = append(ffmpegArgs, "-y")

	// Create ffmpeg input args for the stems as per test-mix-complete.ps1
	// Filter files based on mode
	for _, file := range track.SortedFiles() {
		// Ignore 00 channels in 12-channel tracks as they are silent
		if file.ChannelNo == 0 {
			continue
		}

		// Filter by mode
		switch cliConfig.Mode {
		case ModeIntroOnly:
			if file.Type != "intro" {
				continue
			}
		case ModeLoopOnly:
			if file.Type != "loop" && file.Type != "oneshot" {
				continue
			}
		case ModeFull:
			// Include all files for full mode
		}

		// We either have a oneshot track, or a loop with an optional intro.
		// Loops and oneshots are effectively the same, except oneshots don't loop.
		switch cliConfig.Mode {
		case ModeFull:
			if file.Type == "loop" {
				ffmpegArgs = append(ffmpegArgs, "-stream_loop", "-1")
			}
		case ModeIntroOnly, ModeLoopOnly:
			// No looping in these modes
		}
		ffmpegArgs = append(ffmpegArgs, "-i", file.FilePath)
	}

	//// Assemble -filter_complex
	// Add filter that mixes the tracks together.
	// Stems are pre-normalised, so no need to normalise again here.
	filter := strings.Builder{}

	// Set up inputs based on mode
	switch cliConfig.Mode {
	case ModeIntroOnly:
		// Only intro files: [0][1][2][3][4]amix=inputs=5:normalize=0[intro];
		filter.WriteString("[0][1][2][3][4]amix=inputs=5:normalize=0[intro];")
	case ModeLoopOnly:
		// Only loop files: [0][1][2][3][4]amix=inputs=5:normalize=0[loop];
		filter.WriteString("[0][1][2][3][4]amix=inputs=5:normalize=0[loop];")
	case ModeFull:
		// Full mode: both intro and loop
		filter.WriteString("[0][1][2][3][4]amix=inputs=5:normalize=0[intro];")
		filter.WriteString("[5][6][7][8][9]amix=inputs=5:normalize=0[loop];")
	}

	trackFilter, err := generateTrackFilter(track, cliConfig)
	if err != nil {
		return err
	}
	filter.WriteString(trackFilter)

	ffmpegArgs = append(ffmpegArgs, "-filter_complex", filter.String())

	// Add output file name with mode suffix
	ffmpegArgs = append(ffmpegArgs, "-compression_level", "12")
	outputSuffix := ""
	switch cliConfig.Mode {
	case ModeIntroOnly:
		outputSuffix = "_intro"
	case ModeLoopOnly:
		outputSuffix = "_loop"
	case ModeFull:
		// No suffix for full mode
	}
	outputPath := path.Join(cliConfig.OutputDirectory, fmt.Sprintf("%d_mix%s.flac", track.TrackNo, outputSuffix))
	ffmpegArgs = append(ffmpegArgs, outputPath)

	fmt.Printf("ffmpeg command: %v\n", ffmpegArgs)

	cmd := exec.Command("ffmpeg", ffmpegArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func mixStereoTrack(cliConfig *CliConfig, track *TrackFiles) error {
	fmt.Printf("Mixing stereo track: %v\n", track.FilesByType)

	// Ensure output directory exists
	if err := os.MkdirAll(cliConfig.OutputDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	ffmpegArgs := make([]string, 0, 20)
	ffmpegArgs = append(ffmpegArgs, "-y")

	_, hasIntro := track.FilesByType["intro"]

	// Create ffmpeg input args for the stems
	// Filter files based on mode
	for _, file := range track.SortedFiles() {
		// Filter by mode
		switch cliConfig.Mode {
		case ModeIntroOnly:
			if file.Type != "intro" {
				continue
			}
		case ModeLoopOnly:
			if file.Type != "loop" && file.Type != "oneshot" {
				continue
			}
		case ModeFull:
			// Include all files for full mode
		}

		// We either have a oneshot track, or a loop with an optional intro.
		// Loops and oneshots are effectively the same, except oneshots don't loop.
		switch cliConfig.Mode {
		case ModeFull:
			if file.Type == "loop" {
				ffmpegArgs = append(ffmpegArgs, "-stream_loop", "-1")
			}
		case ModeIntroOnly, ModeLoopOnly:
			// No looping in these modes
		}
		ffmpegArgs = append(ffmpegArgs, "-i", file.FilePath)
	}

	//// Assemble -filter_complex
	// Set up inputs
	filter := strings.Builder{}
	switch cliConfig.Mode {
	case ModeIntroOnly:
		filter.WriteString("[0]anull[intro];")
	case ModeLoopOnly:
		filter.WriteString("[0]anull[loop];")
	case ModeFull:
		// full mode
		if hasIntro {
			filter.WriteString("[0]anull[intro];")
			filter.WriteString("[1]anull[loop];")
		} else {
			filter.WriteString("[0]anull[loop];")
		}
	}

	trackFilter, err := generateTrackFilter(track, cliConfig)
	if err != nil {
		return err
	}
	filter.WriteString(trackFilter)

	ffmpegArgs = append(ffmpegArgs, "-filter_complex", filter.String())

	// Add output file name with mode suffix
	ffmpegArgs = append(ffmpegArgs, "-compression_level", "12")
	outputSuffix := ""
	switch cliConfig.Mode {
	case ModeIntroOnly:
		outputSuffix = "_intro"
	case ModeLoopOnly:
		outputSuffix = "_loop"
	case ModeFull:
		// No suffix for full mode
	}
	outputPath := path.Join(cliConfig.OutputDirectory, fmt.Sprintf("%d_mix%s.flac", track.TrackNo, outputSuffix))
	ffmpegArgs = append(ffmpegArgs, outputPath)

	fmt.Printf("ffmpeg command: %v\n", ffmpegArgs)

	cmd := exec.Command("ffmpeg", ffmpegArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Assumes that [loop] and [intro] (if applicable) labels are defined.
func generateTrackFilter(track *TrackFiles, cliConfig *CliConfig) (string, error) {
	filter := strings.Builder{}

	switch cliConfig.Mode {
	case ModeIntroOnly:
		if !track.HasIntro() {
			return "", fmt.Errorf("intro-only mode selected but track has no intro")
		}
		filter.WriteString("[intro]anull;") // Sink intro label to output to avoid error
		return filter.String(), nil

	case ModeLoopOnly:
		if !track.HasLoop() && !track.HasOneshot() {
			return "", fmt.Errorf("loop-only mode selected but track has no loop or oneshot")
		}
		// For loop-only, just output one loop with no fade
		filter.WriteString("[loop]anull;") // Sink loop label to output to avoid error
		return filter.String(), nil

	case ModeFull:
		// Handle full mode
		if !track.HasOneshot() {
			loopPlan, err := planLoops(cliConfig, *track)
			if err != nil {
				return "", fmt.Errorf("failed to plan loops for track %d: %w", track.TrackNo, err)
			}
			filter.WriteString(generateLoopFadeFilters(loopPlan))

			if track.HasIntro() {
				filter.WriteString("[intro][body][fade]concat=n=3:v=0:a=1;")
			} else {
				filter.WriteString("[body][fade]concat=v=0:a=1;")
			}
		} else {
			if track.HasIntro() {
				filter.WriteString("[intro][loop]concat=v=0:a=1;")
			} else {
				filter.WriteString("[loop]anull;") // Sink hanging loop label to output to avoid error
			}
		}
		return filter.String(), nil
	}

	return "", fmt.Errorf("unknown mode")
}

func generateLoopFadeFilters(loopPlan *LoopPlan) string {
	return fmt.Sprintf(`
		[loop]asplit=2[w0][w1];
		[w0]atrim=0:%.9f,asetpts=N/SR/TB[body];
		[w1]atrim=%.9f:%.9f,asetpts=N/SR/TB,afade=t=out:st=0:d=%.9f[fade];
		`, loopPlan.LoopEnd, loopPlan.LoopEnd, loopPlan.FadeEnd, loopPlan.FadeEnd-loopPlan.LoopEnd)
}
