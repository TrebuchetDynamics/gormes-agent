import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/channel/navivox_channel.dart';
import 'package:navivox/core/channel/navivox_channel_provider.dart';
import 'package:navivox/features/chat/screens/chat_screen.dart';
import 'package:navivox/router/app_router.dart';

import '../../support/test_navivox_channel.dart';

const _servers = [
  NavivoxServer(id: 'local', name: 'Local Gormes', status: 'online'),
  NavivoxServer(id: 'office', name: 'Office', status: 'offline'),
];

final _contacts = [
  NavivoxProfileContact(
    serverId: 'local',
    profileId: 'mineru',
    displayName: 'Mineru Builder',
    serverLabel: 'local',
    health: NavivoxProfileHealth.online,
    latestPreview: 'Ready to work on mineru',
    latestAt: DateTime(2026, 5, 16, 9, 41),
    workspaceRootCount: 2,
    micAvailable: true,
  ),
  NavivoxProfileContact(
    serverId: 'office',
    profileId: 'support',
    displayName: 'Support Triage',
    serverLabel: 'office',
    health: NavivoxProfileHealth.needsAuth,
    latestPreview: 'Waiting for token',
    latestAt: DateTime(2026, 5, 16, 9, 22),
    workspaceRootCount: 1,
    attentionBadges: ['auth'],
    micAvailable: false,
  ),
  NavivoxProfileContact(
    serverId: 'local',
    profileId: 'personal',
    displayName: 'Personal',
    serverLabel: 'local',
    health: NavivoxProfileHealth.offline,
    latestPreview: 'Gateway unavailable',
    latestAt: DateTime(2026, 5, 15, 18),
    workspaceRootCount: 0,
    attentionBadges: ['offline'],
    micAvailable: false,
  ),
];

void main() {
  testWidgets('renders profiles as a flat multi-server contact list', (
    tester,
  ) async {
    final channel = TestNavivoxChannel()
      ..seedServers(_servers, activeServerId: 'local')
      ..seedProfileContacts(_contacts);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [navivoxChannelProvider.overrideWithValue(channel)],
        child: const _RouterTestApp(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Navivox'), findsOneWidget);
    expect(find.text('Mineru Builder'), findsOneWidget);
    expect(find.text('Support Triage'), findsOneWidget);
    expect(find.text('Personal'), findsOneWidget);
    expect(
      find.byKey(const ValueKey('profile-contact-local-mineru')),
      findsOneWidget,
    );
    expect(find.text('local'), findsWidgets);
    expect(find.text('office'), findsOneWidget);
    expect(find.text('2 roots'), findsOneWidget);
    expect(find.text('auth'), findsWidgets);
    expect(find.byTooltip('Add profile'), findsOneWidget);
  });

  testWidgets('selecting a profile opens scoped chat and sends in that scope', (
    tester,
  ) async {
    final channel = TestNavivoxChannel()
      ..seedServers(_servers, activeServerId: 'local')
      ..seedProfileContacts(_contacts);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [navivoxChannelProvider.overrideWithValue(channel)],
        child: const _RouterTestApp(),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Support Triage'));
    await tester.pumpAndSettle();

    expect(channel.selectedProfileScope, (
      serverId: 'office',
      profileId: 'support',
    ));
    expect(find.text('Support Triage'), findsOneWidget);
    expect(find.byKey(const ValueKey('chat-active-profile')), findsOneWidget);
    expect(find.text('office'), findsOneWidget);

    await tester.enterText(
      find.widgetWithText(TextField, 'Message Gormes'),
      'triage latest ticket',
    );
    await tester.tap(find.byIcon(Icons.send));
    await tester.pumpAndSettle();

    expect(channel.sentTexts, ['triage latest ticket']);
    expect(channel.sentTextCalls.last, (
      text: 'triage latest ticket',
      serverId: 'office',
      profileId: 'support',
    ));
  });

  testWidgets('long pressing a profile opens edit/details placeholder', (
    tester,
  ) async {
    final channel = TestNavivoxChannel()
      ..seedServers(_servers, activeServerId: 'local')
      ..seedProfileContacts(_contacts);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [navivoxChannelProvider.overrideWithValue(channel)],
        child: const _RouterTestApp(),
      ),
    );
    await tester.pumpAndSettle();

    await tester.longPress(find.text('Mineru Builder'));
    await tester.pumpAndSettle();

    expect(find.text('Profile details'), findsOneWidget);
    expect(find.text('Mineru Builder\nmineru'), findsOneWidget);
    expect(find.text('Edit profile'), findsOneWidget);
  });

  testWidgets('deep-linked chat route selects the profile scope', (
    tester,
  ) async {
    final channel = TestNavivoxChannel()
      ..seedServers(_servers, activeServerId: 'local')
      ..seedProfileContacts(_contacts);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [navivoxChannelProvider.overrideWithValue(channel)],
        child: const MaterialApp(
          home: ChatScreen(serverId: 'office', profileId: 'support'),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(channel.selectedProfileScope, (
      serverId: 'office',
      profileId: 'support',
    ));
    expect(find.text('Support Triage'), findsOneWidget);
    expect(find.byKey(const ValueKey('chat-active-profile')), findsOneWidget);
  });
}

class _RouterTestApp extends ConsumerWidget {
  const _RouterTestApp();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return MaterialApp.router(routerConfig: ref.watch(routerProvider));
  }
}
