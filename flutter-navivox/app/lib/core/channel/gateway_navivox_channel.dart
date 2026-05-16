import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:uuid/uuid.dart';

import '../gateway/navivox_gateway_client.dart';
import '../gateway/navivox_gateway_protocol.dart';
import '../protocol/navivox_event.dart';
import 'navivox_channel.dart';

class GatewayNavivoxChannel extends ChangeNotifier implements NavivoxChannel {
  GatewayNavivoxChannel({Uuid? uuid, DateTime Function()? clock})
    : _uuid = uuid ?? const Uuid(),
      _clock = clock ?? DateTime.now;

  final Uuid _uuid;
  final DateTime Function() _clock;
  final StreamController<NavivoxApprovalRequest> _approvals =
      StreamController<NavivoxApprovalRequest>.broadcast();

  NavivoxGatewaySocket? _socket;
  StreamSubscription<NavivoxGatewayEvent>? _events;
  NavivoxChannelState _state = const NavivoxChannelState();
  String? _activeSessionId;
  final Map<String, String> _assistantMessageIds = {};

  @override
  NavivoxChannelState get state => _state;

  @override
  Stream<NavivoxApprovalRequest> get approvalRequests => _approvals.stream;

  Future<void> connect(NavivoxGatewayConfig config) async {
    await disconnect();
    final client = NavivoxGatewayClient(config: config);
    await client.status();
    final socket = await client.connectStream();
    _socket = socket;
    _events = client
        .decodeEvents(socket.events)
        .listen(
          _onEvent,
          onError: (Object error) =>
              _appendSystemMessage('Gateway stream error'),
          onDone: () => _setServerStatus('Gateway disconnected'),
        );
    final server = NavivoxServer(
      id: 'navivox-gateway',
      name: 'Gormes Gateway',
      status: 'Gateway online - ${config.baseUri.host}:${config.baseUri.port}',
    );
    _state = _state.copyWith(servers: [server], activeServerId: server.id);
    notifyListeners();
  }

  Future<void> disconnect() async {
    await _events?.cancel();
    _events = null;
    await _socket?.close();
    _socket = null;
    _assistantMessageIds.clear();
  }

  @override
  void sendText(String text) {
    final trimmed = text.trim();
    if (trimmed.isEmpty) return;

    final socket = _socket;
    if (socket == null) {
      _appendSystemMessage('Gateway is not connected.');
      return;
    }

    final requestId = _uuid.v4();
    _appendMessage(
      NavivoxChatMessage(
        id: requestId,
        author: NavivoxMessageAuthor.user,
        kind: NavivoxMessageKind.text,
        createdAt: _clock(),
        text: trimmed,
      ),
    );
    socket.add(
      jsonEncode(
        NavivoxGatewayMessage.startTurn(
          requestId: requestId,
          sessionId: _activeSessionId,
          text: trimmed,
        ).body,
      ),
    );
  }

  @override
  void sendVoice({
    required Uint8List audio,
    required String transcript,
    required Duration duration,
    required double confidence,
  }) {
    final trimmed = transcript.trim();
    if (trimmed.isEmpty) {
      _appendSystemMessage('Voice transcript is empty.');
      return;
    }

    final socket = _socket;
    if (socket == null) {
      _appendSystemMessage('Gateway is not connected.');
      return;
    }

    final requestId = _uuid.v4();
    _appendMessage(
      NavivoxChatMessage(
        id: requestId,
        author: NavivoxMessageAuthor.user,
        kind: NavivoxMessageKind.voice,
        createdAt: _clock(),
        voice: NavivoxVoiceMessage(
          duration: duration,
          transcript: trimmed,
          confidence: confidence,
        ),
      ),
    );
    socket.add(
      jsonEncode(
        NavivoxGatewayMessage.startTurn(
          requestId: requestId,
          sessionId: _activeSessionId,
          text: trimmed,
        ).body,
      ),
    );
  }

  @override
  void respondToApproval({required String approvalId, required bool approved}) {
    _appendSystemMessage(
      'Tool approvals are not available on this channel yet.',
    );
  }

  @override
  void requestAgentList() {
    _appendSystemMessage('Agent listing is not available on this channel yet.');
  }

  @override
  void selectAgent(String agentId) {
    _state = _state.copyWith(selectedAgentId: agentId);
    notifyListeners();
  }

  @override
  void sendConfigSet({required String field, required Object? value}) {
    _appendSystemMessage(
      'Config editing is not available on this channel yet.',
    );
  }

  @override
  void sendConfigSecretSet({required String name, required String secret}) {
    _appendSystemMessage(
      'Secret editing is not available on this channel yet.',
    );
  }

  @override
  void dispose() {
    unawaited(disconnect());
    unawaited(_approvals.close());
    super.dispose();
  }

  void _onEvent(NavivoxGatewayEvent event) {
    switch (event.type) {
      case 'pong':
        return;
      case 'session_started':
        _activeSessionId = event.sessionId ?? _activeSessionId;
      case 'assistant_delta':
        _appendAssistantDelta(event);
      case 'assistant_message':
        _upsertAssistantMessage(event);
      case 'tool_call_started':
        _upsertToolCall(event, 'started');
      case 'tool_call_finished':
        _upsertToolCall(event, event.status ?? 'finished');
      case 'error':
        _appendSystemMessage(event.message ?? 'Gateway error');
      case 'done':
        return;
      default:
        return;
    }
  }

  void _appendAssistantDelta(NavivoxGatewayEvent event) {
    final requestId = event.requestId ?? _uuid.v4();
    final messageId = _assistantMessageIds.putIfAbsent(
      requestId,
      () => 'assistant-$requestId',
    );
    final index = _state.messages.indexWhere((m) => m.id == messageId);
    if (index < 0) {
      _appendMessage(
        NavivoxChatMessage(
          id: messageId,
          author: NavivoxMessageAuthor.assistant,
          kind: NavivoxMessageKind.text,
          createdAt: _clock(),
          text: event.text ?? '',
        ),
      );
      return;
    }
    final existing = _state.messages[index];
    _replaceMessage(
      index,
      NavivoxChatMessage(
        id: existing.id,
        author: existing.author,
        kind: existing.kind,
        createdAt: existing.createdAt,
        text: '${existing.text ?? ''}${event.text ?? ''}',
      ),
    );
  }

  void _upsertAssistantMessage(NavivoxGatewayEvent event) {
    final requestId = event.requestId ?? _uuid.v4();
    final messageId = _assistantMessageIds.putIfAbsent(
      requestId,
      () => 'assistant-$requestId',
    );
    final index = _state.messages.indexWhere((m) => m.id == messageId);
    final message = NavivoxChatMessage(
      id: messageId,
      author: NavivoxMessageAuthor.assistant,
      kind: NavivoxMessageKind.text,
      createdAt: index >= 0 ? _state.messages[index].createdAt : _clock(),
      text: event.text ?? '',
    );
    if (index >= 0) {
      _replaceMessage(index, message);
    } else {
      _appendMessage(message);
    }
  }

  void _upsertToolCall(NavivoxGatewayEvent event, String status) {
    final toolCallId = event.toolCallId ?? 'tool-${_uuid.v4()}';
    final index = _state.messages.indexWhere((m) => m.id == toolCallId);
    final prior = index >= 0 ? _state.messages[index].toolCall : null;
    final message = NavivoxChatMessage(
      id: toolCallId,
      author: NavivoxMessageAuthor.assistant,
      kind: NavivoxMessageKind.toolCall,
      createdAt: index >= 0 ? _state.messages[index].createdAt : _clock(),
      toolCall: NavivoxToolCall(
        name: event.toolName ?? prior?.name ?? 'tool',
        status: status,
        summary: prior?.summary ?? '',
        artifacts: prior?.artifacts ?? const [],
      ),
    );
    if (index >= 0) {
      _replaceMessage(index, message);
    } else {
      _appendMessage(message);
    }
  }

  void _appendSystemMessage(String text) {
    _appendMessage(
      NavivoxChatMessage(
        id: _uuid.v4(),
        author: NavivoxMessageAuthor.system,
        kind: NavivoxMessageKind.text,
        createdAt: _clock(),
        text: text,
      ),
    );
  }

  void _setServerStatus(String status) {
    if (_state.servers.isEmpty) return;
    final server = _state.servers.first;
    _state = _state.copyWith(
      servers: [
        NavivoxServer(id: server.id, name: server.name, status: status),
      ],
    );
    notifyListeners();
  }

  void _appendMessage(NavivoxChatMessage message) {
    _state = _state.copyWith(messages: [..._state.messages, message]);
    notifyListeners();
  }

  void _replaceMessage(int index, NavivoxChatMessage message) {
    final messages = [..._state.messages];
    messages[index] = message;
    _state = _state.copyWith(messages: messages);
    notifyListeners();
  }
}
