package probe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type FFProbeJSONOutput struct {
	Streams []struct {
		Duration string `json:"duration"`
	} `json:"streams"`
}

func DurationSeconds(ctx context.Context, filePath string) (float64, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=duration",
		"-of", "json=compact=1",
		filePath,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("ffprobe error: %w, stderr: %s", err, strings.TrimSpace(stderr.String()))
	}

	var ffprobeOutput FFProbeJSONOutput
	if err := json.Unmarshal(stdout.Bytes(), &ffprobeOutput); err != nil {
		return 0, fmt.Errorf("failed to parse ffprobe JSON output: %w", err)
	}

	if len(ffprobeOutput.Streams) == 0 {
		return 0, fmt.Errorf("no audio streams found in file: %s", filePath)
	}

	durationStr := ffprobeOutput.Streams[0].Duration
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration string: %w", err)
	}

	return duration, nil
}
