import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/protocol/navivox_event.dart';
import 'package:navivox/features/chat/widgets/simple_chat_adapter.dart';
import 'package:navivox/features/voice/services/voice_capture_service.dart';

void main() {
  testWidgets('mic button is hidden when no voice service is provided',
      (tester) async {
    await tester.pumpWidget(MaterialApp(
      home: Scaffold(
        body: SimpleChatAdapter(
          messages: const <NavivoxChatMessage>[],
          onSend: (_) {},
        ),
      ),
    ));

    expect(find.byIcon(Icons.mic), findsNothing);
    expect(find.byIcon(Icons.send), findsOneWidget);
  });

  testWidgets('tap mic invokes the voice service and forwards result via onVoice',
      (tester) async {
    final service = FakeVoiceCaptureService(
      audio: Uint8List.fromList([10, 20, 30, 40]),
      transcript: 'hello voice',
      duration: const Duration(milliseconds: 700),
      confidence: 0.88,
    );
    VoiceCapture? captured;

    await tester.pumpWidget(MaterialApp(
      home: Scaffold(
        body: SimpleChatAdapter(
          messages: const <NavivoxChatMessage>[],
          onSend: (_) {},
          voiceCaptureService: service,
          onVoice: (capture) => captured = capture,
        ),
      ),
    ));

    expect(find.byIcon(Icons.mic), findsOneWidget);

    await tester.tap(find.byIcon(Icons.mic));
    await tester.pumpAndSettle();

    expect(captured, isNotNull);
    expect(captured!.transcript, 'hello voice');
    expect(captured!.audio, [10, 20, 30, 40]);
  });

  testWidgets('mic shows recording indicator while capture is in flight',
      (tester) async {
    final service = FakeVoiceCaptureService(
      audio: Uint8List.fromList([1]),
      transcript: 't',
      duration: const Duration(milliseconds: 50),
      confidence: 1.0,
      captureLatency: const Duration(milliseconds: 100),
    );

    await tester.pumpWidget(MaterialApp(
      home: Scaffold(
        body: SimpleChatAdapter(
          messages: const <NavivoxChatMessage>[],
          onSend: (_) {},
          voiceCaptureService: service,
          onVoice: (_) {},
        ),
      ),
    ));

    await tester.tap(find.byIcon(Icons.mic));
    await tester.pump(); // start frame
    await tester.pump(const Duration(milliseconds: 30));

    // While recording the mic icon flips to a stop icon and the surface label
    // announces the active state.
    expect(find.byIcon(Icons.stop), findsOneWidget);
    expect(find.byIcon(Icons.mic), findsNothing);

    await tester.pumpAndSettle();

    // After the capture resolves the mic icon is back.
    expect(find.byIcon(Icons.mic), findsOneWidget);
    expect(find.byIcon(Icons.stop), findsNothing);
  });
}
