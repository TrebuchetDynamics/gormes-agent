import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../../../core/protocol/navivox_event.dart';
import '../../voice/services/voice_capture_service.dart';
import '../../voice/widgets/voice_morph_surface.dart';

class SimpleChatAdapter extends StatefulWidget {
  const SimpleChatAdapter({
    required this.messages,
    required this.onSend,
    this.voiceCaptureService,
    this.onVoice,
    this.voiceCaptureTimeout = const Duration(seconds: 30),
    super.key,
  });

  final List<NavivoxChatMessage> messages;
  final ValueChanged<String> onSend;

  /// When provided, a microphone button is rendered next to the send button.
  /// Tapping it starts a [VoiceCaptureService.capture] and the result is
  /// delivered via [onVoice].
  final VoiceCaptureService? voiceCaptureService;
  final ValueChanged<VoiceCapture>? onVoice;
  final Duration voiceCaptureTimeout;

  @override
  State<SimpleChatAdapter> createState() => _SimpleChatAdapterState();
}

class _SimpleChatAdapterState extends State<SimpleChatAdapter> {
  final _controller = TextEditingController();
  bool _capturing = false;
  String? _captureError;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Expanded(
          child: ListView.builder(
            padding: const EdgeInsets.all(16),
            itemCount: widget.messages.length,
            itemBuilder: (context, index) {
              return _MessageTile(message: widget.messages[index]);
            },
          ),
        ),
        if (_captureError != null)
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Text(_captureError!,
                style: const TextStyle(color: Colors.red)),
          ),
        SafeArea(
          top: false,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _controller,
                    decoration: const InputDecoration(
                      border: OutlineInputBorder(),
                      hintText: 'Message fake Navivox',
                    ),
                    onSubmitted: _send,
                  ),
                ),
                const SizedBox(width: 8),
                IconButton.filled(
                  tooltip: 'Send',
                  onPressed: () => _send(_controller.text),
                  icon: const Icon(Icons.send),
                ),
                if (widget.voiceCaptureService != null) ...[
                  const SizedBox(width: 8),
                  IconButton.filled(
                    tooltip: _capturing ? 'Stop voice capture' : 'Record voice',
                    onPressed: _toggleVoiceCapture,
                    icon: Icon(_capturing ? Icons.stop : Icons.mic),
                  ),
                ],
              ],
            ),
          ),
        ),
      ],
    );
  }

  void _send(String text) {
    widget.onSend(text);
    _controller.clear();
  }

  Future<void> _toggleVoiceCapture() async {
    final service = widget.voiceCaptureService;
    if (service == null) return;

    if (_capturing) {
      // Tapping while recording cancels — the timeout-based capture has no
      // public stop hook in v1, so we just visually de-activate. The capture
      // future still completes on its own and onVoice fires when it does.
      setState(() => _capturing = false);
      return;
    }

    setState(() {
      _capturing = true;
      _captureError = null;
    });
    try {
      final capture = await service.capture(timeout: widget.voiceCaptureTimeout);
      if (!mounted) return;
      widget.onVoice?.call(capture);
    } on VoiceCaptureTimeout {
      if (mounted) setState(() => _captureError = 'Voice capture timed out.');
    } catch (e) {
      if (mounted) setState(() => _captureError = 'Voice capture failed: $e');
    } finally {
      if (mounted) setState(() => _capturing = false);
    }
  }
}

class _MessageTile extends StatelessWidget {
  const _MessageTile({required this.message});

  final NavivoxChatMessage message;

  @override
  Widget build(BuildContext context) {
    final alignment = message.author == NavivoxMessageAuthor.user
        ? Alignment.centerRight
        : Alignment.centerLeft;

    return Align(
      alignment: alignment,
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 520),
        child: Card(
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  '${message.author.name} • '
                  '${DateFormat.Hm().format(message.createdAt)}',
                  style: Theme.of(context).textTheme.labelSmall,
                ),
                const SizedBox(height: 6),
                _MessageBody(message: message),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _MessageBody extends StatelessWidget {
  const _MessageBody({required this.message});

  final NavivoxChatMessage message;

  @override
  Widget build(BuildContext context) {
    return switch (message.kind) {
      NavivoxMessageKind.text => Text(message.text ?? ''),
      NavivoxMessageKind.toolCall => _ToolCallBody(toolCall: message.toolCall!),
      NavivoxMessageKind.voice => _VoiceBody(voice: message.voice!),
    };
  }
}

class _ToolCallBody extends StatelessWidget {
  const _ToolCallBody({required this.toolCall});

  final NavivoxToolCall toolCall;

  @override
  Widget build(BuildContext context) {
    return ListTile(
      contentPadding: EdgeInsets.zero,
      leading: const Icon(Icons.build_circle),
      title: const Text('Tool call'),
      subtitle: Text(
        '${toolCall.name} • ${toolCall.status}\n${toolCall.summary}',
      ),
    );
  }
}

class _VoiceBody extends StatelessWidget {
  const _VoiceBody({required this.voice});

  final NavivoxVoiceMessage voice;

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        VoiceMorphSurface(
          state: VoiceMorphState.speaking,
          intensity: voice.confidence,
          size: 72,
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 6),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text('Voice message'),
                const SizedBox(height: 4),
                Text(
                  '${voice.duration.inSeconds}s • '
                  '${(voice.confidence * 100).round()}% confidence\n'
                  '${voice.transcript}',
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }
}
