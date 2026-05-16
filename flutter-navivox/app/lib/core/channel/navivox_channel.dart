import 'package:flutter/foundation.dart';

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

enum NavivoxProfileHealth { online, offline, needsAuth, warning }

class NavivoxProfileContact {
  const NavivoxProfileContact({
    required this.serverId,
    required this.profileId,
    required this.displayName,
    required this.serverLabel,
    required this.health,
    required this.latestPreview,
    this.latestAt,
    this.workspaceRootCount = 0,
    this.workspaceRootsOk = true,
    this.attentionBadges = const [],
    this.micAvailable = false,
    String? avatarSeed,
  }) : avatarSeed = avatarSeed ?? '$serverId:$profileId';

  final String serverId;
  final String profileId;
  final String displayName;
  final String serverLabel;
  final NavivoxProfileHealth health;
  final String latestPreview;
  final DateTime? latestAt;
  final int workspaceRootCount;
  final bool workspaceRootsOk;
  final List<String> attentionBadges;
  final bool micAvailable;
  final String avatarSeed;

  String get key => '$serverId::$profileId';
}

class NavivoxChannelState {
  const NavivoxChannelState({
    this.servers = const [],
    this.activeServerId,
    this.messages = const [],
    this.agents = const [],
    this.selectedAgentId,
    this.profileContacts = const [],
    this.selectedProfileContactKey,
    this.configSchema,
    this.configValues = const {},
    this.configDiff,
  });

  final List<NavivoxServer> servers;
  final String? activeServerId;
  final List<NavivoxChatMessage> messages;
  final List<NavivoxAgent> agents;
  final String? selectedAgentId;
  final List<NavivoxProfileContact> profileContacts;
  final String? selectedProfileContactKey;
  final Map<String, Object?>? configSchema;
  final Map<String, Object?> configValues;
  final Map<String, Object?>? configDiff;

  bool get hasServers => servers.isNotEmpty;
  NavivoxServer? get activeServer =>
      servers.where((server) => server.id == activeServerId).firstOrNull;
  NavivoxProfileContact? get activeProfileContact => profileContacts
      .where((contact) => contact.key == selectedProfileContactKey)
      .firstOrNull;

  NavivoxChannelState copyWith({
    List<NavivoxServer>? servers,
    String? activeServerId,
    List<NavivoxChatMessage>? messages,
    List<NavivoxAgent>? agents,
    String? selectedAgentId,
    List<NavivoxProfileContact>? profileContacts,
    String? selectedProfileContactKey,
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
      profileContacts: profileContacts ?? this.profileContacts,
      selectedProfileContactKey:
          selectedProfileContactKey ?? this.selectedProfileContactKey,
      configSchema: configSchema ?? this.configSchema,
      configValues: configValues ?? this.configValues,
      configDiff: configDiff ?? this.configDiff,
    );
  }
}

abstract interface class NavivoxChannel implements Listenable {
  NavivoxChannelState get state;
  Stream<NavivoxApprovalRequest> get approvalRequests;
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
  void selectProfileContact({
    required String serverId,
    required String profileId,
  });
  void sendConfigSet({required String field, required Object? value});
  void sendConfigSecretSet({required String name, required String secret});
}
