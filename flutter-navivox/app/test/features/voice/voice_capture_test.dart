import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/channel/fake_navivox_channel.dart';
import 'package:navivox/core/protocol/navivox_event.dart';
import 'package:navivox/core/protocol/navivox_frame.dart';
import 'package:navivox/features/voice/services/voice_capture_service.dart';

void main() {
  group('FakeVoiceCaptureService', () {
    test('returns canned audio + transcript + duration + confidence', () async {
      final service = FakeVoiceCaptureService(
        audio: Uint8List.fromList(List<int>.generate(64, (i) => i)),
        transcript: 'hello assistant',
        duration: const Duration(seconds: 2),
        confidence: 0.92,
      );

      final capture = await service.capture(
        timeout: const Duration(seconds: 5),
      );

      expect(capture.audio.length, 64);
      expect(capture.transcript, 'hello assistant');
      expect(capture.duration, const Duration(seconds: 2));
      expect(capture.confidence, 0.92);
    });

    test('honours timeout by throwing when capture takes too long', () async {
      final service = FakeVoiceCaptureService(
        audio: Uint8List(0),
        transcript: '',
        duration: Duration.zero,
        confidence: 0.0,
        captureLatency: const Duration(milliseconds: 50),
      );

      await expectLater(
        () => service.capture(timeout: const Duration(milliseconds: 5)),
        throwsA(isA<VoiceCaptureTimeout>()),
      );
    });
  });

  group('FakeNavivoxChannel.sendVoice', () {
    test('encodes voice.submit with audio payload and metadata header', () {
      final channel = FakeNavivoxChannel();
      channel.enterFakeServerMode();
      final beforeCount = channel.state.messages.length;

      final audio = Uint8List.fromList(List<int>.generate(256, (i) => i % 256));
      channel.sendVoice(
        audio: audio,
        transcript: 'play that funky music',
        duration: const Duration(milliseconds: 1800),
        confidence: 0.87,
      );

      final sent = channel.lastSentFrame;
      expect(sent, isNotNull);
      expect(sent!.type, 'voice.submit');
      expect(sent.contentType, 'audio/pcm');
      expect(sent.payload, equals(audio));
      expect(sent.metadata['transcript'], 'play that funky music');
      expect(sent.metadata['duration_ms'], 1800);
      expect(sent.metadata['confidence'], 0.87);

      final wire = channel.lastSentFrameBytes!;
      final decoded = NavivoxFrameCodec.decode(wire);
      expect(decoded.payload, equals(audio));
      expect(decoded.metadata['transcript'], 'play that funky music');

      final received = channel.lastReceivedFrames;
      expect(received.map((f) => f.type).toList(), [
        'voice.transcript',
        'voice.audio',
      ]);
      expect(received[0].correlationId, sent.messageId);
      expect(received[1].correlationId, sent.messageId);
      expect(received[1].contentType, 'audio/pcm');
      expect(received[1].payload.length, greaterThan(0));

      final messages = channel.state.messages;
      expect(messages.length, beforeCount + 2);

      final user = messages[beforeCount];
      expect(user.author, NavivoxMessageAuthor.user);
      expect(user.kind, NavivoxMessageKind.voice);
      expect(user.voice, isNotNull);
      expect(user.voice!.transcript, 'play that funky music');
      expect(user.voice!.duration, const Duration(milliseconds: 1800));
      expect(user.voice!.confidence, 0.87);

      final assistant = messages[beforeCount + 1];
      expect(assistant.author, NavivoxMessageAuthor.assistant);
      expect(assistant.kind, NavivoxMessageKind.voice);
      expect(assistant.voice!.transcript, isNotEmpty);
    });

    test('rejects empty audio without recording a frame', () {
      final channel = FakeNavivoxChannel();
      channel.enterFakeServerMode();
      final before = channel.state.messages.length;

      channel.sendVoice(
        audio: Uint8List(0),
        transcript: 'silent',
        duration: Duration.zero,
        confidence: 0.5,
      );

      expect(channel.lastSentFrame, isNull);
      expect(channel.state.messages.length, before);
    });

    test('voice.submit metadata round-trips utf-8 transcripts', () {
      final channel = FakeNavivoxChannel();
      channel.enterFakeServerMode();

      channel.sendVoice(
        audio: Uint8List.fromList([1, 2, 3, 4]),
        transcript: 'こんにちは — привет — 안녕',
        duration: const Duration(milliseconds: 500),
        confidence: 1.0,
      );

      final wire = channel.lastSentFrameBytes!;
      final decoded = NavivoxFrameCodec.decode(wire);
      // The header must be valid UTF-8 JSON, surviving multilingual text.
      expect(decoded.metadata['transcript'], 'こんにちは — привет — 안녕');
      // Spot-check JSON encoding by re-parsing the header from the raw bytes.
      final headerLen = ByteData.sublistView(wire).getUint32(8, Endian.big);
      final header = jsonDecode(utf8.decode(wire.sublist(12, 12 + headerLen)));
      expect(header['metadata']['transcript'], 'こんにちは — привет — 안녕');
    });
  });
}
