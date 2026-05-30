package cmd

import (
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

// voiceAvailability holds the cached set of installed voice names.
type voiceAvailability struct {
	mu     sync.RWMutex
	voices map[string]bool
}

var voiceCache voiceAvailability

// voiceLoader queries the installed voices. It is a package var so tests can
// supply a deterministic set without invoking `say -v?`.
var voiceLoader = loadInstalledVoices

// loadInstalledVoices runs `say -v?` and parses its output into the set of
// installed voice names. Each line begins with the voice name followed by a
// locale column, e.g.:
//
//	Albert                en_US    # Hello! My name is Albert.
//	Eddy (German (Germany)) de_DE  # Hallo! Ich heiße Eddy.
func loadInstalledVoices() (map[string]bool, error) {
	voices := make(map[string]bool)
	if runtime.GOOS != "darwin" {
		return voices, nil
	}

	out, err := exec.Command("/usr/bin/say", "-v?").Output()
	if err != nil {
		return nil, err
	}

	for line := range strings.SplitSeq(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// The voice name is every field before the locale column.
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		voiceNameParts := []string{}
		for i, part := range parts {
			if isLocale(part) {
				voiceNameParts = parts[:i]
				break
			}
		}
		if len(voiceNameParts) == 0 {
			voiceNameParts = parts[:1]
		}
		voices[strings.Join(voiceNameParts, " ")] = true
	}

	return voices, nil
}

// getInstalledVoices returns the cached installed-voice set, loading it on first
// use. Use refreshInstalledVoices to force a re-query.
func getInstalledVoices() (map[string]bool, error) {
	voiceCache.mu.RLock()
	cached := voiceCache.voices
	voiceCache.mu.RUnlock()
	if cached != nil {
		return cached, nil
	}
	return refreshInstalledVoices()
}

// refreshInstalledVoices re-queries the installed voices and replaces the cache.
func refreshInstalledVoices() (map[string]bool, error) {
	voices, err := voiceLoader()
	if err != nil {
		return nil, err
	}
	voiceCache.mu.Lock()
	voiceCache.voices = voices
	voiceCache.mu.Unlock()
	return voices, nil
}

// isLocale checks if a string looks like a locale code (e.g., en_US, de_DE)
func isLocale(s string) bool {
	if len(s) != 5 {
		return false
	}
	// Pattern: xx_XX where x is lowercase and X is uppercase
	return s[0] >= 'a' && s[0] <= 'z' &&
		s[1] >= 'a' && s[1] <= 'z' &&
		s[2] == '_' &&
		s[3] >= 'A' && s[3] <= 'Z' &&
		s[4] >= 'A' && s[4] <= 'Z'
}

// IsVoiceInstalled reports whether voiceName is installed. On a cache miss it
// refreshes the cache once before reporting false, so a voice downloaded after
// the process started (e.g. a Premium voice added in System Settings) is
// recognized without restarting a long-lived server.
func IsVoiceInstalled(voiceName string) (bool, error) {
	voices, err := getInstalledVoices()
	if err != nil {
		return false, err
	}
	if voices[voiceName] {
		return true, nil
	}

	voices, err = refreshInstalledVoices()
	if err != nil {
		return false, err
	}
	return voices[voiceName], nil
}

// VoiceNotInstalledError returns a user-friendly error message for missing voices
func VoiceNotInstalledError(voiceName string) string {
	return "Voice \"" + voiceName + "\" is not installed. " +
		"To download additional voices, go to: System Settings → Accessibility → Spoken Content → System Voice → Manage Voices"
}

// resetVoiceCache clears the cached voice data (test helper).
func resetVoiceCache() {
	voiceCache.mu.Lock()
	voiceCache.voices = nil
	voiceCache.mu.Unlock()
}
