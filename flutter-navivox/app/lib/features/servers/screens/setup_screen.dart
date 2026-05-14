import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/channel/navivox_channel_provider.dart';
import '../../../core/gateway/navibox_gateway_protocol.dart';
import '../../../router/app_routes.dart';
import '../../keys/providers/key_store_providers.dart';
import '../../keys/services/key_store.dart';

class SetupScreen extends ConsumerStatefulWidget {
  const SetupScreen({super.key});

  @override
  ConsumerState<SetupScreen> createState() => _SetupScreenState();
}

class _SetupScreenState extends ConsumerState<SetupScreen> {
  final _baseUrlController = TextEditingController(
    text: 'http://127.0.0.1:8765',
  );
  final _tokenController = TextEditingController();
  bool _connecting = false;
  String? _error;

  @override
  void dispose() {
    _baseUrlController.dispose();
    _tokenController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final serverStore = ref.watch(serverStoreProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('Navivox')),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 520),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: SingleChildScrollView(
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
                    'Connect to the native Gormes gateway over HTTP/WebSocket, '
                    'or use fake local protocol state for UI development.',
                  ),
                  const SizedBox(height: 24),
                  TextField(
                    controller: _baseUrlController,
                    decoration: const InputDecoration(
                      border: OutlineInputBorder(),
                      labelText: 'Gateway base URL',
                    ),
                    keyboardType: TextInputType.url,
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _tokenController,
                    decoration: const InputDecoration(
                      border: OutlineInputBorder(),
                      labelText: 'Pairing token',
                    ),
                    obscureText: true,
                  ),
                  const SizedBox(height: 12),
                  FilledButton.icon(
                    onPressed: _connecting ? null : _connectGateway,
                    icon: _connecting
                        ? const SizedBox.square(
                            dimension: 18,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Icon(Icons.hub),
                    label: const Text('Connect to Gormes gateway'),
                  ),
                  if (_error != null) ...[
                    const SizedBox(height: 8),
                    Text(_error!, style: const TextStyle(color: Colors.red)),
                  ],
                  const SizedBox(height: 24),
                  FilledButton.icon(
                    onPressed: () {
                      ref
                          .read(activeNavivoxChannelProvider)
                          .enterFakeServerMode();
                      context.go(AppRoutes.chats);
                    },
                    icon: const Icon(Icons.offline_bolt),
                    label: const Text('Use fake local server'),
                  ),
                  StreamBuilder<List<StoredServer>>(
                    stream: serverStore.watch(),
                    builder: (context, snapshot) {
                      final servers = snapshot.data ?? const <StoredServer>[];
                      if (servers.isEmpty) return const SizedBox.shrink();
                      return Padding(
                        padding: const EdgeInsets.only(top: 24),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              'Recently imported',
                              style: Theme.of(context).textTheme.titleSmall,
                            ),
                            const SizedBox(height: 8),
                            for (final s in servers)
                              ListTile(
                                contentPadding: EdgeInsets.zero,
                                leading: const Icon(Icons.dns),
                                title: Text(s.label),
                                subtitle: Text(
                                  '${s.username}@${s.hostname}:${s.port}',
                                ),
                              ),
                          ],
                        ),
                      );
                    },
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _connectGateway() async {
    setState(() {
      _connecting = true;
      _error = null;
    });
    try {
      final config = NaviboxGatewayConfig.fromBaseUrl(
        _baseUrlController.text.trim(),
        token: _tokenController.text.trim(),
      );
      await ref.read(gatewayNaviboxChannelProvider).connect(config);
      ref.read(activeNavivoxChannelProvider).useGateway();
      if (mounted) context.go(AppRoutes.chats);
    } catch (_) {
      if (mounted) {
        setState(() => _error = 'Could not connect to Gormes gateway.');
      }
    } finally {
      if (mounted) setState(() => _connecting = false);
    }
  }
}
