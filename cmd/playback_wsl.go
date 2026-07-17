//go:build !darwin

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

var (
	wslDetectOnce sync.Once
	isWSL         bool
	wslTempDir    string
)

// detectWSL checks if we're running inside Windows Subsystem for Linux.
func detectWSL() {
	wslDetectOnce.Do(func() {
		data, err := os.ReadFile("/proc/version")
		if err != nil {
			return
		}
		lower := strings.ToLower(string(data))
		isWSL = strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
		if isWSL {
			// Determine temp dir on the Windows side
			user := os.Getenv("USER")
			wslTempDir = fmt.Sprintf("/mnt/c/Users/%s/.cache/mcp-tts", user)
			if err := os.MkdirAll(wslTempDir, 0755); err != nil {
				log.Warn("Could not create WSL temp dir, falling back to in-process audio", "path", wslTempDir, "error", err)
				isWSL = false
				return
			}
			log.Debug("WSL detected, audio will play via Windows", "tempDir", wslTempDir)
		}
	})
}

// wslPlayFullWAV writes a complete WAV buffer (with header) to a temp file and plays it.
func wslPlayFullWAV(wavData []byte) error {
	filename := fmt.Sprintf("tts_%d.wav", time.Now().UnixMilli())
	wavPath := filepath.Join(wslTempDir, filename)

	if err := os.WriteFile(wavPath, wavData, 0644); err != nil {
		return fmt.Errorf("failed to write WAV file: %w", err)
	}

	winPath := unixToWindowsPath(wavPath)
	log.Debug("Playing WAV via Windows PowerShell", "path", winPath)

	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command",
		fmt.Sprintf(`(New-Object System.Media.SoundPlayer '%s').PlaySync()`, winPath))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		os.Remove(wavPath)
		return fmt.Errorf("PowerShell playback failed: %w — %s", err, stderr.String())
	}

	os.Remove(wavPath)
	return nil
}

// wslPlayWAV writes PCM data as a WAV file and plays it via PowerShell on Windows.
func wslPlayWAV(audioData []byte, sampleRate int) error {
	filename := fmt.Sprintf("tts_%d.wav", time.Now().UnixMilli())
	wavPath := filepath.Join(wslTempDir, filename)

	f, err := os.Create(wavPath)
	if err != nil {
		return fmt.Errorf("failed to create WAV file: %w", err)
	}
	if err := writeWAVHeader(f, len(audioData), sampleRate, 1, 16); err != nil {
		f.Close()
		os.Remove(wavPath)
		return fmt.Errorf("failed to write WAV header: %w", err)
	}
	if _, err := f.Write(audioData); err != nil {
		f.Close()
		os.Remove(wavPath)
		return fmt.Errorf("failed to write WAV data: %w", err)
	}
	f.Close()

	// Convert Unix path to Windows path for PowerShell
	winPath := unixToWindowsPath(wavPath)

	log.Debug("Playing audio via Windows PowerShell", "path", winPath)

	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command",
		fmt.Sprintf(`(New-Object System.Media.SoundPlayer '%s').PlaySync()`, winPath))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		os.Remove(wavPath)
		return fmt.Errorf("PowerShell playback failed: %w — %s", err, stderr.String())
	}

	// Cleanup temp file after playback
	os.Remove(wavPath)
	return nil
}

// wslPlayMP3 writes MP3 data to a file and plays it via Windows default app.
func wslPlayMP3(audioData []byte) error {
	filename := fmt.Sprintf("tts_%d.mp3", time.Now().UnixMilli())
	mp3Path := filepath.Join(wslTempDir, filename)

	if err := os.WriteFile(mp3Path, audioData, 0644); err != nil {
		return fmt.Errorf("failed to write MP3 file: %w", err)
	}

	winPath := unixToWindowsPath(mp3Path)

	log.Debug("Playing MP3 via Windows", "path", winPath)

	// Use cmd.exe start /wait to play and wait for completion
	cmd := exec.Command("cmd.exe", "/c", "start", "/wait", "", winPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		os.Remove(mp3Path)
		return fmt.Errorf("Windows MP3 playback failed: %w — %s", err, stderr.String())
	}

	os.Remove(mp3Path)
	return nil
}

// unixToWindowsPath converts /mnt/c/Users/... to C:\Users\...
func unixToWindowsPath(p string) string {
	if !strings.HasPrefix(p, "/mnt/") {
		return p
	}
	// /mnt/c/Users/nizar/... → C:\Users\nizar\...
	parts := strings.SplitN(p, "/", 4) // ["", "mnt", "c", "Users/nizar/..."]
	if len(parts) < 4 {
		return p
	}
	drive := strings.ToUpper(parts[2])
	rest := strings.ReplaceAll(parts[3], "/", `\`)
	return fmt.Sprintf(`%s:\%s`, drive, rest)
}
