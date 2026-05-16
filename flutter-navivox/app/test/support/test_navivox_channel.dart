import 'dart:async';

import 'package:flutter/foundation.dart';

import 'package:navivox/core/channel/navivox_channel.dart';
import 'package:navivox/core/protocol/navivox_event.dart';

/// A minimal in-memory [NavivoxChannel] for widget tests. Records calls into
/// public fields and surfaces a mutable [NavivoxChannelState] so tests can
/// seed servers/agents/messages without touching the wire layer.
class TestNavivoxChannel extends ChangeNotifier implements NavivoxChannel {
  TestNavivoxChannel({
    NavivoxChannelState initial = const NavivoxChannelState(),
  }) : _state = initial;

  NavivoxChannelState _state;
  final StreamController<NavivoxApprovalRequest> _approvals =
      StreamController<NavivoxApprovalRequest>.broadcast();
  int _messageCounter = 0;

  final List<String> sentTexts = [];
  final List<({String text, String? serverId, String? profileId})>
  sentTextCalls = [];
  final List<SentVoiceCall> sentVoiceCalls = [];
  final List<({String approvalId, bool approved})> approvalResponses = [];
  final List<({String field, Object? value})> configSetCalls = [];
  final List<({String name, String secret})> configSecretSetCalls = [];
  String? lastSelectedAgentId;
  ({String serverId, String profileId})? selectedProfileScope;
  int agentListRequests = 0;

  @override
  NavivoxChannelState get state => _state;

  set state(NavivoxChannelState next) {
    _state = next;
    notifyListeners();
  }

  void seedAgents(List<NavivoxAgent> agents, {String? selectedAgentId}) {
    state = _state.copyWith(
      agents: agents,
      selectedAgentId: selectedAgentId ?? _state.selectedAgentId,
    );
  }

  void seedServers(List<NavivoxServer> servers, {String? activeServerId}) {
    state = _state.copyWith(
      servers: servers,
      activeServerId: activeServerId ?? _state.activeServerId,
    );
  }

  void seedProfileContacts(
    List<NavivoxProfileContact> contacts, {
    String? selectedKey,
  }) {
    state = _state.copyWith(
      profileContacts: contacts,
      selectedProfileContactKey:
          selectedKey ?? _state.selectedProfileContactKey,
    );
  }

  void emitApprovalRequest(NavivoxApprovalRequest request) {
    _approvals.add(request);
  }

  void emitConfigSchema(Map<String, Object?> schema) {
    state = _state.copyWith(configSchema: schema);
  }

  void emitConfigValues(Map<String, Object?> values) {
    state = _state.copyWith(configValues: values);
  }

  void emitConfigDiff(Map<String, Object?> diff) {
    state = _state.copyWith(configDiff: diff);
  }

  @override
  Stream<NavivoxApprovalRequest> get approvalRequests => _approvals.stream;

  @override
  void sendText(String text) {
    final trimmed = text.trim();
    if (trimmed.isEmpty) return;

    sentTexts.add(trimmed);
    final active = _state.activeProfileContact;
    sentTextCalls.add((
      text: trimmed,
      serverId: active?.serverId,
      profileId: active?.profileId,
    ));
    state = _state.copyWith(
      messages: [
        ..._state.messages,
        NavivoxChatMessage(
          id: 'test-user-${++_messageCounter}',
          author: NavivoxMessageAuthor.user,
          kind: NavivoxMessageKind.text,
          createdAt: DateTime.utc(2026, 5, 16, 12, 0, _messageCounter),
          text: trimmed,
        ),
      ],
    );
  }

  @override
  void sendVoice({
    required Uint8List audio,
    required String transcript,
    required Duration duration,
    required double confidence,
  }) {
    sentVoiceCalls.add(
      SentVoiceCall(
        audio: audio,
        transcript: transcript,
        duration: duration,
        confidence: confidence,
      ),
    );
  }

  @override
  void respondToApproval({required String approvalId, required bool approved}) {
    approvalResponses.add((approvalId: approvalId, approved: approved));
  }

  @override
  void requestAgentList() => agentListRequests += 1;

  @override
  void selectAgent(String agentId) {
    lastSelectedAgentId = agentId;
    state = _state.copyWith(selectedAgentId: agentId);
  }

  @override
  void selectProfileContact({
    required String serverId,
    required String profileId,
  }) {
    selectedProfileScope = (serverId: serverId, profileId: profileId);
    state = _state.copyWith(
      activeServerId: serverId,
      selectedProfileContactKey: '$serverId::$profileId',
    );
  }

  @override
  void sendConfigSet({required String field, required Object? value}) {
    configSetCalls.add((field: field, value: value));
  }

  @override
  void sendConfigSecretSet({required String name, required String secret}) {
    configSecretSetCalls.add((name: name, secret: secret));
  }

  @override
  void dispose() {
    _approvals.close();
    super.dispose();
  }
}

class SentVoiceCall {
  const SentVoiceCall({
    required this.audio,
    required this.transcript,
    required this.duration,
    required this.confidence,
  });

  final Uint8List audio;
  final String transcript;
  final Duration duration;
  final double confidence;
}
