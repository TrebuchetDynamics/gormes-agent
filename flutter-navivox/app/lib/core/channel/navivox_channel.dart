import 'dart:typed_data';

import '../protocol/navivox_event.dart';

/// A pending approval request issued by the server while a tool call is mid-
/// flight. The user resolves it via [NavivoxChannel.respondToApproval].
class NavivoxApprovalRequest {
  const NavivoxApprovalRequest({
    required this.id,
    required this.toolCallId,
    required this.prompt,
    this.risk,
  });

  final String id;
  final String toolCallId;
  final String prompt;
  final String? risk;
}

class NavivoxServer {
  const NavivoxServer({
    required this.id,
    required this.name,
    required this.status,
  });

  final String id;
  final String name;
  final String status;
}

class NavivoxAgent {
  const NavivoxAgent({
    required this.id,
    required this.name,
    required this.status,
  });

  final String id;
  final String name;
  final String status;
}

class NavivoxChannelState {
  const NavivoxChannelState({
    this.servers = const [],
    this.activeServerId,
    this.messages = const [],
    this.agents = const [],
    this.selectedAgentId,
    this.configSchema,
    this.configValues = const {},
    this.configDiff,
  });

  final List<NavivoxServer> servers;
  final String? activeServerId;
  final List<NavivoxChatMessage> messages;
  final List<NavivoxAgent> agents;
  final String? selectedAgentId;
  final Map<String, Object?>? configSchema;
  final Map<String, Object?> configValues;
  final Map<String, Object?>? configDiff;

  bool get hasServers => servers.isNotEmpty;
  NavivoxServer? get activeServer =>
      servers.where((server) => server.id == activeServerId).firstOrNull;

  NavivoxChannelState copyWith({
    List<NavivoxServer>? servers,
    String? activeServerId,
    List<NavivoxChatMessage>? messages,
    List<NavivoxAgent>? agents,
    String? selectedAgentId,
    Map<String, Object?>? configSchema,
    Map<String, Object?>? configValues,
    Map<String, Object?>? configDiff,
  }) {
    return NavivoxChannelState(
      servers: servers ?? this.servers,
      activeServerId: activeServerId ?? this.activeServerId,
      messages: messages ?? this.messages,
      agents: agents ?? this.agents,
      selectedAgentId: selectedAgentId ?? this.selectedAgentId,
      configSchema: configSchema ?? this.configSchema,
      configValues: configValues ?? this.configValues,
      configDiff: configDiff ?? this.configDiff,
    );
  }
}

abstract interface class NavivoxChannel {
  NavivoxChannelState get state;
  Stream<NavivoxApprovalRequest> get approvalRequests;
  void enterFakeServerMode();
  void sendText(String text);
  void sendVoice({
    required Uint8List audio,
    required String transcript,
    required Duration duration,
    required double confidence,
  });
  void respondToApproval({required String approvalId, required bool approved});
  void requestAgentList();
  void selectAgent(String agentId);
  void sendConfigSet({required String field, required Object? value});
  void sendConfigSecretSet({required String name, required String secret});
}
