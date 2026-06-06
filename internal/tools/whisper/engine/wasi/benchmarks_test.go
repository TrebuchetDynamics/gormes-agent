package wasi

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func BenchmarkWhisperWASI(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	b.Run("tiny.en", func(b *testing.B) {
		benchmarkWhisperWASITinyEn(b, ctx)
	})
}

func benchmarkWhisperWASITinyEn(b *testing.B, ctx context.Context) {
	modelPath := testTinyEnModelPath(b, ctx)
	wasm := readWhisperWASM(b)
	fixturePath := whisperTestdataPath("jfk.wav")
	samples, err := DecodePCM16Mono16kWAV(fixturePath)
	if err != nil {
		b.Fatalf("decode benchmark fixture: %v", err)
	}
	audioDuration := time.Duration(float64(len(samples)) / 16_000 * float64(time.Second))
	if audioDuration <= 0 {
		b.Fatalf("audio duration = %s", audioDuration)
	}

	var totalLoad, totalInference time.Duration
	var transcriptChars int
	var peakAllocBytes uint64

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runtime.GC()
		var before, afterLoad, afterInference runtime.MemStats
		runtime.ReadMemStats(&before)

		loadStarted := time.Now()
		transcriber, err := NewTranscriber(ctx, modelPath, wasm)
		if err != nil {
			b.Fatalf("NewTranscriber: %v", err)
		}
		loadElapsed := time.Since(loadStarted)
		runtime.ReadMemStats(&afterLoad)

		inferenceStarted := time.Now()
		transcript, err := transcriber.TranscribeWAV(ctx, fixturePath)
		inferenceElapsed := time.Since(inferenceStarted)
		runtime.ReadMemStats(&afterInference)
		if closeErr := transcriber.Close(context.Background()); closeErr != nil {
			b.Fatalf("Close: %v", closeErr)
		}
		if err != nil {
			b.Fatalf("TranscribeWAV: %v", err)
		}
		if !strings.Contains(strings.ToLower(transcript), "ask not") {
			b.Fatalf("benchmark transcript missing expected JFK text:\n%s", transcript)
		}

		totalLoad += loadElapsed
		totalInference += inferenceElapsed
		transcriptChars = len([]rune(transcript))
		peak := max(afterLoad.Alloc, afterInference.Alloc)
		if peak > before.Alloc && peak-before.Alloc > peakAllocBytes {
			peakAllocBytes = peak - before.Alloc
		}
	}
	b.StopTimer()

	iterations := float64(b.N)
	avgLoad := totalLoad.Seconds() / iterations
	avgInference := totalInference.Seconds() / iterations
	b.ReportMetric(avgLoad*1000, "model_load_ms")
	b.ReportMetric(avgInference*1000, "inference_ms")
	b.ReportMetric(avgInference/audioDuration.Seconds(), "realtime_factor")
	b.ReportMetric(float64(peakAllocBytes)/1048576, "peak_memory_mb")
	b.ReportMetric(float64(transcriptChars), "transcript_chars")
}
