import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/channel/fake_navivox_channel.dart';
import 'package:navivox/core/channel/navivox_channel.dart';
import 'package:navivox/core/protocol/navivox_event.dart';
import 'package:navivox/features/chat/screens/chat_screen.dart';
import 'package:navivox/features/voice/services/voice_capture_service.dart';

void main() {
  testWidgets(
      'tapping mic on chat screen sends a voice frame through the channel',
      (tester) async {
    final channel = FakeNavivoxChannel()..enterFakeServerMode();
    final beforeCount = channel.state.messages.length;

    final voiceService = FakeVoiceCaptureService(
      audio: Uint8List.fromList(List<int>.generate(64, (i) => i)),
      transcript: 'wire-test',
      duration: const Duration(milliseconds: 600),
      confidence: 0.81,
    );

    await tester.pumpWidget(ProviderScope(
      overrides: [
        fakeNavivoxChannelProvider.overrideWithValue(channel),
        chatVoiceCaptureServiceProvider.overrideWithValue(voiceService),
      ],
      child: const MaterialApp(home: ChatScreen()),
    ));

    expect(find.byIcon(Icons.mic), findsOneWidget);
    await tester.tap(find.byIcon(Icons.mic));
    // VoiceMorphSurface uses an infinite Ticker so pumpAndSettle never
    // settles. Pump explicitly to let the FakeVoiceCaptureService Timer fire,
    // microtasks drain, and the state rebuilds run.
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));
    await tester.pump(const Duration(milliseconds: 50));

    final newMessages =
        channel.state.messages.skip(beforeCount).toList(growable: false);
    final userVoice = newMessages.firstWhere(
      (m) => m.author == NavivoxMessageAuthor.user &&
          m.kind == NavivoxMessageKind.voice,
    );
    expect(userVoice.voice!.transcript, 'wire-test');
    expect(userVoice.voice!.duration, const Duration(milliseconds: 600));
    expect(channel.lastSentFrame, isNotNull);
    expect(channel.lastSentFrame!.type, 'voice.submit');
  });

  testWidgets('approval banner appears on the chat screen when channel emits a request',
      (tester) async {
    final channel = FakeNavivoxChannel()..enterFakeServerMode();

    await tester.pumpWidget(ProviderScope(
      overrides: [
        fakeNavivoxChannelProvider.overrideWithValue(channel),
      ],
      child: const MaterialApp(home: ChatScreen()),
    ));

    expect(find.text('Approval requested'), findsNothing);

    channel.emitApprovalRequest(const NavivoxApprovalRequest(
      id: 'ap-9',
      toolCallId: 'tc-9',
      prompt: 'Run shell.run with elevated privileges?',
      risk: 'high',
    ));
    // Broadcast-stream listener fires on the microtask queue, then the banner
    // setState schedules a frame. pumpAndSettle hangs because the seeded voice
    // message renders VoiceMorphSurface (infinite Ticker) — pump explicitly.
    await tester.pump();
    await tester.pump();

    expect(find.text('Approval requested'), findsOneWidget);
    expect(find.textContaining('shell.run'), findsOneWidget);

    await tester.tap(find.text('Allow'));
    await tester.pump();

    expect(channel.approvalResponses.single.approvalId, 'ap-9');
    expect(channel.approvalResponses.single.approved, isTrue);
    expect(find.text('Approval requested'), findsNothing);
  });
}
