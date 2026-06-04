package chunking

import "time"

func ChunkPCM(samples []int16, sampleRate int, maxChunk time.Duration) [][]int16 {
	return ChunkPCMWithOverlap(samples, sampleRate, maxChunk, 0)
}

func ChunkPCMWithOverlap(samples []int16, sampleRate int, maxChunk, overlap time.Duration) [][]int16 {
	if len(samples) == 0 {
		return nil
	}
	chunkSize := len(samples)
	if sampleRate > 0 && maxChunk > 0 {
		if calculated := int((int64(sampleRate) * int64(maxChunk)) / int64(time.Second)); calculated > 0 {
			chunkSize = calculated
		}
	}
	overlapSize := 0
	if sampleRate > 0 && overlap > 0 {
		overlapSize = int((int64(sampleRate) * int64(overlap)) / int64(time.Second))
		if overlapSize >= chunkSize {
			overlapSize = chunkSize - 1
		}
		if overlapSize < 0 {
			overlapSize = 0
		}
	}
	step := chunkSize - overlapSize
	if step <= 0 {
		step = chunkSize
	}

	chunks := make([][]int16, 0, (len(samples)+step-1)/step)
	for start := 0; start < len(samples); start += step {
		end := start + chunkSize
		if end > len(samples) {
			end = len(samples)
		}
		chunk := append([]int16(nil), samples[start:end]...)
		chunks = append(chunks, chunk)
		if end == len(samples) {
			break
		}
	}
	return chunks
}
