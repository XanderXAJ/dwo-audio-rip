package main

import (
	"context"
	"dwo-audio-rip/internal/probe"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
)

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

func (t *TrackFiles) HasIntro() bool {
	_, hasIntro := t.FilesByType["intro"]
	return hasIntro
}

func (t *TrackFiles) HasLoop() bool {
	_, hasLoop := t.FilesByType["loop"]
	return hasLoop
}

func (t *TrackFiles) HasOneshot() bool {
	_, hasOneshot := t.FilesByType["oneshot"]
	return hasOneshot
}

// Returns intro duration in seconds, or -1 if no intro.
func (t *TrackFiles) IntroDurationSeconds() float64 {
	intros, hasIntro := t.FilesByType["intro"]
	if !hasIntro || len(intros) == 0 {
		return -1
	}
	return intros[0].DurationSeconds
}

// Returns main duration in seconds (loop or oneshot), or -1 if neither exist.
func (t *TrackFiles) MainDurationSeconds() float64 {
	loops, hasLoop := t.FilesByType["loop"]
	oneshots, hasOneshot := t.FilesByType["oneshot"]

	if hasLoop && len(loops) > 0 {
		return loops[0].DurationSeconds
	} else if hasOneshot && len(oneshots) > 0 {
		return oneshots[0].DurationSeconds
	}
	return -1
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
	ChannelNo       int
	DurationSeconds float64
	Extension       string
	FileName        string
	FilePath        string
	TrackNo         int
	Type            string
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

	durationSeconds, err := probe.DurationSeconds(context.Background(), filePath)
	if err != nil {
		return nil, fmt.Errorf("error probing duration: %w", err)
	}

	return &TrackFile{
		ChannelNo:       channelNo,
		DurationSeconds: durationSeconds,
		Extension:       extension,
		FileName:        fileName,
		FilePath:        filePath,
		TrackNo:         trackNo,
		Type:            fileType,
	}, nil
}
