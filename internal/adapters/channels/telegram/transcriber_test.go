package telegram

import "context"

type fakeAudioTranscriber struct {
	Transcript string
	Err        error
}

func (f fakeAudioTranscriber) Transcribe(ctx context.Context, audio AudioInput) (string, error) {
	_ = ctx
	if f.Err != nil {
		return "", f.Err
	}
	return f.Transcript, nil
}
