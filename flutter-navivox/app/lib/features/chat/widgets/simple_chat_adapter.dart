import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../../../core/protocol/navivox_event.dart';

class SimpleChatAdapter extends StatefulWidget {
  const SimpleChatAdapter({
    required this.messages,
    required this.onSend,
    super.key,
  });

  final List<NavivoxChatMessage> messages;
  final ValueChanged<String> onSend;

  @override
  State<SimpleChatAdapter> createState() => _SimpleChatAdapterState();
}

class _SimpleChatAdapterState extends State<SimpleChatAdapter> {
  final _controller = TextEditingController();

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
    return ListTile(
      contentPadding: EdgeInsets.zero,
      leading: const Icon(Icons.play_circle),
      title: const Text('Voice message'),
      subtitle: Text(
        '${voice.duration.inSeconds}s • '
        '${(voice.confidence * 100).round()}% confidence\n'
        '${voice.transcript}',
      ),
    );
  }
}
