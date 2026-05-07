import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/channel/ssh_navivox_channel.dart';
import 'package:navivox/core/protocol/navivox_event.dart';
import 'package:navivox/core/protocol/navivox_frame.dart';
import 'package:navivox/core/transport/in_memory_transport.dart';

void main() {
  group('SshNavivoxChannel', () {
    test('sendText writes a chat.submit frame to the transport', () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      channel.sendText('hello over ssh');

      final wire = await pair.serverReadOne();
      final frame = NavivoxFrameCodec.decode(wire);
      expect(frame.type, 'chat.submit');
      expect(jsonDecode(utf8.decode(frame.payload)), {
        'text': 'hello over ssh',
      });

      await channel.close();
    });

    test('decodes server frames and appends messages to state', () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      // Server pushes a chat.message before any client send.
      pair.serverWrite(
        NavivoxFrameCodec.encode(
          NavivoxFrame(
            type: 'chat.message',
            messageId: 'srv-1',
            timestamp: DateTime.utc(2026, 5, 5, 12),
            contentType: 'application/json',
            payload: Uint8List.fromList(
              utf8.encode(jsonEncode({'text': 'welcome aboard'})),
            ),
          ),
        ),
      );

      // Wait for the channel to drain and process the frame.
      await Future<void>.delayed(const Duration(milliseconds: 10));

      final messages = channel.state.messages
          .where((m) => m.author == NavivoxMessageAuthor.assistant)
          .toList();
      expect(messages, isNotEmpty);
      expect(messages.last.text, 'welcome aboard');

      await channel.close();
    });

    test('round-trips a chat turn end-to-end through the transport', () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);
      var notifications = 0;
      channel.addListener(() => notifications += 1);

      channel.sendText('ping');

      // Read what the channel sent and emulate the server's side.
      final submitBytes = await pair.serverReadOne();
      final submit = NavivoxFrameCodec.decode(submitBytes);
      expect(submit.type, 'chat.submit');

      pair.serverWrite(
        NavivoxFrameCodec.encode(
          NavivoxFrame(
            type: 'chat.message',
            messageId: 'srv-2',
            timestamp: DateTime.utc(2026, 5, 5, 12, 0, 1),
            correlationId: submit.messageId,
            contentType: 'application/json',
            payload: Uint8List.fromList(
              utf8.encode(jsonEncode({'text': 'pong'})),
            ),
          ),
        ),
      );
      pair.serverWrite(
        NavivoxFrameCodec.encode(
          NavivoxFrame(
            type: 'chat.final',
            messageId: 'srv-3',
            timestamp: DateTime.utc(2026, 5, 5, 12, 0, 2),
            correlationId: submit.messageId,
            contentType: 'application/json',
            payload: Uint8List.fromList(
              utf8.encode(jsonEncode({'turn_complete': true})),
            ),
          ),
        ),
      );

      await Future<void>.delayed(const Duration(milliseconds: 20));

      final assistant = channel.state.messages.lastWhere(
        (m) => m.author == NavivoxMessageAuthor.assistant,
      );
      expect(assistant.text, 'pong');
      expect(notifications, greaterThanOrEqualTo(2));

      await channel.close();
    });

    test('close stops emitting state updates', () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      await channel.close();

      // After close, sendText is a no-op (transport is closed) — must not throw.
      expect(() => channel.sendText('after close'), returnsNormally);
    });
  });

  group('InMemoryByteTransportPair', () {
    test(
      'client writes are visible on the server side and vice versa',
      () async {
        final pair = InMemoryByteTransportPair();

        pair.client.sink.add(Uint8List.fromList([1, 2, 3]));
        final fromClient = await pair.serverReadBytes(3);
        expect(fromClient, [1, 2, 3]);

        pair.serverWrite(Uint8List.fromList([9, 8, 7]));
        final fromServer = await pair.client.bytes.first;
        expect(fromServer, [9, 8, 7]);
      },
    );

    test('closing the client closes both sides cleanly', () async {
      final pair = InMemoryByteTransportPair();
      pair.client.sink.add(Uint8List.fromList([1]));
      await pair.client.close();

      final remaining = await pair.serverReadAll();
      expect(remaining, [1]);
    });
  });
}
