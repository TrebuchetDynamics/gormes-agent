import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/channel/navivox_channel.dart';
import 'package:navivox/core/channel/ssh_navivox_channel.dart';
import 'package:navivox/core/protocol/navivox_event.dart';
import 'package:navivox/core/protocol/navivox_frame.dart';
import 'package:navivox/core/transport/in_memory_transport.dart';

const _toolCallId = 'tc-42';

Uint8List _toolFrame({
  required String type,
  required String messageId,
  required Map<String, Object?> payload,
  String? toolCallId,
  String? approvalId,
}) {
  final metadata = <String, Object?>{
    if (toolCallId != null) 'tool_call_id': toolCallId,
    if (approvalId != null) 'approval_id': approvalId,
  };
  return NavivoxFrameCodec.encode(
    NavivoxFrame(
      type: type,
      messageId: messageId,
      timestamp: DateTime.utc(2026, 5, 5, 12),
      contentType: 'application/json',
      payload: Uint8List.fromList(utf8.encode(jsonEncode(payload))),
      metadata: metadata,
    ),
  );
}

void main() {
  group('SshNavivoxChannel tool lifecycle', () {
    test(
      'tool.call.started appends a toolCall message keyed by tool_call_id',
      () async {
        final pair = InMemoryByteTransportPair();
        final channel = SshNavivoxChannel(transport: pair.client);

        pair.serverWrite(
          _toolFrame(
            type: 'tool.call.started',
            messageId: 'frame-1',
            toolCallId: _toolCallId,
            payload: {
              'name': 'workspace.read',
              'summary': 'reading project root',
            },
          ),
        );
        await Future<void>.delayed(const Duration(milliseconds: 10));

        final calls = channel.state.messages
            .where((m) => m.kind == NavivoxMessageKind.toolCall)
            .toList();
        expect(calls.length, 1);
        expect(calls.single.id, _toolCallId);
        expect(calls.single.toolCall!.name, 'workspace.read');
        expect(calls.single.toolCall!.status, 'started');
        expect(calls.single.toolCall!.summary, 'reading project root');

        await channel.close();
      },
    );

    test(
      'tool.call.progress updates the existing toolCall message in place',
      () async {
        final pair = InMemoryByteTransportPair();
        final channel = SshNavivoxChannel(transport: pair.client);

        pair.serverWrite(
          _toolFrame(
            type: 'tool.call.started',
            messageId: 'frame-1',
            toolCallId: _toolCallId,
            payload: {'name': 'shell.run', 'summary': 'starting'},
          ),
        );
        pair.serverWrite(
          _toolFrame(
            type: 'tool.call.progress',
            messageId: 'frame-2',
            toolCallId: _toolCallId,
            payload: {'summary': 'running tests…'},
          ),
        );
        await Future<void>.delayed(const Duration(milliseconds: 10));

        final calls = channel.state.messages
            .where((m) => m.kind == NavivoxMessageKind.toolCall)
            .toList();
        expect(
          calls.length,
          1,
          reason: 'progress should not duplicate the message',
        );
        expect(calls.single.toolCall!.status, 'progress');
        expect(calls.single.toolCall!.summary, 'running tests…');

        await channel.close();
      },
    );

    test(
      'tool.call.completed marks status completed and preserves summary',
      () async {
        final pair = InMemoryByteTransportPair();
        final channel = SshNavivoxChannel(transport: pair.client);

        pair.serverWrite(
          _toolFrame(
            type: 'tool.call.started',
            messageId: 'frame-1',
            toolCallId: _toolCallId,
            payload: {'name': 'shell.run', 'summary': 'go test ./...'},
          ),
        );
        pair.serverWrite(
          _toolFrame(
            type: 'tool.call.completed',
            messageId: 'frame-2',
            toolCallId: _toolCallId,
            payload: {'summary': 'all tests passed (39/39)'},
          ),
        );
        await Future<void>.delayed(const Duration(milliseconds: 10));

        final call = channel.state.messages.firstWhere(
          (m) => m.kind == NavivoxMessageKind.toolCall,
        );
        expect(call.toolCall!.status, 'completed');
        expect(call.toolCall!.summary, 'all tests passed (39/39)');

        await channel.close();
      },
    );

    test(
      'tool.call.failed and .cancelled and .blocked land as terminal statuses',
      () async {
        for (final entry in {
          'tool.call.failed': 'failed',
          'tool.call.cancelled': 'cancelled',
          'tool.call.blocked': 'blocked',
        }.entries) {
          final pair = InMemoryByteTransportPair();
          final channel = SshNavivoxChannel(transport: pair.client);

          pair.serverWrite(
            _toolFrame(
              type: 'tool.call.started',
              messageId: 'frame-1',
              toolCallId: _toolCallId,
              payload: {'name': 'shell.run', 'summary': 'starting'},
            ),
          );
          pair.serverWrite(
            _toolFrame(
              type: entry.key,
              messageId: 'frame-2',
              toolCallId: _toolCallId,
              payload: {'summary': 'no good'},
            ),
          );
          await Future<void>.delayed(const Duration(milliseconds: 10));

          final call = channel.state.messages.firstWhere(
            (m) => m.kind == NavivoxMessageKind.toolCall,
          );
          expect(call.toolCall!.status, entry.value);

          await channel.close();
        }
      },
    );
  });

  group('SshNavivoxChannel approval flow', () {
    test(
      'tool.approval.requested is emitted on approvalRequests stream',
      () async {
        final pair = InMemoryByteTransportPair();
        final channel = SshNavivoxChannel(transport: pair.client);
        final received = <NavivoxApprovalRequest>[];
        final sub = channel.approvalRequests.listen(received.add);

        pair.serverWrite(
          _toolFrame(
            type: 'tool.approval.requested',
            messageId: 'frame-1',
            toolCallId: _toolCallId,
            approvalId: 'ap-7',
            payload: {
              'prompt': 'Allow shell.run to delete /tmp/x?',
              'risk': 'medium',
            },
          ),
        );
        await Future<void>.delayed(const Duration(milliseconds: 10));

        expect(received.length, 1);
        expect(received.single.id, 'ap-7');
        expect(received.single.toolCallId, _toolCallId);
        expect(received.single.prompt, contains('shell.run'));
        expect(received.single.risk, 'medium');

        await sub.cancel();
        await channel.close();
      },
    );

    test('respondToApproval emits a tool.approval.responded frame', () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      channel.respondToApproval(approvalId: 'ap-7', approved: true);

      final wire = await pair.serverReadOne();
      final frame = NavivoxFrameCodec.decode(wire);
      expect(frame.type, 'tool.approval.responded');
      expect(frame.metadata['approval_id'], 'ap-7');
      expect(jsonDecode(utf8.decode(frame.payload)), {'approved': true});

      await channel.close();
    });
  });
}
