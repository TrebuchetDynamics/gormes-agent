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
    final contactPayloads = await client.profileContacts();
    final profileContacts = contactPayloads
        .map(_profileContactFromJson)
        .toList(growable: false);
    final contacts = profileContacts.isEmpty
        ? [_fallbackProfileContact()]
        : profileContacts;
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
    _state = _state.copyWith(
      servers: _serversFromProfileContacts(contacts, config),
      activeServerId: contacts.first.serverId,
      profileContacts: contacts,
      selectedProfileContactKey: contacts.first.key,
    );
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
    final activeProfile = _state.activeProfileContact;
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
          metadata: _turnMetadata(activeProfile),
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
    final activeProfile = _state.activeProfileContact;
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
          metadata: _turnMetadata(activeProfile),
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
  void selectProfileContact({
    required String serverId,
    required String profileId,
  }) {
    final key = '$serverId::$profileId';
    if (_state.selectedProfileContactKey == key &&
        _state.activeServerId == serverId) {
      return;
    }
    _state = _state.copyWith(
      selectedProfileContactKey: key,
      activeServerId: serverId,
    );
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
      case 'tool_call_updated':
        _upsertToolCall(event, event.status ?? 'updated');
      case 'tool_call_finished':
        _upsertToolCall(event, event.status ?? 'finished');
      case 'profile_contact_update':
        final contact = event.contact;
        if (contact != null) {
          _upsertProfileContact(_profileContactFromJson(contact));
        }
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
    final summary = event.message ?? event.text ?? prior?.summary ?? '';
    final message = NavivoxChatMessage(
      id: toolCallId,
      author: NavivoxMessageAuthor.assistant,
      kind: NavivoxMessageKind.toolCall,
      createdAt: index >= 0 ? _state.messages[index].createdAt : _clock(),
      toolCall: NavivoxToolCall(
        name: event.toolName ?? prior?.name ?? 'tool',
        status: status,
        summary: summary,
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

  void _upsertProfileContact(NavivoxProfileContact contact) {
    final contacts = [..._state.profileContacts];
    final index = contacts.indexWhere(
      (existing) => existing.key == contact.key,
    );
    if (index >= 0) {
      contacts[index] = contact;
    } else {
      contacts.add(contact);
    }
    final servers = _upsertServer(_state.servers, contact);
    _state = _state.copyWith(
      servers: servers,
      activeServerId: _state.activeServerId ?? contact.serverId,
      profileContacts: contacts,
      selectedProfileContactKey:
          _state.selectedProfileContactKey ?? contact.key,
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

  Map<String, Object?> _turnMetadata(NavivoxProfileContact? profile) {
    return {
      'client': 'navivox',
      'platform': 'flutter',
      if (profile != null) ...{
        'server_id': profile.serverId,
        'profile_id': profile.profileId,
      },
    };
  }

  NavivoxProfileContact _fallbackProfileContact() {
    return NavivoxProfileContact(
      serverId: 'navivox-gateway',
      profileId: 'default',
      displayName: 'Default profile',
      serverLabel: 'Gormes Gateway',
      health: NavivoxProfileHealth.online,
      latestPreview: 'Gateway online',
      latestPreviewKind: 'status',
      workspaceRootCount: 1,
      workspaceRootsOk: true,
      micAvailable: true,
    );
  }

  List<NavivoxServer> _serversFromProfileContacts(
    List<NavivoxProfileContact> contacts,
    NavivoxGatewayConfig config,
  ) {
    final servers = <String, NavivoxServer>{};
    for (final contact in contacts) {
      servers.putIfAbsent(
        contact.serverId,
        () => NavivoxServer(
          id: contact.serverId,
          name: contact.serverLabel,
          status: _serverStatus(contact, config),
        ),
      );
    }
    return servers.values.toList(growable: false);
  }

  List<NavivoxServer> _upsertServer(
    List<NavivoxServer> servers,
    NavivoxProfileContact contact,
  ) {
    final index = servers.indexWhere((server) => server.id == contact.serverId);
    if (index >= 0) return servers;
    return [
      ...servers,
      NavivoxServer(
        id: contact.serverId,
        name: contact.serverLabel,
        status: _profileHealthStatus(contact),
      ),
    ];
  }

  String _serverStatus(
    NavivoxProfileContact contact,
    NavivoxGatewayConfig config,
  ) {
    if (contact.serverId == 'navivox-gateway') {
      return 'Gateway online - ${config.baseUri.host}:${config.baseUri.port}';
    }
    return _profileHealthStatus(contact);
  }

  String _profileHealthStatus(NavivoxProfileContact contact) {
    return switch (contact.health) {
      NavivoxProfileHealth.online => 'Gateway online',
      NavivoxProfileHealth.offline => 'Gateway offline',
      NavivoxProfileHealth.needsAuth => 'Provider auth required',
      NavivoxProfileHealth.warning => 'Profile warning',
    };
  }

  NavivoxProfileContact _profileContactFromJson(Map<String, Object?> json) {
    final serverId = _stringFromJson(
      json['server_id'],
      fallback: 'navivox-gateway',
    );
    final profileId = _stringFromJson(json['profile_id'], fallback: 'default');
    final serverLabel = _stringFromJson(
      json['server_label'],
      fallback: 'Gormes Gateway',
    );
    return NavivoxProfileContact(
      serverId: serverId,
      profileId: profileId,
      displayName: _stringFromJson(
        json['display_name'],
        fallback: profileId == 'default' ? 'Default profile' : profileId,
      ),
      serverLabel: serverLabel,
      health: _profileHealthFromJson(json['health']),
      latestPreview: _stringFromJson(
        json['latest_preview'],
        fallback: 'Profile ready',
      ),
      latestPreviewKind: _stringFromJson(
        json['latest_preview_kind'],
        fallback: 'status',
      ),
      latestAt: _dateFromJson(json['latest_preview_at']),
      workspaceRootCount: _intFromJson(json['workspace_root_count']),
      workspaceRootsOk: _boolFromJson(
        json['workspace_roots_ok'],
        fallback: true,
      ),
      workspaceRootsWarning: _intFromJson(json['workspace_roots_warning']),
      workspaceRootsError: _intFromJson(json['workspace_roots_error']),
      attentionBadges: _stringListFromJson(json['attention_badges']),
      micAvailable: _boolFromJson(json['mic_available']),
      activeTurnState: _stringFromJson(
        json['active_turn_state'],
        fallback: 'idle',
      ),
      avatarSeed: _stringFromJson(
        json['avatar_seed'],
        fallback: '$serverId:$profileId',
      ),
    );
  }

  NavivoxProfileHealth _profileHealthFromJson(Object? value) {
    return switch (value?.toString().trim().toLowerCase()) {
      'offline' => NavivoxProfileHealth.offline,
      'needs_auth' ||
      'needsauth' ||
      'needs-auth' => NavivoxProfileHealth.needsAuth,
      'warning' => NavivoxProfileHealth.warning,
      _ => NavivoxProfileHealth.online,
    };
  }

  DateTime? _dateFromJson(Object? value) {
    final text = value?.toString().trim();
    if (text == null || text.isEmpty) return null;
    return DateTime.tryParse(text);
  }

  String _stringFromJson(Object? value, {required String fallback}) {
    final text = value?.toString().trim();
    if (text == null || text.isEmpty) return fallback;
    return text;
  }

  int _intFromJson(Object? value) {
    if (value is num) return value.toInt();
    return int.tryParse(value?.toString() ?? '') ?? 0;
  }

  bool _boolFromJson(Object? value, {bool fallback = false}) {
    if (value is bool) return value;
    final text = value?.toString().trim().toLowerCase();
    if (text == 'true') return true;
    if (text == 'false') return false;
    return fallback;
  }

  List<String> _stringListFromJson(Object? value) {
    if (value is! List) return const [];
    return value
        .map((item) => item.toString().trim())
        .where((item) => item.isNotEmpty)
        .toList(growable: false);
  }
}
