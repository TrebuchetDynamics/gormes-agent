package audiodelivery

import "strings"

// Guidance is appended to session context when a turn should receive gateway
// synthesized audio after the final written response.
const Guidance = "## Audio Delivery\nGateway audio delivery is enabled for this turn. Answer normally in written text; the gateway will synthesize and attach the audio after your final answer. Do not claim that you generated audio yourself, and do not claim the TTS provider is unavailable."

var intentTextReplacer = strings.NewReplacer(
	"á", "a",
	"é", "e",
	"í", "i",
	"ó", "o",
	"ú", "u",
	"ü", "u",
)

// RequestsAudioReply reports whether inbound text or attachment kinds ask the
// gateway to attach synthesized speech for the final response.
func RequestsAudioReply(text string, attachmentKinds []string) bool {
	for _, kind := range attachmentKinds {
		switch strings.ToLower(strings.TrimSpace(kind)) {
		case "voice", "audio", "voice_transcript":
			return true
		}
	}
	text = intentTextReplacer.Replace(strings.ToLower(strings.Join(strings.Fields(text), " ")))
	if text == "" {
		return false
	}
	for _, phrase := range []string{
		"cannot read",
		"can't read",
		"cant read",
		"audio version",
		"send audio",
		"voice reply",
		"reply by voice",
		"read it aloud",
		"read out loud",
		"mandame audio",
		"mandamelo en audio",
		"mandalo en audio",
		"enviame audio",
		"enviamelo en audio",
		"envialo en audio",
		"pasame audio",
		"pasamelo en audio",
		"por audio",
		"para audio",
		"audio por favor",
		"leelo en voz alta",
		"leemelo",
		"voz alta",
	} {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}

// AppendGuidance appends the audio delivery instruction block when enabled.
func AppendGuidance(sessionBlock string, enabled bool) string {
	if !enabled {
		return sessionBlock
	}
	if strings.TrimSpace(sessionBlock) == "" {
		return Guidance
	}
	return strings.TrimRight(sessionBlock, "\n") + "\n\n" + Guidance
}
