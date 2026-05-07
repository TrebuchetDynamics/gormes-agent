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

class NavivoxChannelState {
  const NavivoxChannelState({
    this.servers = const [],
    this.activeServerId,
    this.messages = const [],
  });

  final List<NavivoxServer> servers;
  final String? activeServerId;
  final List<NavivoxChatMessage> messages;

  bool get hasServers => servers.isNotEmpty;
  NavivoxServer? get activeServer =>
      servers.where((server) => server.id == activeServerId).firstOrNull;

  NavivoxChannelState copyWith({
    List<NavivoxServer>? servers,
    String? activeServerId,
    List<NavivoxChatMessage>? messages,
  }) {
    return NavivoxChannelState(
      servers: servers ?? this.servers,
      activeServerId: activeServerId ?? this.activeServerId,
      messages: messages ?? this.messages,
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
}
