import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/channel/navivox_channel.dart';
import '../../../core/channel/navivox_channel_provider.dart';
import '../../voice/services/voice_capture_service.dart';
import '../widgets/approval_banner.dart';
import '../widgets/simple_chat_adapter.dart';

/// Voice-capture service used by the chat input bar. Override in tests with
/// [FakeVoiceCaptureService]; production wiring slots in
/// [RecordVoiceCaptureService] once the real mic + STT plugins land.
final chatVoiceCaptureServiceProvider = Provider<VoiceCaptureService?>(
  (_) => null,
);

class ChatScreen extends ConsumerStatefulWidget {
  const ChatScreen({this.serverId, this.profileId, super.key});

  final String? serverId;
  final String? profileId;

  @override
  ConsumerState<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends ConsumerState<ChatScreen> {
  NavivoxChannel? _subscribed;

  void _onChannelChanged() {
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    _subscribed?.removeListener(_onChannelChanged);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final channel = ref.watch(navivoxChannelProvider);
    if (!identical(_subscribed, channel)) {
      _subscribed?.removeListener(_onChannelChanged);
      channel.addListener(_onChannelChanged);
      _subscribed = channel;
    }
    _syncRouteProfile(channel);

    final state = channel.state;
    final server = state.activeServer;
    final activeProfile = state.activeProfileContact;
    final voiceService = ref.watch(chatVoiceCaptureServiceProvider);
    final selectedAgent = state.selectedAgentId == null
        ? null
        : state.agents
              .where((agent) => agent.id == state.selectedAgentId)
              .firstOrNull;

    return Scaffold(
      appBar: AppBar(
        leading: const BackButton(),
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(activeProfile?.displayName ?? server?.name ?? 'Chats'),
            if (activeProfile != null)
              Text(
                '${activeProfile.serverLabel} • ${_profileHealthLabel(activeProfile.health)}',
                style: Theme.of(context).textTheme.labelMedium,
              )
            else if (server != null)
              Text(
                server.status,
                style: Theme.of(context).textTheme.labelMedium,
              ),
          ],
        ),
        actions: [
          if (activeProfile != null)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12),
              child: Center(
                child: Chip(
                  key: const ValueKey('chat-active-profile'),
                  avatar: const Icon(Icons.person, size: 16),
                  label: Text(activeProfile.serverLabel),
                ),
              ),
            ),
          if (selectedAgent != null)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12),
              child: Center(
                child: Chip(
                  key: const ValueKey('chat-active-agent'),
                  avatar: const Icon(Icons.smart_toy, size: 16),
                  label: Text(selectedAgent.name),
                ),
              ),
            ),
        ],
      ),
      body: Column(
        children: [
          ApprovalBanner(channel: channel),
          Expanded(
            child: SimpleChatAdapter(
              messages: state.messages,
              onSend: channel.sendText,
              voiceCaptureService: voiceService,
              onVoice: (capture) => channel.sendVoice(
                audio: capture.audio,
                transcript: capture.transcript,
                duration: capture.duration,
                confidence: capture.confidence,
              ),
            ),
          ),
        ],
      ),
    );
  }

  String _profileHealthLabel(NavivoxProfileHealth health) {
    return switch (health) {
      NavivoxProfileHealth.online => 'online',
      NavivoxProfileHealth.offline => 'offline',
      NavivoxProfileHealth.needsAuth => 'auth required',
      NavivoxProfileHealth.warning => 'warning',
    };
  }

  void _syncRouteProfile(NavivoxChannel channel) {
    final serverId = widget.serverId;
    final profileId = widget.profileId;
    if (serverId == null || profileId == null) return;

    final key = '$serverId::$profileId';
    if (channel.state.selectedProfileContactKey == key &&
        channel.state.activeServerId == serverId) {
      return;
    }
    final exists = channel.state.profileContacts.any(
      (contact) => contact.serverId == serverId && contact.profileId == profileId,
    );
    if (!exists) return;

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (channel.state.selectedProfileContactKey == key &&
          channel.state.activeServerId == serverId) {
        return;
      }
      channel.selectProfileContact(serverId: serverId, profileId: profileId);
    });
  }
}
