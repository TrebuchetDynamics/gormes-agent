import '../protocol/navivox_event.dart';

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
  void enterFakeServerMode();
  void sendText(String text);
}
