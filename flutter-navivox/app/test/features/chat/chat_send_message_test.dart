import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/channel/fake_navivox_channel.dart';
import 'package:navivox/core/protocol/navivox_event.dart';
import 'package:navivox/core/protocol/navivox_frame.dart';

void main() {
  group('FakeNavivoxChannel chat round trip', () {
    test('sendText emits a chat.submit frame through the codec', () {
      final channel = FakeNavivoxChannel();
      channel.enterFakeServerMode();
      final beforeCount = channel.state.messages.length;

      channel.sendText('hello navivox');

      final sent = channel.lastSentFrame;
      expect(
        sent,
        isNotNull,
        reason: 'sendText should record the encoded frame',
      );
      expect(sent!.type, 'chat.submit');
      expect(sent.messageId.isNotEmpty, isTrue);

      final payload =
          jsonDecode(utf8.decode(sent.payload)) as Map<String, Object?>;
      expect(payload['text'], 'hello navivox');

      // Wire bytes round-trip through the real codec, not just an in-memory shortcut.
      final wire = channel.lastSentFrameBytes!;
      final decoded = NavivoxFrameCodec.decode(wire);
      expect(decoded.type, 'chat.submit');
      expect(decoded.messageId, sent.messageId);
      expect(jsonDecode(utf8.decode(decoded.payload)), payload);

      final messages = channel.state.messages;
      expect(
        messages.length,
        greaterThan(beforeCount),
        reason: 'user message and assistant response should be appended',
      );

      final user = messages.lastWhere(
        (m) => m.author == NavivoxMessageAuthor.user,
      );
      expect(user.text, 'hello navivox');

      final assistant = messages.lastWhere(
        (m) => m.author == NavivoxMessageAuthor.assistant,
      );
      expect(assistant.text, isNotNull);
      expect(
        assistant.text,
        isNot(
          equals('Navivox fake channel is ready for local protocol UI work.'),
        ),
      );
    });

    test('sendText ignores blank input and does not record a frame', () {
      final channel = FakeNavivoxChannel();
      channel.enterFakeServerMode();
      final before = channel.state.messages.length;

      channel.sendText('   ');

      expect(channel.lastSentFrame, isNull);
      expect(channel.state.messages.length, before);
    });

    test(
      'mock server response is built from real chat.message + chat.final frames',
      () {
        final channel = FakeNavivoxChannel();
        channel.enterFakeServerMode();
        channel.sendText('ping');

        final received = channel.lastReceivedFrames;
        expect(received.length, 2);
        expect(received[0].type, 'chat.message');
        expect(received[1].type, 'chat.final');
        expect(received[0].correlationId, channel.lastSentFrame!.messageId);
        expect(received[1].correlationId, channel.lastSentFrame!.messageId);
      },
    );
  });

  group('NavivoxFrameCodec.encode', () {
    test('encode/decode round-trips a chat.submit frame with payload', () {
      final payload = Uint8List.fromList(
        utf8.encode(jsonEncode({'text': 'hi'})),
      );
      final original = NavivoxFrame(
        type: 'chat.submit',
        messageId: 'frame-1',
        timestamp: DateTime.utc(2026, 5, 5, 12),
        payload: payload,
      );

      final bytes = NavivoxFrameCodec.encode(original);
      final decoded = NavivoxFrameCodec.decode(bytes);

      expect(decoded.type, 'chat.submit');
      expect(decoded.messageId, 'frame-1');
      expect(decoded.timestamp, DateTime.utc(2026, 5, 5, 12));
      expect(decoded.payload, equals(payload));
    });

    test('encode preserves optional correlation/turn/agent fields', () {
      final original = NavivoxFrame(
        type: 'chat.message',
        messageId: 'frame-2',
        timestamp: DateTime.utc(2026, 5, 5, 12, 0, 1),
        payload: Uint8List(0),
        correlationId: 'frame-1',
        turnId: 'turn-7',
        agentId: 'default',
        contentType: 'application/json',
        metadata: const {'origin': 'fake'},
      );

      final decoded = NavivoxFrameCodec.decode(
        NavivoxFrameCodec.encode(original),
      );

      expect(decoded.correlationId, 'frame-1');
      expect(decoded.turnId, 'turn-7');
      expect(decoded.agentId, 'default');
      expect(decoded.contentType, 'application/json');
      expect(decoded.metadata, {'origin': 'fake'});
    });
  });
}
