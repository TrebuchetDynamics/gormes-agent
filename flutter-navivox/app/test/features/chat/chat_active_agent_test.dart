import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/channel/navivox_channel.dart';
import 'package:navivox/core/channel/navivox_channel_provider.dart';
import 'package:navivox/features/chat/screens/chat_screen.dart';

import '../../support/test_navivox_channel.dart';

const _seedAgents = [
  NavivoxAgent(id: 'def', name: 'Default', status: 'ready'),
  NavivoxAgent(id: 'arch', name: 'Architect', status: 'ready'),
];

const _seedServers = [
  NavivoxServer(id: 'srv1', name: 'Local', status: 'ready'),
];

void main() {
  testWidgets(
      'chat AppBar omits the active-agent indicator when no agent is selected',
      (tester) async {
    final channel = TestNavivoxChannel()
      ..seedServers(_seedServers, activeServerId: 'srv1')
      ..seedAgents(_seedAgents);

    await tester.pumpWidget(ProviderScope(
      overrides: [navivoxChannelProvider.overrideWithValue(channel)],
      child: const MaterialApp(home: ChatScreen()),
    ));

    expect(find.byKey(const ValueKey('chat-active-agent')), findsNothing);
  });

  testWidgets('chat AppBar shows the selected agent name as an indicator',
      (tester) async {
    final channel = TestNavivoxChannel()
      ..seedServers(_seedServers, activeServerId: 'srv1')
      ..seedAgents(_seedAgents, selectedAgentId: 'arch');

    await tester.pumpWidget(ProviderScope(
      overrides: [navivoxChannelProvider.overrideWithValue(channel)],
      child: const MaterialApp(home: ChatScreen()),
    ));

    final indicator = find.byKey(const ValueKey('chat-active-agent'));
    expect(indicator, findsOneWidget);
    expect(
      find.descendant(of: indicator, matching: find.text('Architect')),
      findsOneWidget,
    );
  });
}
