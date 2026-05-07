import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/channel/ssh_navivox_channel.dart';
import 'package:navivox/core/protocol/navivox_frame.dart';
import 'package:navivox/core/transport/in_memory_transport.dart';

Uint8List _frame(
  String type, {
  required String messageId,
  required Object payload,
  Map<String, Object?> metadata = const {},
}) {
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
  group('SshNavivoxChannel agent admin', () {
    test('agent.list payload populates state.agents', () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      pair.serverWrite(_frame('agent.list',
          messageId: 'f-1',
          payload: {
            'agents': [
              {'id': 'default', 'name': 'Default', 'status': 'active'},
              {'id': 'arch', 'name': 'Architect', 'status': 'active'},
              {'id': 'old', 'name': 'Legacy', 'status': 'archived'},
            ],
          }));
      await Future<void>.delayed(const Duration(milliseconds: 10));

      expect(channel.state.agents.map((a) => a.id).toList(),
          ['default', 'arch', 'old']);
      expect(channel.state.agents[0].name, 'Default');
      expect(channel.state.agents[2].status, 'archived');

      await channel.close();
    });

    test('agent.list ignored when payload is malformed', () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      pair.serverWrite(_frame('agent.list',
          messageId: 'f-1', payload: {'wrong_key': 1}));
      await Future<void>.delayed(const Duration(milliseconds: 10));

      expect(channel.state.agents, isEmpty);
      await channel.close();
    });

    test('selectAgent emits an agent.select request frame', () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      channel.selectAgent('arch');

      final wire = await pair.serverReadOne();
      final frame = NavivoxFrameCodec.decode(wire);
      expect(frame.type, 'agent.select');
      expect(jsonDecode(utf8.decode(frame.payload)), {'agent_id': 'arch'});

      await channel.close();
    });

    test('agent.select response updates state.selectedAgentId', () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      pair.serverWrite(_frame('agent.select',
          messageId: 'f-1', payload: {'agent_id': 'arch'}));
      await Future<void>.delayed(const Duration(milliseconds: 10));

      expect(channel.state.selectedAgentId, 'arch');
      await channel.close();
    });

    test('requestAgentList emits an agent.list request frame', () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      channel.requestAgentList();

      final wire = await pair.serverReadOne();
      final frame = NavivoxFrameCodec.decode(wire);
      expect(frame.type, 'agent.list');
      // Empty payload is acceptable for the request side.
      expect(frame.payload.length, lessThanOrEqualTo(2));

      await channel.close();
    });
  });
}
