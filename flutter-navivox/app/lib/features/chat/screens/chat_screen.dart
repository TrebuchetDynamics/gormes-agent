import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/channel/fake_navivox_channel.dart';
import '../widgets/simple_chat_adapter.dart';

class ChatScreen extends ConsumerWidget {
  const ChatScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final channel = ref.watch(fakeNavivoxChannelProvider);
    final state = channel.state;
    final server = state.activeServer;

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
      body: SimpleChatAdapter(
        messages: state.messages,
        onSend: channel.sendText,
      ),
    );
  }
}
