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

type CliConfig struct {
	Duration        float64
	FadeDuration    float64
	InputDirectory  string
	OutputDirectory string
	Loops           int
	Workers         int
	Verbose         bool
}

func (c *CliConfig) Validate() error {
	if c.InputDirectory == "" {
		return fmt.Errorf("input directory is required")
	}

	// Require either Duration or Loops to be set
	if c.Duration <= 0 && c.Loops <= 0 {
		return fmt.Errorf("either -duration or -loops must be set to positive numbers")
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

	flag.Float64Var(&cliConfig.Duration, "duration", -1, "Minimum duration (in seconds) for mixed tracks. Takes precedence over -loops.")
	flag.Float64Var(&cliConfig.FadeDuration, "fade", 15.0, "Fade out duration at the end of tracks")
	flag.StringVar(&cliConfig.InputDirectory, "i", inputDefault, inputUsage)
	flag.StringVar(&cliConfig.InputDirectory, "input", inputDefault, inputUsage)
	flag.StringVar(&cliConfig.OutputDirectory, "o", outputDefault, outputUsage)
	flag.StringVar(&cliConfig.OutputDirectory, "output", outputDefault, outputUsage)
	flag.IntVar(&cliConfig.Loops, "l", loopsDefault, loopsUsage)
	flag.IntVar(&cliConfig.Loops, "loops", loopsDefault, loopsUsage)
	flag.IntVar(&cliConfig.Workers, "w", -1, workersUsage)
	flag.IntVar(&cliConfig.Workers, "workers", -1, workersUsage)
	flag.BoolVar(&cliConfig.Verbose, "v", verboseDefault, verboseUsage)
	flag.BoolVar(&cliConfig.Verbose, "verbose", verboseDefault, verboseUsage)
	flag.Parse()

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
	for _, file := range track.SortedFiles() {
		// Ignore 00 channels in 12-channel tracks as they are silent
		if file.ChannelNo == 0 {
			continue
		}
		// We either have a oneshot track, or a loop with an optional intro.
		// Loops and oneshots are effectively the same, except oneshots don't loop.
		if file.Type == "loop" {
			ffmpegArgs = append(ffmpegArgs, "-stream_loop", "-1")
		}
		ffmpegArgs = append(ffmpegArgs, "-i", file.FilePath)
	}

	//// Assemble -filter_complex
	// Add filter that mixes the tracks together.
	// Stems are pre-normalised, so no need to normalise again here.
	filter := strings.Builder{}

	// Set up inputs
	filter.WriteString("[0][1][2][3][4]amix=inputs=5:normalize=0[intro];")
	filter.WriteString("[5][6][7][8][9]amix=inputs=5:normalize=0[loop];")

	trackFilter, err := generateTrackFilter(track, cliConfig)
	if err != nil {
		return err
	}
	filter.WriteString(trackFilter)

	ffmpegArgs = append(ffmpegArgs, "-filter_complex", filter.String())

	// Add output file name
	outputPath := path.Join(cliConfig.OutputDirectory, fmt.Sprintf("%d_mix.flac", track.TrackNo))
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
	for _, file := range track.SortedFiles() {
		// We either have a oneshot track, or a loop with an optional intro.
		// Loops and oneshots are effectively the same, except oneshots don't loop.
		if file.Type == "loop" {
			ffmpegArgs = append(ffmpegArgs, "-stream_loop", "-1")
		}
		ffmpegArgs = append(ffmpegArgs, "-i", file.FilePath)
	}

	//// Assemble -filter_complex
	// Set up inputs
	filter := strings.Builder{}
	if hasIntro {
		filter.WriteString("[0]anull[intro];")
		filter.WriteString("[1]anull[loop];")
	} else {
		filter.WriteString("[0]anull[loop];")
	}

	trackFilter, err := generateTrackFilter(track, cliConfig)
	if err != nil {
		return err
	}
	filter.WriteString(trackFilter)

	ffmpegArgs = append(ffmpegArgs, "-filter_complex", filter.String())

	// Add output file name
	outputPath := path.Join(cliConfig.OutputDirectory, fmt.Sprintf("%d_mix.flac", track.TrackNo))
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

func generateLoopFadeFilters(loopPlan *LoopPlan) string {
	return fmt.Sprintf(`
		[loop]asplit=2[w0][w1];
		[w0]atrim=0:%.9f,asetpts=N/SR/TB[body];
		[w1]atrim=%.9f:%.9f,asetpts=N/SR/TB,afade=t=out:st=0:d=%.9f[fade];
		`, loopPlan.LoopEnd, loopPlan.LoopEnd, loopPlan.FadeEnd, loopPlan.FadeEnd-loopPlan.LoopEnd)
}
