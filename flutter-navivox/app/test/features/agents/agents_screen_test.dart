import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/channel/fake_navivox_channel.dart';
import 'package:navivox/features/agents/screens/agents_screen.dart';

void main() {
  testWidgets(
      'shows empty-state and Refresh button when no agents are known yet',
      (tester) async {
    final channel = FakeNavivoxChannel();

    await tester.pumpWidget(ProviderScope(
      overrides: [fakeNavivoxChannelProvider.overrideWithValue(channel)],
      child: const MaterialApp(home: AgentsScreen()),
    ));

    expect(find.text('No agents loaded'), findsOneWidget);
    expect(find.text('Refresh'), findsOneWidget);

    await tester.tap(find.text('Refresh'));
    await tester.pump();

    // FakeNavivoxChannel.requestAgentList() seeds two agents.
    expect(channel.state.agents, hasLength(2));
    expect(find.text('Default'), findsOneWidget);
    expect(find.text('Architect'), findsOneWidget);
  });

  testWidgets('tapping an agent tile selects it through the channel',
      (tester) async {
    final channel = FakeNavivoxChannel()..requestAgentList();

    await tester.pumpWidget(ProviderScope(
      overrides: [fakeNavivoxChannelProvider.overrideWithValue(channel)],
      child: const MaterialApp(home: AgentsScreen()),
    ));

    expect(find.text('Default'), findsOneWidget);
    // Nothing selected yet, so no check icons should be rendered.
    expect(find.byIcon(Icons.check), findsNothing);

    await tester.tap(find.text('Architect'));
    await tester.pump();

    expect(channel.state.selectedAgentId, 'arch');
    expect(find.byIcon(Icons.check), findsOneWidget);
  });
}
