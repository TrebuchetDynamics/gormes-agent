import 'dart:typed_data';

import 'package:navivox/core/channel/gateway_navivox_channel.dart';
import 'package:navivox/core/channel/navivox_channel.dart';
import 'package:navivox/core/gateway/navivox_gateway_protocol.dart';
import 'package:navivox/core/protocol/navivox_event.dart';

class ConnectAndTalkChannel extends GatewayNavivoxChannel {
  NavivoxChannelState _state = const NavivoxChannelState();
  NavivoxGatewayConfig? connectedConfig;
  final List<String> sentTexts = [];

  @override
  NavivoxChannelState get state => _state;

  @override
  Future<void> connect(NavivoxGatewayConfig config) async {
    connectedConfig = config;
    const server = NavivoxServer(
      id: 'navivox-gateway',
      name: 'Gormes Gateway',
      status: 'Gateway online - 127.0.0.1:8765',
    );
    _state = _state.copyWith(servers: [server], activeServerId: server.id);
    notifyListeners();
  }

  @override
  void sendText(String text) {
    final trimmed = text.trim();
    if (trimmed.isEmpty) return;
    sentTexts.add(trimmed);
    final now = DateTime(2026, 5, 16, 9, 41);
    _state = _state.copyWith(
      messages: [
        ..._state.messages,
        NavivoxChatMessage(
          id: 'user-${sentTexts.length}',
          author: NavivoxMessageAuthor.user,
          kind: NavivoxMessageKind.text,
          createdAt: now,
          text: trimmed,
        ),
        NavivoxChatMessage(
          id: 'assistant-${sentTexts.length}',
          author: NavivoxMessageAuthor.assistant,
          kind: NavivoxMessageKind.text,
          createdAt: now,
          text: 'hello from gateway',
        ),
      ],
    );
    notifyListeners();
  }

  @override
  void sendVoice({
    required Uint8List audio,
    required String transcript,
    required Duration duration,
    required double confidence,
  }) {
    sendText(transcript);
  }
}

class FailingConnectChannel extends GatewayNavivoxChannel {
  @override
  Future<void> connect(NavivoxGatewayConfig config) async {
    throw StateError('connection failed for ${config.token}');
  }
}
