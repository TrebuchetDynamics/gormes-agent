import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/channel/ssh_navivox_channel.dart';
import 'package:navivox/core/protocol/navivox_event.dart';
import 'package:navivox/core/protocol/navivox_frame.dart';
import 'package:navivox/core/transport/in_memory_transport.dart';

const _tcId = 'tc-99';

Uint8List _frame(
  String type, {
  required String messageId,
  required Map<String, Object?> payload,
  Map<String, Object?> metadata = const {},
  String? contentType = 'application/json',
}) {
  return NavivoxFrameCodec.encode(
    NavivoxFrame(
      type: type,
      messageId: messageId,
      timestamp: DateTime.utc(2026, 5, 5, 12),
      contentType: contentType,
      payload: Uint8List.fromList(utf8.encode(jsonEncode(payload))),
      metadata: metadata,
    ),
  );
}

void main() {
  group('SshNavivoxChannel artifacts', () {
    test('tool.artifact.ready attaches an artifact to the matching tool call',
        () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      pair.serverWrite(_frame('tool.call.started',
          messageId: 'f-1',
          metadata: {'tool_call_id': _tcId},
          payload: {'name': 'shell.run', 'summary': 'go test ./...'}));
      pair.serverWrite(_frame('tool.artifact.ready',
          messageId: 'f-2',
          metadata: {
            'tool_call_id': _tcId,
            'artifact_id': 'art-1',
            'kind': 'terminal',
          },
          payload: {
            'title': 'go test output',
            'summary': '39/39 passed',
            'ref': 'artifact://terminal/abc',
          }));
      await Future<void>.delayed(const Duration(milliseconds: 10));

      final call = channel.state.messages
          .firstWhere((m) => m.kind == NavivoxMessageKind.toolCall);
      expect(call.toolCall!.artifacts.length, 1);
      final art = call.toolCall!.artifacts.single;
      expect(art.id, 'art-1');
      expect(art.kind, 'terminal');
      expect(art.title, 'go test output');
      expect(art.summary, '39/39 passed');
      expect(art.ref, 'artifact://terminal/abc');

      await channel.close();
    });

    test('multiple artifacts accumulate in arrival order', () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      pair.serverWrite(_frame('tool.call.started',
          messageId: 'f-1',
          metadata: {'tool_call_id': _tcId},
          payload: {'name': 'patch.apply', 'summary': 'editing files'}));
      pair.serverWrite(_frame('tool.artifact.ready',
          messageId: 'f-2',
          metadata: {'tool_call_id': _tcId, 'artifact_id': 'a', 'kind': 'diff'},
          payload: {'title': 'before'}));
      pair.serverWrite(_frame('tool.artifact.ready',
          messageId: 'f-3',
          metadata: {'tool_call_id': _tcId, 'artifact_id': 'b', 'kind': 'diff'},
          payload: {'title': 'after'}));
      await Future<void>.delayed(const Duration(milliseconds: 10));

      final call = channel.state.messages
          .firstWhere((m) => m.kind == NavivoxMessageKind.toolCall);
      expect(call.toolCall!.artifacts.map((a) => a.id).toList(), ['a', 'b']);
      expect(call.toolCall!.artifacts.map((a) => a.title).toList(),
          ['before', 'after']);

      await channel.close();
    });

    test('artifact frame without a matching tool call is ignored gracefully',
        () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      pair.serverWrite(_frame('tool.artifact.ready',
          messageId: 'f-1',
          metadata: {
            'tool_call_id': 'no-such-tool-call',
            'artifact_id': 'orphan',
            'kind': 'file',
          },
          payload: {'title': 'orphan'}));
      await Future<void>.delayed(const Duration(milliseconds: 10));

      final calls = channel.state.messages
          .where((m) => m.kind == NavivoxMessageKind.toolCall);
      expect(calls, isEmpty);

      await channel.close();
    });

    test('artifacts survive subsequent tool.call.progress / completed updates',
        () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      pair.serverWrite(_frame('tool.call.started',
          messageId: 'f-1',
          metadata: {'tool_call_id': _tcId},
          payload: {'name': 'shell.run', 'summary': 'starting'}));
      pair.serverWrite(_frame('tool.artifact.ready',
          messageId: 'f-2',
          metadata: {'tool_call_id': _tcId, 'artifact_id': 'a', 'kind': 'file'},
          payload: {'title': 'output.log'}));
      pair.serverWrite(_frame('tool.call.completed',
          messageId: 'f-3',
          metadata: {'tool_call_id': _tcId},
          payload: {'summary': 'done'}));
      await Future<void>.delayed(const Duration(milliseconds: 10));

      final call = channel.state.messages
          .firstWhere((m) => m.kind == NavivoxMessageKind.toolCall);
      expect(call.toolCall!.status, 'completed');
      expect(call.toolCall!.artifacts.length, 1);
      expect(call.toolCall!.artifacts.single.id, 'a');

      await channel.close();
    });
  });
}
