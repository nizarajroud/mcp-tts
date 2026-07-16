/*
Copyright © 2025 blacktop

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/log"
)

// PiperTTSParams holds the input parameters for the piper_tts tool.
type PiperTTSParams struct {
	Text        string   `json:"text" mcp:"The text to convert to speech using Piper TTS (offline neural engine)"`
	Model       *string  `json:"model,omitempty,omitzero" mcp:"Path to ONNX model file or model name. Uses PIPER_MODEL env var if not set."`
	Speaker     *int     `json:"speaker,omitempty,omitzero" mcp:"Speaker ID for multi-speaker models (default: 0)"`
	LengthScale *float64 `json:"length_scale,omitempty,omitzero" mcp:"Speaking rate — higher values are slower (default: 1.0)"`
	NoiseScale  *float64 `json:"noise_scale,omitempty,omitzero" mcp:"Variation in speech (default: 0.667)"`
}

// piperModelConfig represents the JSON config file that accompanies a Piper model.
type piperModelConfig struct {
	Audio struct {
		SampleRate int `json:"sample_rate"`
	} `json:"audio"`
	NumSpeakers int `json:"num_speakers"`
}

// DefaultPiperLengthScale is the default speaking rate (1.0 = normal).
const DefaultPiperLengthScale = 1.0

// DefaultPiperNoiseScale is the default speech variation.
const DefaultPiperNoiseScale = 0.667

// DefaultPiperSampleRate is the fallback sample rate when the model config
// cannot be read. Most Piper models use 22050 Hz.
const DefaultPiperSampleRate = 22050

// isPiperAvailable returns true if the piper binary is in PATH.
func isPiperAvailable() bool {
	_, err := exec.LookPath("piper")
	return err == nil
}

// piperSampleRate reads the model's JSON config to determine the sample rate.
// Falls back to DefaultPiperSampleRate if the config is unreadable.
func piperSampleRate(modelPath string) int {
	configPath := modelPath + ".json"
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Debug("Could not read piper model config, using default sample rate", "path", configPath, "error", err)
		return DefaultPiperSampleRate
	}
	var cfg piperModelConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Debug("Could not parse piper model config, using default sample rate", "path", configPath, "error", err)
		return DefaultPiperSampleRate
	}
	if cfg.Audio.SampleRate > 0 {
		return cfg.Audio.SampleRate
	}
	return DefaultPiperSampleRate
}

// runPiper executes the piper binary and returns raw PCM 16-bit LE audio data.
func runPiper(ctx context.Context, params PiperTTSParams) ([]byte, int, error) {
	model := ""
	if params.Model != nil && *params.Model != "" {
		model = *params.Model
	} else {
		model = os.Getenv("PIPER_MODEL")
	}
	if model == "" {
		return nil, 0, fmt.Errorf("no model specified: set PIPER_MODEL env var or provide 'model' parameter")
	}

	args := []string{"--model", model, "--output-raw"}

	if params.Speaker != nil {
		args = append(args, "--speaker", fmt.Sprintf("%d", *params.Speaker))
	}

	if params.LengthScale != nil {
		args = append(args, "--length-scale", fmt.Sprintf("%.3f", *params.LengthScale))
	}

	if params.NoiseScale != nil {
		args = append(args, "--noise-scale", fmt.Sprintf("%.3f", *params.NoiseScale))
	}

	log.Debug("Running piper", "args", args, "text_length", len(params.Text))

	cmd := exec.CommandContext(ctx, "piper", args...)
	cmd.Stdin = strings.NewReader(params.Text)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		log.Error("Piper failed", "error", err, "stderr", stderrStr)
		return nil, 0, fmt.Errorf("piper failed: %w — %s", err, stderrStr)
	}

	audioData := stdout.Bytes()
	if len(audioData) == 0 {
		return nil, 0, fmt.Errorf("piper produced no audio output")
	}

	sampleRate := piperSampleRate(model)
	log.Debug("Piper audio generated", "bytes", len(audioData), "sample_rate", sampleRate)

	return audioData, sampleRate, nil
}

// piperSettingsSchema returns the elicitation schema for Piper TTS settings.
func piperSettingsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"length_scale": map[string]any{
				"type":        "number",
				"title":       "Speaking Rate",
				"description": "Higher values are slower (default: 1.0)",
				"default":     DefaultPiperLengthScale,
			},
			"speaker": map[string]any{
				"type":        "integer",
				"title":       "Speaker ID",
				"description": "Speaker ID for multi-speaker models (default: 0)",
				"default":     0,
			},
		},
	}
}

// applyPiperSettings applies elicited settings to PiperTTSParams.
func applyPiperSettings(input *PiperTTSParams, content map[string]any) {
	if input == nil || content == nil {
		return
	}
	if v, ok := content["length_scale"]; ok {
		if f, ok := v.(float64); ok {
			input.LengthScale = &f
		}
	}
	if v, ok := content["speaker"]; ok {
		switch s := v.(type) {
		case float64:
			i := int(s)
			input.Speaker = &i
		case int:
			input.Speaker = &s
		}
	}
}

// piperRecommendationArgs builds the recommendation args for the tts tool.
func piperRecommendationArgs(input PiperTTSParams) map[string]any {
	args := map[string]any{"text": input.Text}
	if input.Model != nil && *input.Model != "" {
		args["model"] = *input.Model
	}
	if input.Speaker != nil {
		args["speaker"] = *input.Speaker
	}
	if input.LengthScale != nil {
		args["length_scale"] = *input.LengthScale
	}
	if input.NoiseScale != nil {
		args["noise_scale"] = *input.NoiseScale
	}
	return args
}
