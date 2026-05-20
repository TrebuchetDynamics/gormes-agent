import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/protocol/navivox_event.dart';
import 'package:navivox/features/chat/widgets/simple_chat_adapter.dart';

void main() {
  testWidgets(
    'tool-call tile shows nothing extra when there are no artifacts',
    (tester) async {
      final message = NavivoxChatMessage(
        id: 'm-1',
        author: NavivoxMessageAuthor.assistant,
        kind: NavivoxMessageKind.toolCall,
        createdAt: DateTime(2026, 5, 7, 10),
        toolCall: const NavivoxToolCall(
          name: 'shell.run',
          status: 'completed',
          summary: 'ls -la',
        ),
      );

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SimpleChatAdapter(messages: [message], onSend: (_) {}),
          ),
        ),
      );

      expect(find.text('shell.run'), findsOneWidget);
      expect(find.text('completed'), findsOneWidget);
      expect(find.text('ls -la'), findsOneWidget);
      expect(find.byIcon(Icons.attachment), findsNothing);
    },
  );

  testWidgets(
    'tool-call tile lists each artifact with kind + title + summary',
    (tester) async {
      final message = NavivoxChatMessage(
        id: 'm-2',
        author: NavivoxMessageAuthor.assistant,
        kind: NavivoxMessageKind.toolCall,
        createdAt: DateTime(2026, 5, 7, 10),
        toolCall: const NavivoxToolCall(
          name: 'shell.run',
          status: 'completed',
          summary: 'ran git diff',
          artifacts: [
            NavivoxToolArtifact(
              id: 'a-1',
              kind: 'file',
              title: 'diff.patch',
              summary: '14 lines changed',
            ),
            NavivoxToolArtifact(
              id: 'a-2',
              kind: 'image',
              title: 'screenshot.png',
              ref: 'artifacts/a-2',
            ),
          ],
        ),
      );

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SimpleChatAdapter(messages: [message], onSend: (_) {}),
          ),
        ),
      );

      expect(find.byIcon(Icons.attachment), findsNWidgets(2));
      expect(find.text('diff.patch'), findsOneWidget);
      expect(find.text('14 lines changed'), findsOneWidget);
      expect(find.text('screenshot.png'), findsOneWidget);
      expect(find.text('image'), findsOneWidget);
      expect(find.text('file'), findsOneWidget);
    },
  );
}
