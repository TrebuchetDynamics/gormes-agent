import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/channel/navivox_channel.dart';
import '../../../core/channel/navivox_channel_provider.dart';
import '../../../core/protocol/navivox_event.dart';
import '../../../router/app_routes.dart';
import '../../settings/providers/voice_settings_provider.dart';
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
  const ChatScreen({
    this.serverId,
    this.profileId,
    this.voiceCaptureServiceOverride,
    this.voiceAutoSendGrace = const Duration(milliseconds: 800),
    this.voiceCommandTimeout = const Duration(seconds: 5),
    super.key,
  });

  final String? serverId;
  final String? profileId;
  final VoiceCaptureService? voiceCaptureServiceOverride;
  final Duration voiceAutoSendGrace;
  final Duration voiceCommandTimeout;

  @override
  ConsumerState<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends ConsumerState<ChatScreen> {
  NavivoxChannel? _subscribed;
  VoiceCapture? _pendingVoice;
  DateTime? _pendingVoiceCreatedAt;
  Timer? _pendingVoiceTimer;
  Timer? _commandModeTimer;
  bool _commandMode = false;
  bool _routeProfileSynced = false;
  String? _lastRouteProfileKey;
  String? _voiceNotice;

  void _onChannelChanged() {
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    _subscribed?.removeListener(_onChannelChanged);
    _pendingVoiceTimer?.cancel();
    _commandModeTimer?.cancel();
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
    final voiceService =
        widget.voiceCaptureServiceOverride ??
        ref.watch(chatVoiceCaptureServiceProvider);
    final voiceSettings = ref.watch(navivoxVoiceSettingsProvider);
    final voiceDisabledReason = _voiceDisabledReason(
      voiceService: voiceService,
      activeProfile: activeProfile,
      settings: voiceSettings,
    );
    final adapterMessages = [
      ...state.messages,
      if (_pendingVoice != null)
        NavivoxChatMessage(
          id: 'pending-voice',
          author: NavivoxMessageAuthor.user,
          kind: NavivoxMessageKind.voice,
          createdAt: _pendingVoiceCreatedAt ?? DateTime.now(),
          voice: NavivoxVoiceMessage(
            duration: _pendingVoice!.duration,
            transcript: _pendingVoice!.transcript,
            confidence: _pendingVoice!.confidence,
          ),
        ),
    ];
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
          _VoiceModeBanner(
            commandMode: _commandMode,
            disabledReason: voiceDisabledReason,
            notice: _voiceNotice,
            pending: _pendingVoice != null,
            ready:
                voiceService != null &&
                activeProfile != null &&
                voiceDisabledReason == null,
            canTrustServer:
                activeProfile != null &&
                voiceService != null &&
                !_voiceSettings(ref).isTrusted(activeProfile.serverId),
            onTrustServer: activeProfile == null
                ? null
                : () => ref
                      .read(navivoxVoiceSettingsProvider.notifier)
                      .setServerTrusted(activeProfile.serverId, true),
            onCancelPending: _cancelPendingVoice,
          ),
          Expanded(
            child: SimpleChatAdapter(
              messages: adapterMessages,
              onSend: (text) => _handleTextSubmit(channel, text),
              voiceCaptureService: voiceDisabledReason == null
                  ? voiceService
                  : null,
              onVoice: (capture) => _handleVoiceCapture(channel, capture),
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

  NavivoxVoiceSettings _voiceSettings(WidgetRef ref) {
    return ref.read(navivoxVoiceSettingsProvider);
  }

  String? _voiceDisabledReason({
    required VoiceCaptureService? voiceService,
    required NavivoxProfileContact? activeProfile,
    required NavivoxVoiceSettings settings,
  }) {
    if (voiceService == null) return 'device STT unavailable';
    if (!settings.continuousVoiceEnabled) return 'disabled in Settings';
    if (activeProfile == null) return 'select a profile';
    if (!settings.isTrusted(activeProfile.serverId)) {
      return 'trust ${activeProfile.serverLabel}';
    }
    if (activeProfile.health != NavivoxProfileHealth.online) {
      return _profileHealthLabel(activeProfile.health);
    }
    if (!activeProfile.micAvailable) return 'mic unavailable';
    return null;
  }

  void _handleTextSubmit(NavivoxChannel channel, String text) {
    if (_handleLocalCommand(channel, text, fromVoice: false)) return;
    channel.sendText(text);
  }

  void _handleVoiceCapture(NavivoxChannel channel, VoiceCapture capture) {
    if (_handleLocalCommand(channel, capture.transcript, fromVoice: true)) {
      return;
    }
    _pendingVoiceTimer?.cancel();
    setState(() {
      _pendingVoice = capture;
      _pendingVoiceCreatedAt = DateTime.now();
      _voiceNotice = 'Sending...';
    });
    _pendingVoiceTimer = Timer(widget.voiceAutoSendGrace, () {
      if (!mounted || _pendingVoice == null) return;
      final pending = _pendingVoice!;
      setState(() {
        _pendingVoice = null;
        _pendingVoiceCreatedAt = null;
        _voiceNotice = null;
      });
      channel.sendVoice(
        audio: pending.audio,
        transcript: pending.transcript,
        duration: pending.duration,
        confidence: pending.confidence,
      );
    });
  }

  void _cancelPendingVoice() {
    _pendingVoiceTimer?.cancel();
    setState(() {
      _pendingVoice = null;
      _pendingVoiceCreatedAt = null;
      _voiceNotice = 'Voice turn cancelled before server commit.';
    });
  }

  bool _handleLocalCommand(
    NavivoxChannel channel,
    String raw, {
    required bool fromVoice,
  }) {
    final body = _commandBody(raw, fromVoice: fromVoice);
    if (body == null) return false;
    if (body.isEmpty) {
      _enterCommandMode();
      return true;
    }
    _exitCommandMode(clearNotice: false);
    return _runCommandBody(channel, body);
  }

  String? _commandBody(String raw, {required bool fromVoice}) {
    final text = raw.trim();
    if (text.isEmpty) return null;
    final commandWord = _voiceSettings(ref).commandWord.toLowerCase();
    final lower = text.toLowerCase();
    if (_commandMode &&
        fromVoice &&
        !_startsWithCommandWord(lower, commandWord)) {
      return text;
    }
    if (!_startsWithCommandWord(lower, commandWord)) return null;
    return text.length == commandWord.length
        ? ''
        : text.substring(commandWord.length).trim();
  }

  bool _startsWithCommandWord(String lower, String commandWord) {
    if (lower == commandWord) return true;
    return lower.startsWith('$commandWord ');
  }

  void _enterCommandMode() {
    _commandModeTimer?.cancel();
    setState(() {
      _commandMode = true;
      _voiceNotice = 'Command mode';
    });
    _commandModeTimer = Timer(widget.voiceCommandTimeout, () {
      if (!mounted) return;
      setState(() {
        _commandMode = false;
        _voiceNotice = 'Command mode timed out.';
      });
    });
  }

  void _exitCommandMode({required bool clearNotice}) {
    _commandModeTimer?.cancel();
    if (!_commandMode && !clearNotice) return;
    setState(() {
      _commandMode = false;
      if (clearNotice) _voiceNotice = null;
    });
  }

  bool _runCommandBody(NavivoxChannel channel, String body) {
    final normalized = _normalizeCommand(body);
    switch (normalized) {
      case 'cancel':
        if (_pendingVoice != null) {
          _cancelPendingVoice();
        } else {
          channel.cancelActiveTurn();
          _showCommandMessage(
            'Cancel requested. Started side effects may still exist.',
          );
        }
        return true;
      case 'stop':
        channel.stopActiveTurn();
        _showCommandMessage(
          'Stop requested. Started side effects may still exist.',
        );
        return true;
      case 'settings':
        context.go(AppRoutes.settings);
        return true;
      case 'help':
        _showCommandMessage(
          'Voice commands: navi <profile>, cancel, stop, settings, help.',
        );
        return true;
    }
    return _switchProfileFromCommand(channel, body, normalized);
  }

  bool _switchProfileFromCommand(
    NavivoxChannel channel,
    String rawBody,
    String normalized,
  ) {
    final settings = _voiceSettings(ref);
    if (!settings.profileSwitchingEnabled) {
      _showCommandMessage('Voice profile switching is disabled.');
      return true;
    }
    final matches = channel.state.profileContacts
        .where((contact) => _contactCommandNames(contact).contains(normalized))
        .toList(growable: false);
    if (matches.length == 1) {
      final contact = matches.single;
      channel.selectProfileContact(
        serverId: contact.serverId,
        profileId: contact.profileId,
      );
      GoRouter.maybeOf(context)?.go(
        '/chats/${Uri.encodeComponent(contact.serverId)}/'
        '${Uri.encodeComponent(contact.profileId)}',
      );
      _showCommandMessage('Switched to ${contact.displayName}.');
      return true;
    }
    if (matches.length > 1) {
      _showCommandMessage('Choose one profile named ${rawBody.trim()}.');
      return true;
    }
    _showCommandMessage('Voice command not recognized: ${rawBody.trim()}.');
    return true;
  }

  Set<String> _contactCommandNames(NavivoxProfileContact contact) {
    return {
      _normalizeCommand(contact.profileId),
      _normalizeCommand(contact.displayName),
    };
  }

  String _normalizeCommand(String value) {
    return value
        .trim()
        .toLowerCase()
        .replaceAll(RegExp(r'[^a-z0-9]+'), ' ')
        .trim()
        .replaceAll(RegExp(r'\s+'), ' ');
  }

  void _showCommandMessage(String message) {
    setState(() => _voiceNotice = message);
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text(message)));
  }

  void _syncRouteProfile(NavivoxChannel channel) {
    final serverId = widget.serverId;
    final profileId = widget.profileId;
    if (serverId == null || profileId == null) return;

    final key = '$serverId::$profileId';
    if (_lastRouteProfileKey != key) {
      _lastRouteProfileKey = key;
      _routeProfileSynced = false;
    }
    if (_routeProfileSynced) return;
    if (channel.state.selectedProfileContactKey == key &&
        channel.state.activeServerId == serverId) {
      _routeProfileSynced = true;
      return;
    }
    final exists = channel.state.profileContacts.any(
      (contact) =>
          contact.serverId == serverId && contact.profileId == profileId,
    );
    if (!exists) {
      _routeProfileSynced = true;
      return;
    }

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (channel.state.selectedProfileContactKey == key &&
          channel.state.activeServerId == serverId) {
        _routeProfileSynced = true;
        return;
      }
      channel.selectProfileContact(serverId: serverId, profileId: profileId);
      _routeProfileSynced = true;
    });
  }
}

class _VoiceModeBanner extends StatelessWidget {
  const _VoiceModeBanner({
    required this.commandMode,
    required this.disabledReason,
    required this.notice,
    required this.pending,
    required this.ready,
    required this.canTrustServer,
    required this.onTrustServer,
    required this.onCancelPending,
  });

  final bool commandMode;
  final String? disabledReason;
  final String? notice;
  final bool pending;
  final bool ready;
  final bool canTrustServer;
  final VoidCallback? onTrustServer;
  final VoidCallback onCancelPending;

  @override
  Widget build(BuildContext context) {
    final text = pending
        ? 'Sending...'
        : commandMode
        ? 'Command mode'
        : disabledReason != null
        ? 'Voice disabled: $disabledReason'
        : ready
        ? 'Continuous voice ready'
        : notice;
    if (text == null) return const SizedBox.shrink();
    return Material(
      color: Theme.of(context).colorScheme.surfaceContainerHighest,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        child: Row(
          children: [
            Icon(
              disabledReason == null ? Icons.keyboard_voice : Icons.mic_off,
              size: 18,
            ),
            const SizedBox(width: 8),
            Expanded(child: Text(text)),
            if (pending)
              TextButton(
                onPressed: onCancelPending,
                child: const Text('Cancel'),
              ),
            if (!pending && canTrustServer)
              TextButton(
                onPressed: onTrustServer,
                child: const Text('Trust server'),
              ),
          ],
        ),
      ),
    );
  }
}
