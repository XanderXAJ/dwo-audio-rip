package main

import "fmt"

type LoopPlan struct {
	// Time, in seconds, when looping ends. Excludes intro time (if present).
	LoopEnd float64
	// Time, in seconds, when fade out ends. Comes after LoopEnd. Excludes intro time (if present).
	FadeEnd float64
}

func planLoops(cliConfig *CliConfig, track TrackFiles) (*LoopPlan, error) {
	loopPlan := LoopPlan{}

	// If the track is a oneshot, most properties are set:
	//   There's one loop and no fade.
	// Note: We're assuming there aren't also any loop files present...
	if track.HasOneshot() {
		loopPlan.LoopEnd = track.MainDurationSeconds()
		loopPlan.FadeEnd = loopPlan.LoopEnd
		return &loopPlan, nil
	}

	// Our total duration is determined by either the user-specified duration or
	// loops. Duration takes precedence.
	if cliConfig.Duration > 0 {
		targetDuration := cliConfig.Duration

		// Reduce target duration if intro is present -- we don't need to loop as long.
		if track.HasIntro() {
			targetDuration -= track.IntroDurationSeconds()
		}

		loopsNeeded := loopsForDuration(targetDuration, track.MainDurationSeconds())
		loopPlan.LoopEnd = float64(loopsNeeded) * track.MainDurationSeconds()
		loopPlan.FadeEnd = loopPlan.LoopEnd + cliConfig.FadeDuration
		return &loopPlan, nil
	}

	if cliConfig.Loops > 0 {
		loopPlan.LoopEnd = float64(cliConfig.Loops) * track.MainDurationSeconds()
		loopPlan.FadeEnd = loopPlan.LoopEnd + cliConfig.FadeDuration
		return &loopPlan, nil
	}

	return nil, fmt.Errorf("unable to plan loops: unexpected state")
}

// Returns loops required to _at least_ cover the duration.
// This means that loopDuration * loops >= duration.
func loopsForDuration(targetDuration float64, loopDuration float64) int {
	if loopDuration <= 0 {
		return 0
	}

	return int(targetDuration/loopDuration) + 1
}
