import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/channel/fake_navivox_channel.dart';
import '../../voice/services/voice_capture_service.dart';
import '../widgets/approval_banner.dart';
import '../widgets/simple_chat_adapter.dart';

/// Voice-capture service used by the chat input bar. Override in tests with
/// [FakeVoiceCaptureService]; production wiring slots in
/// [RecordVoiceCaptureService] once the real mic + STT plugins land.
final chatVoiceCaptureServiceProvider = Provider<VoiceCaptureService?>(
  (_) => null,
);

class ChatScreen extends ConsumerWidget {
  const ChatScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final channel = ref.watch(fakeNavivoxChannelProvider);
    final state = channel.state;
    final server = state.activeServer;
    final voiceService = ref.watch(chatVoiceCaptureServiceProvider);

    return Scaffold(
      appBar: AppBar(
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(server?.name ?? 'Chats'),
            if (server != null)
              Text(
                server.status,
                style: Theme.of(context).textTheme.labelMedium,
              ),
          ],
        ),
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
}
