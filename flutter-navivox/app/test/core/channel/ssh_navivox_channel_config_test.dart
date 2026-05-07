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
  group('SshNavivoxChannel config admin', () {
    test('config.schema response is captured into state.configSchema',
        () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      pair.serverWrite(_frame('config.schema',
          messageId: 'f-1',
          payload: {
            'fields': [
              {'name': 'provider', 'type': 'string', 'required': true},
              {'name': 'temperature', 'type': 'number', 'required': false},
            ],
          }));
      await Future<void>.delayed(const Duration(milliseconds: 10));

      expect(channel.state.configSchema?['fields'], isA<List>());
      await channel.close();
    });

    test('config.get response populates state.configValues', () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      pair.serverWrite(_frame('config.get',
          messageId: 'f-1',
          payload: {
            'values': {
              'provider': 'anthropic',
              'temperature': 0.4,
            },
          }));
      await Future<void>.delayed(const Duration(milliseconds: 10));

      expect(channel.state.configValues, {
        'provider': 'anthropic',
        'temperature': 0.4,
      });
      await channel.close();
    });

    test('sendConfigSet writes a config.set frame with the value', () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      channel.sendConfigSet(field: 'temperature', value: 0.7);

      final wire = await pair.serverReadOne();
      final frame = NavivoxFrameCodec.decode(wire);
      expect(frame.type, 'config.set');
      expect(jsonDecode(utf8.decode(frame.payload)), {
        'field': 'temperature',
        'value': 0.7,
      });

      await channel.close();
    });

    test('sendConfigSecretSet writes a config.secret.set frame and never echoes the secret back to local state',
        () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      channel.sendConfigSecretSet(name: 'TELEGRAM_BOT_TOKEN', secret: 'super-secret');

      final wire = await pair.serverReadOne();
      final frame = NavivoxFrameCodec.decode(wire);
      expect(frame.type, 'config.secret.set');
      final body = jsonDecode(utf8.decode(frame.payload)) as Map<String, Object?>;
      expect(body['name'], 'TELEGRAM_BOT_TOKEN');
      expect(body['value'], 'super-secret');

      // Local state must never carry the raw secret value.
      expect(channel.state.configValues['TELEGRAM_BOT_TOKEN'], isNull);
      await channel.close();
    });

    test('config.diff response is captured into state.configDiff', () async {
      final pair = InMemoryByteTransportPair();
      final channel = SshNavivoxChannel(transport: pair.client);

      pair.serverWrite(_frame('config.diff',
          messageId: 'f-1',
          payload: {
            'changes': [
              {
                'field': 'temperature',
                'before': 0.4,
                'after': 0.7,
              },
            ],
          }));
      await Future<void>.delayed(const Duration(milliseconds: 10));

      expect(channel.state.configDiff?['changes'], isA<List>());
      await channel.close();
    });
  });
}
