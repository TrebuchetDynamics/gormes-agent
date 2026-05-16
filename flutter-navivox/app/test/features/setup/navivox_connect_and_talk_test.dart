import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/channel/gateway_navivox_channel.dart';
import 'package:navivox/core/channel/navivox_channel.dart';
import 'package:navivox/core/channel/navivox_channel_provider.dart';
import 'package:navivox/core/gateway/navivox_gateway_protocol.dart';
import 'package:navivox/core/protocol/navivox_event.dart';
import 'package:navivox/router/app_router.dart';

void main() {
  testWidgets('connect-info lands on chat and sends a text turn', (
    tester,
  ) async {
    final channel = _ConnectAndTalkChannel();
    addTearDown(channel.dispose);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [navivoxChannelProvider.overrideWithValue(channel)],
        child: const _RouterTestApp(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Connect to Gormes'), findsOneWidget);
    expect(find.text('Connect and talk'), findsOneWidget);
    expect(find.textContaining('gormes navivox connect-info'), findsWidgets);
    expect(_caseInsensitiveText('telephony'), findsNothing);
    expect(_caseInsensitiveText('fake'), findsNothing);

    await tester.enterText(
      find.widgetWithText(TextField, 'Gateway base URL'),
      'http://127.0.0.1:8765',
    );
    await tester.enterText(
      find.widgetWithText(TextField, 'Pairing token'),
      'nvbx_test_token',
    );
    await tester.tap(find.text('Connect and talk'));
    await tester.pumpAndSettle();

    expect(
      channel.connectedConfig?.baseUri.toString(),
      'http://127.0.0.1:8765',
    );
    expect(channel.connectedConfig?.token, 'nvbx_test_token');
    expect(find.text('Gormes Gateway'), findsOneWidget);
    expect(find.textContaining('Gateway online'), findsOneWidget);

    await tester.enterText(
      find.widgetWithText(TextField, 'Message Gormes'),
      'hello gateway',
    );
    await tester.tap(find.byIcon(Icons.send));
    await tester.pumpAndSettle();

    expect(channel.sentTexts, ['hello gateway']);
    expect(find.text('hello gateway'), findsOneWidget);
    expect(find.text('hello from gateway'), findsOneWidget);
    expect(_caseInsensitiveText('telephony'), findsNothing);
  });

  testWidgets(
    'connect failure gives connect-info guidance without token leak',
    (tester) async {
      final channel = _FailingConnectChannel();
      addTearDown(channel.dispose);

      await tester.pumpWidget(
        ProviderScope(
          overrides: [navivoxChannelProvider.overrideWithValue(channel)],
          child: const _RouterTestApp(),
        ),
      );
      await tester.pumpAndSettle();

      await tester.enterText(
        find.widgetWithText(TextField, 'Gateway base URL'),
        'http://127.0.0.1:8765',
      );
      await tester.enterText(
        find.widgetWithText(TextField, 'Pairing token'),
        'nvbx_secret_should_not_render',
      );
      await tester.tap(find.text('Connect and talk'));
      await tester.pumpAndSettle();

      expect(find.text('Could not connect to Gormes gateway.'), findsOneWidget);
      expect(find.textContaining('gormes navivox connect-info'), findsWidgets);
      expect(
        _caseInsensitiveText('nvbx_secret_should_not_render'),
        findsNothing,
      );
      expect(_caseInsensitiveText('telephony'), findsNothing);
    },
  );
}

Finder _caseInsensitiveText(String needle) {
  return find.byWidgetPredicate((widget) {
    if (widget is! Text) return false;
    final data = widget.data;
    if (data == null) return false;
    return data.toLowerCase().contains(needle.toLowerCase());
  });
}

class _RouterTestApp extends ConsumerWidget {
  const _RouterTestApp();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return MaterialApp.router(routerConfig: ref.watch(routerProvider));
  }
}

class _ConnectAndTalkChannel extends GatewayNavivoxChannel {
  NavivoxChannelState _state = const NavivoxChannelState();
  NavivoxGatewayConfig? connectedConfig;
  final List<String> sentTexts = [];

  @override
  NavivoxChannelState get state => _state;

  @override
  Future<void> connect(NavivoxGatewayConfig config) async {
    connectedConfig = config;
    const server = NavivoxServer(
      id: 'navivox-gateway',
      name: 'Gormes Gateway',
      status: 'Gateway online - 127.0.0.1:8765',
    );
    _state = _state.copyWith(servers: [server], activeServerId: server.id);
    notifyListeners();
  }

  @override
  void sendText(String text) {
    final trimmed = text.trim();
    if (trimmed.isEmpty) return;
    sentTexts.add(trimmed);
    final now = DateTime(2026, 5, 16, 9, 41);
    _state = _state.copyWith(
      messages: [
        ..._state.messages,
        NavivoxChatMessage(
          id: 'user-${sentTexts.length}',
          author: NavivoxMessageAuthor.user,
          kind: NavivoxMessageKind.text,
          createdAt: now,
          text: trimmed,
        ),
        NavivoxChatMessage(
          id: 'assistant-${sentTexts.length}',
          author: NavivoxMessageAuthor.assistant,
          kind: NavivoxMessageKind.text,
          createdAt: now,
          text: 'hello from gateway',
        ),
      ],
    );
    notifyListeners();
  }

  @override
  void sendVoice({
    required Uint8List audio,
    required String transcript,
    required Duration duration,
    required double confidence,
  }) {
    sendText(transcript);
  }
}

class _FailingConnectChannel extends GatewayNavivoxChannel {
  @override
  Future<void> connect(NavivoxGatewayConfig config) async {
    throw StateError('connection failed for ${config.token}');
  }
}
