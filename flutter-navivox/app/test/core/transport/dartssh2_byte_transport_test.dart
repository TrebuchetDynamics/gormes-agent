import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/channel/ssh_navivox_channel.dart';
import 'package:navivox/core/protocol/navivox_event.dart';
import 'package:navivox/core/protocol/navivox_frame.dart';
import 'package:navivox/core/transport/dartssh2_byte_transport.dart';

void main() {
  group('Dartssh2ByteTransport.fromParts', () {
    test('forwards stdout chunks to bytes stream', () async {
      final stdout = StreamController<Uint8List>();
      final stdin = StreamController<List<int>>();
      var closeCalls = 0;

      final transport = Dartssh2ByteTransport.fromParts(
        stdout: stdout.stream,
        stdin: stdin.sink,
        onClose: () async {
          closeCalls++;
        },
      );

      final received = <Uint8List>[];
      final sub = transport.bytes.listen(received.add);

      stdout.add(Uint8List.fromList([1, 2, 3]));
      stdout.add(Uint8List.fromList([4, 5]));
      await Future<void>.delayed(const Duration(milliseconds: 5));

      expect(received.length, 2);
      expect(received[0], [1, 2, 3]);
      expect(received[1], [4, 5]);

      await sub.cancel();
      unawaited(stdout.close());
      unawaited(stdin.close());
      await transport.close();
      expect(closeCalls, 1);
    });

    test('writes Uint8List to sink, reaching the underlying stdin', () async {
      final stdout = StreamController<Uint8List>();
      final stdin = StreamController<List<int>>();

      final transport = Dartssh2ByteTransport.fromParts(
        stdout: stdout.stream,
        stdin: stdin.sink,
        onClose: () async {},
      );

      final captured = <List<int>>[];
      final sub = stdin.stream.listen(captured.add);

      transport.sink.add(Uint8List.fromList([7, 8, 9]));
      transport.sink.add(Uint8List.fromList([10]));
      await Future<void>.delayed(const Duration(milliseconds: 5));

      expect(captured.length, 2);
      expect(captured[0], [7, 8, 9]);
      expect(captured[1], [10]);

      await sub.cancel();
      unawaited(stdout.close());
      unawaited(stdin.close());
      await transport.close();
    });

    test('close() runs the closer exactly once even if invoked twice',
        () async {
      var closeCalls = 0;
      final transport = Dartssh2ByteTransport.fromParts(
        stdout: const Stream<Uint8List>.empty(),
        stdin: StreamController<List<int>>().sink,
        onClose: () async {
          closeCalls++;
        },
      );

      await transport.close();
      await transport.close();
      expect(closeCalls, 1);
    });
  });

  group('SshNavivoxChannel over Dartssh2ByteTransport.fromParts', () {
    test('end-to-end chat round trip works through the adapter', () async {
      final stdout = StreamController<Uint8List>();
      final stdin = StreamController<List<int>>();

      final transport = Dartssh2ByteTransport.fromParts(
        stdout: stdout.stream,
        stdin: stdin.sink,
        onClose: () async {},
      );
      final channel = SshNavivoxChannel(transport: transport);

      // Capture what the channel writes to "stdin".
      final sentFrames = <Uint8List>[];
      final stdinSub = stdin.stream.listen(
        (chunk) => sentFrames.add(Uint8List.fromList(chunk)),
      );

      channel.sendText('hello via dartssh2');
      await Future<void>.delayed(const Duration(milliseconds: 5));

      expect(sentFrames, isNotEmpty);
      final submit = NavivoxFrameCodec.decode(sentFrames.single);
      expect(submit.type, 'chat.submit');
      expect(jsonDecode(utf8.decode(submit.payload)),
          {'text': 'hello via dartssh2'});

      // Pretend the server replied.
      stdout.add(NavivoxFrameCodec.encode(NavivoxFrame(
        type: 'chat.message',
        messageId: 'srv-1',
        timestamp: DateTime.utc(2026, 5, 5, 12),
        correlationId: submit.messageId,
        contentType: 'application/json',
        payload: Uint8List.fromList(
            utf8.encode(jsonEncode({'text': 'hi from server'}))),
      )));
      await Future<void>.delayed(const Duration(milliseconds: 10));

      final assistant = channel.state.messages.lastWhere(
        (m) => m.author == NavivoxMessageAuthor.assistant,
      );
      expect(assistant.text, 'hi from server');

      await stdinSub.cancel();
      unawaited(stdout.close());
      unawaited(stdin.close());
      await channel.close();
    });
  });
}
