import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:intl/intl.dart';

import '../../../core/channel/navivox_channel.dart';
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
    this.forwardTargets = const [],
    this.onForward,
    super.key,
  });

  final List<NavivoxChatMessage> messages;
  final ValueChanged<String> onSend;
  final VoiceCaptureService? voiceCaptureService;
  final ValueChanged<VoiceCapture>? onVoice;
  final Duration voiceCaptureTimeout;
  final List<NavivoxProfileContact> forwardTargets;
  final void Function(NavivoxChatMessage message, NavivoxProfileContact target)?
  onForward;

  @override
  State<SimpleChatAdapter> createState() => _SimpleChatAdapterState();
}

class _SimpleChatAdapterState extends State<SimpleChatAdapter> {
  final _controller = TextEditingController();
  final _scrollController = ScrollController();
  bool _capturing = false;
  String? _captureError;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _scrollToEnd());
  }

  @override
  void didUpdateWidget(SimpleChatAdapter oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.messages.length > oldWidget.messages.length) {
      WidgetsBinding.instance.addPostFrameCallback((_) => _scrollToEnd());
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  void _scrollToEnd() {
    if (_scrollController.hasClients) {
      _scrollController.animateTo(
        _scrollController.position.maxScrollExtent,
        duration: const Duration(milliseconds: 200),
        curve: Curves.easeOut,
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      children: [
        Expanded(
          child: widget.messages.isEmpty
              ? Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(
                        Icons.chat_bubble_outline,
                        size: 64,
                        color: theme.colorScheme.outline,
                      ),
                      const SizedBox(height: 12),
                      Text(
                        'Start a conversation',
                        style: theme.textTheme.titleMedium?.copyWith(
                          color: theme.colorScheme.outline,
                        ),
                      ),
                    ],
                  ),
                )
              : ListView.builder(
                  controller: _scrollController,
                  padding: const EdgeInsets.symmetric(
                    horizontal: 12,
                    vertical: 8,
                  ),
                  itemCount: widget.messages.length,
                  itemBuilder: (context, index) {
                    final msg = widget.messages[index];
                    final isUser = msg.author == NavivoxMessageAuthor.user;
                    final prev = index > 0 ? widget.messages[index - 1] : null;
                    final showTail = prev == null || prev.author != msg.author;
                    return _TelegramBubble(
                      message: msg,
                      isUser: isUser,
                      showTail: showTail,
                      forwardTargets: widget.forwardTargets,
                      onForward: widget.onForward,
                    );
                  },
                ),
        ),
        if (_captureError != null)
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
            child: Text(
              _captureError!,
              style: TextStyle(color: theme.colorScheme.error, fontSize: 12),
            ),
          ),
        SafeArea(
          top: false,
          child: _InputBar(
            controller: _controller,
            onSend: _send,
            voiceService: widget.voiceCaptureService,
            capturing: _capturing,
            onToggleVoice: _toggleVoiceCapture,
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
      setState(() => _capturing = false);
      return;
    }

    setState(() {
      _capturing = true;
      _captureError = null;
    });
    try {
      final capture = await service.capture(
        timeout: widget.voiceCaptureTimeout,
      );
      if (!mounted) return;
      widget.onVoice?.call(capture);
    } on VoiceCaptureTimeout {
      if (mounted) {
        setState(() => _captureError = 'Voice capture timed out.');
      }
    } catch (e) {
      if (mounted) {
        setState(() => _captureError = 'Voice capture failed: $e');
      }
    } finally {
      if (mounted) setState(() => _capturing = false);
    }
  }
}

class _TelegramBubble extends StatelessWidget {
  const _TelegramBubble({
    required this.message,
    required this.isUser,
    required this.showTail,
    required this.forwardTargets,
    required this.onForward,
  });

  final NavivoxChatMessage message;
  final bool isUser;
  final bool showTail;
  final List<NavivoxProfileContact> forwardTargets;
  final void Function(NavivoxChatMessage message, NavivoxProfileContact target)?
  onForward;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final bubbleColor = isUser
        ? theme.colorScheme.primaryContainer
        : theme.colorScheme.surfaceContainerHigh;
    final textColor = isUser
        ? theme.colorScheme.onPrimaryContainer
        : theme.colorScheme.onSurface;
    final timeColor = isUser
        ? theme.colorScheme.onPrimaryContainer.withValues(alpha: 0.6)
        : theme.colorScheme.onSurfaceVariant;

    return GestureDetector(
      behavior: HitTestBehavior.translucent,
      onLongPress: () => _showMessageActions(context),
      child: Padding(
        padding: const EdgeInsets.only(bottom: 2),
        child: LayoutBuilder(
          builder: (context, constraints) {
            final tailWidth = showTail ? 12.0 : 0.0;
            final maxBubbleWidth = (constraints.maxWidth - tailWidth) * 0.78;
            return Row(
              mainAxisAlignment: isUser
                  ? MainAxisAlignment.end
                  : MainAxisAlignment.start,
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                if (!isUser && showTail)
                  Padding(
                    padding: const EdgeInsets.only(right: 4),
                    child: CustomPaint(
                      size: const Size(8, 12),
                      painter: _BubbleTailPainter(
                        color: bubbleColor,
                        flip: false,
                      ),
                    ),
                  ),
                ConstrainedBox(
                  constraints: BoxConstraints(maxWidth: maxBubbleWidth),
                  child: Container(
                    padding: const EdgeInsets.fromLTRB(10, 6, 10, 4),
                    decoration: BoxDecoration(
                      color: bubbleColor,
                      borderRadius: BorderRadius.only(
                        topLeft: const Radius.circular(12),
                        topRight: const Radius.circular(12),
                        bottomLeft: Radius.circular(
                          isUser ? 12 : (showTail ? 4 : 12),
                        ),
                        bottomRight: Radius.circular(
                          isUser ? (showTail ? 4 : 12) : 12,
                        ),
                      ),
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        _MessageBody(message: message, textColor: textColor),
                        Align(
                          alignment: Alignment.bottomRight,
                          child: Padding(
                            padding: const EdgeInsets.only(left: 8, top: 2),
                            child: Text(
                              DateFormat.Hm().format(message.createdAt),
                              style: TextStyle(color: timeColor, fontSize: 11),
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
                if (isUser && showTail)
                  Padding(
                    padding: const EdgeInsets.only(left: 4),
                    child: CustomPaint(
                      size: const Size(8, 12),
                      painter: _BubbleTailPainter(
                        color: bubbleColor,
                        flip: true,
                      ),
                    ),
                  ),
              ],
            );
          },
        ),
      ),
    );
  }

  Future<void> _showMessageActions(BuildContext context) {
    final text = _messageActionText(message);
    return showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder: (sheetContext) => SafeArea(
        child: ListView(
          shrinkWrap: true,
          padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
          children: [
            Text(
              'Message actions',
              style: Theme.of(sheetContext).textTheme.titleLarge,
            ),
            const SizedBox(height: 12),
            if (text.isNotEmpty)
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: Theme.of(
                    sheetContext,
                  ).colorScheme.surfaceContainerHigh,
                  borderRadius: BorderRadius.circular(16),
                ),
                child: SelectableText(text),
              ),
            if (text.isNotEmpty)
              ListTile(
                leading: const Icon(Icons.copy),
                title: const Text('Copy text'),
                onTap: () async {
                  await Clipboard.setData(ClipboardData(text: text));
                  if (!sheetContext.mounted) return;
                  Navigator.of(sheetContext).pop();
                  ScaffoldMessenger.maybeOf(context)?.showSnackBar(
                    const SnackBar(content: Text('Message copied')),
                  );
                },
              ),
            if (forwardTargets.isNotEmpty && onForward != null) ...[
              const Divider(),
              const ListTile(
                leading: Icon(Icons.forward),
                title: Text('Forward to'),
              ),
              for (final target in forwardTargets)
                ListTile(
                  leading: const CircleAvatar(child: Icon(Icons.person)),
                  title: Text(target.displayName),
                  subtitle: Text(target.serverLabel),
                  onTap: () {
                    Navigator.of(sheetContext).pop();
                    onForward?.call(message, target);
                  },
                ),
            ],
          ],
        ),
      ),
    );
  }
}

String _messageActionText(NavivoxChatMessage message) {
  return switch (message.kind) {
    NavivoxMessageKind.text => message.text ?? '',
    NavivoxMessageKind.voice => message.voice?.transcript ?? '',
    NavivoxMessageKind.toolCall => [
      message.toolCall?.name,
      message.toolCall?.status,
      message.toolCall?.summary,
    ].whereType<String>().where((part) => part.isNotEmpty).join('\n'),
  };
}

class _BubbleTailPainter extends CustomPainter {
  const _BubbleTailPainter({required this.color, required this.flip});

  final Color color;
  final bool flip;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()..color = color;
    final path = Path();
    if (flip) {
      path.moveTo(size.width, 0);
      path.lineTo(0, size.height);
      path.lineTo(size.width, size.height * 0.6);
    } else {
      path.moveTo(0, 0);
      path.lineTo(size.width, size.height);
      path.lineTo(0, size.height * 0.6);
    }
    path.close();
    canvas.drawPath(path, paint);
  }

  @override
  bool shouldRepaint(_BubbleTailPainter oldDelegate) =>
      color != oldDelegate.color || flip != oldDelegate.flip;
}

class _MessageBody extends StatelessWidget {
  const _MessageBody({required this.message, this.textColor});

  final NavivoxChatMessage message;
  final Color? textColor;

  @override
  Widget build(BuildContext context) {
    return switch (message.kind) {
      NavivoxMessageKind.text => Text(
        message.text ?? '',
        style: TextStyle(color: textColor, fontSize: 15),
      ),
      NavivoxMessageKind.toolCall => _ToolCallBody(
        toolCall: message.toolCall!,
        textColor: textColor,
      ),
      NavivoxMessageKind.voice => _VoiceBody(
        voice: message.voice!,
        textColor: textColor,
      ),
    };
  }
}

class _ToolCallBody extends StatelessWidget {
  const _ToolCallBody({required this.toolCall, this.textColor});

  final NavivoxToolCall toolCall;
  final Color? textColor;

  @override
  Widget build(BuildContext context) {
    final statusColor = switch (toolCall.status) {
      'started' => Colors.orange,
      'finished' => Colors.green,
      'failed' => Colors.red,
      _ => Colors.grey,
    };
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Row(
          children: [
            Icon(
              Icons.build_circle,
              size: 16,
              color: textColor?.withValues(alpha: 0.7),
            ),
            const SizedBox(width: 6),
            Text(
              toolCall.name,
              style: TextStyle(
                color: textColor,
                fontWeight: FontWeight.w500,
                fontSize: 13,
              ),
            ),
            const Spacer(),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
              decoration: BoxDecoration(
                color: statusColor.withValues(alpha: 0.15),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Text(
                toolCall.status,
                style: TextStyle(color: statusColor, fontSize: 11),
              ),
            ),
          ],
        ),
        if (toolCall.summary.isNotEmpty) ...[
          const SizedBox(height: 4),
          Text(
            toolCall.summary,
            style: TextStyle(
              color: textColor?.withValues(alpha: 0.8),
              fontSize: 13,
            ),
          ),
        ],
        for (final artifact in toolCall.artifacts) ...[
          const SizedBox(height: 4),
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Icon(
                Icons.attachment,
                size: 14,
                color: textColor?.withValues(alpha: 0.6),
              ),
              const SizedBox(width: 4),
              Flexible(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Wrap(
                      spacing: 6,
                      crossAxisAlignment: WrapCrossAlignment.center,
                      children: [
                        Text(
                          artifact.title,
                          style: TextStyle(
                            color: textColor?.withValues(alpha: 0.7),
                            fontSize: 12,
                          ),
                        ),
                        Text(
                          artifact.kind,
                          style: TextStyle(
                            color: textColor?.withValues(alpha: 0.55),
                            fontSize: 11,
                          ),
                        ),
                      ],
                    ),
                    if (artifact.summary != null &&
                        artifact.summary!.isNotEmpty)
                      Text(
                        artifact.summary!,
                        style: TextStyle(
                          color: textColor?.withValues(alpha: 0.65),
                          fontSize: 12,
                        ),
                      ),
                  ],
                ),
              ),
            ],
          ),
        ],
      ],
    );
  }
}

class _VoiceBody extends StatelessWidget {
  const _VoiceBody({required this.voice, this.textColor});

  final NavivoxVoiceMessage voice;
  final Color? textColor;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        VoiceMorphSurface(
          state: VoiceMorphState.speaking,
          intensity: voice.confidence,
          size: 40,
        ),
        const SizedBox(width: 10),
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              'Voice message',
              style: TextStyle(
                color: textColor,
                fontWeight: FontWeight.w500,
                fontSize: 13,
              ),
            ),
            Text(
              '${voice.duration.inSeconds}s',
              style: TextStyle(
                color: textColor?.withValues(alpha: 0.6),
                fontSize: 11,
              ),
            ),
            if (voice.transcript.isNotEmpty)
              Text(
                voice.transcript,
                style: TextStyle(
                  color: textColor?.withValues(alpha: 0.7),
                  fontSize: 12,
                ),
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
          ],
        ),
      ],
    );
  }
}

class _InputBar extends StatelessWidget {
  const _InputBar({
    required this.controller,
    required this.onSend,
    this.voiceService,
    this.capturing = false,
    this.onToggleVoice,
  });

  final TextEditingController controller;
  final ValueChanged<String> onSend;
  final VoiceCaptureService? voiceService;
  final bool capturing;
  final VoidCallback? onToggleVoice;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.fromLTRB(8, 8, 8, 8),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border(
          top: BorderSide(color: theme.colorScheme.outlineVariant, width: 0.5),
        ),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.end,
        children: [
          Expanded(
            child: TextField(
              controller: controller,
              maxLines: null,
              textInputAction: TextInputAction.send,
              decoration: InputDecoration(
                hintText: 'Message Gormes',
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(24),
                  borderSide: BorderSide.none,
                ),
                filled: true,
                fillColor: theme.colorScheme.surfaceContainerHighest,
                contentPadding: const EdgeInsets.symmetric(
                  horizontal: 16,
                  vertical: 8,
                ),
              ),
              onSubmitted: onSend,
            ),
          ),
          const SizedBox(width: 6),
          if (voiceService != null)
            IconButton.filledTonal(
              onPressed: onToggleVoice,
              icon: Icon(capturing ? Icons.stop : Icons.mic),
              style: IconButton.styleFrom(
                backgroundColor: capturing
                    ? theme.colorScheme.errorContainer
                    : null,
              ),
            ),
          const SizedBox(width: 4),
          IconButton.filled(
            onPressed: () => onSend(controller.text),
            icon: const Icon(Icons.send, size: 20),
          ),
        ],
      ),
    );
  }
}
