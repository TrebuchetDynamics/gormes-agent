import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/channel/fake_navivox_channel.dart';
import '../../../router/app_routes.dart';

class SetupScreen extends ConsumerWidget {
  const SetupScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('Navivox')),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 520),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  'Set up Navivox',
                  style: Theme.of(context).textTheme.headlineMedium,
                ),
                const SizedBox(height: 12),
                const Text(
                  'Start with fake local protocol state while the stdio server '
                  'contract is being implemented.',
                ),
                const SizedBox(height: 24),
                FilledButton.icon(
                  onPressed: () {
                    ref.read(fakeNavivoxChannelProvider).enterFakeServerMode();
                    context.go(AppRoutes.chats);
                  },
                  icon: const Icon(Icons.offline_bolt),
                  label: const Text('Use fake local server'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
