package audio

import "time"

func ChunkPCM(samples []int16, sampleRate int, maxChunk time.Duration) [][]int16 {
	if len(samples) == 0 {
		return nil
	}
	chunkSize := len(samples)
	if sampleRate > 0 && maxChunk > 0 {
		if calculated := int((int64(sampleRate) * int64(maxChunk)) / int64(time.Second)); calculated > 0 {
			chunkSize = calculated
		}
	}

	chunks := make([][]int16, 0, (len(samples)+chunkSize-1)/chunkSize)
	for start := 0; start < len(samples); start += chunkSize {
		end := start + chunkSize
		if end > len(samples) {
			end = len(samples)
		}
		chunk := append([]int16(nil), samples[start:end]...)
		chunks = append(chunks, chunk)
	}
	return chunks
}
