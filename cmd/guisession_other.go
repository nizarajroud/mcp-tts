//go:build !darwin

package cmd

import (
	"context"
	"errors"

	"github.com/charmbracelet/log"
)

var errGUIUnsupported = errors.New("GUI session routing is only supported on macOS")

// sessionMode classifies the launchd audio context of this process. On
// non-darwin platforms there is no launchd session, so the type exists only so
// shared code in root.go compiles; detection always reports sessionAqua.
type sessionMode int

const (
	sessionAqua sessionMode = iota
	sessionRoutable
	sessionHeadless
)

func currentSessionMode() (sessionMode, string) { return sessionAqua, "" }

// sessionModeFn mirrors the darwin seam so shared code in root.go compiles. On
// non-darwin platforms it always reports sessionAqua (in-process audio).
var sessionModeFn = currentSessionMode

func headlessAudioError(string) string { return "" }

func guiSayPlay(context.Context, []string, string, string, bool) error {
	return errGUIUnsupported
}

func startRoutableSayPlayback(_ []string, _, _ string) error { return errGUIUnsupported }

// routeCloudPlayback on non-darwin platforms checks for WSL and routes audio
// through Windows PowerShell if detected. Otherwise falls through to in-process
// beep library playback.
func routeCloudPlayback(label, ext string, build func() ([]byte, error)) (bool, error) {
	detectWSL()
	if !isWSL {
		return false, nil
	}

	audioData, err := build()
	if err != nil {
		return true, err
	}

	log.Debug("WSL playback route active", "label", label, "ext", ext, "bytes", len(audioData))

	switch ext {
	case "wav":
		// Audio is already a full WAV (with header) from the build func
		if err := wslPlayFullWAV(audioData); err != nil {
			return true, err
		}
	case "mp3":
		if err := wslPlayMP3(audioData); err != nil {
			return true, err
		}
	default:
		log.Warn("Unknown audio format for WSL playback, falling through", "ext", ext)
		return false, nil
	}

	return true, nil
}

func cleanupStaleGUIJobs() {}
