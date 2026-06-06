package normalization

import (
	"math"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/contract"
)

const (
	speechNormalizeTargetPeak = 26000
	speechTrimPaddingSamples  = 1600
)

func NormalizeSpeechPCM(pcm contract.PCM) contract.PCM {
	if len(pcm.Samples) == 0 {
		return pcm
	}
	centered := removeDCOffset(pcm.Samples)
	start, end := speechBounds(centered)
	if start >= end {
		return contract.PCM{Samples: centered, SampleRate: pcm.SampleRate}
	}
	start -= min(start, speechTrimPadding(pcm.SampleRate))
	end += min(len(centered)-end, speechTrimPadding(pcm.SampleRate))
	trimmed := append([]int16(nil), centered[start:end]...)
	return contract.PCM{Samples: normalizePeak(trimmed), SampleRate: pcm.SampleRate}
}

func removeDCOffset(samples []int16) []int16 {
	var sum int64
	for _, sample := range samples {
		sum += int64(sample)
	}
	mean := int(sum / int64(len(samples)))
	centered := make([]int16, len(samples))
	for i, sample := range samples {
		centered[i] = clampInt16(int(sample) - mean)
	}
	return centered
}

func speechBounds(samples []int16) (int, int) {
	var total int64
	var peak int
	for _, sample := range samples {
		abs := absInt16(sample)
		total += int64(abs)
		if abs > peak {
			peak = abs
		}
	}
	if peak == 0 {
		return 0, 0
	}
	avg := int(total / int64(len(samples)))
	thresholdFloor := 200
	if peak < 800 {
		thresholdFloor = max(1, peak/8)
	}
	threshold := max(thresholdFloor, avg*3)
	threshold = min(threshold, max(thresholdFloor, peak/4))

	start := 0
	for start < len(samples) && absInt16(samples[start]) < threshold {
		start++
	}
	end := len(samples)
	for end > start && absInt16(samples[end-1]) < threshold {
		end--
	}
	return start, end
}

func speechTrimPadding(sampleRate int) int {
	if sampleRate <= 0 {
		return speechTrimPaddingSamples
	}
	return max(1, sampleRate/10)
}

func normalizePeak(samples []int16) []int16 {
	peak := 0
	for _, sample := range samples {
		peak = max(peak, absInt16(sample))
	}
	if peak == 0 || peak >= speechNormalizeTargetPeak {
		return samples
	}
	scale := float64(speechNormalizeTargetPeak) / float64(peak)
	out := make([]int16, len(samples))
	for i, sample := range samples {
		out[i] = clampInt16(int(math.Round(float64(sample) * scale)))
	}
	return out
}

func absInt16(v int16) int {
	if v < 0 {
		return -int(v)
	}
	return int(v)
}

func clampInt16(v int) int16 {
	if v > math.MaxInt16 {
		return math.MaxInt16
	}
	if v < math.MinInt16 {
		return math.MinInt16
	}
	return int16(v)
}
