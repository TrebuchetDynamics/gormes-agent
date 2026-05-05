import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:uuid/uuid.dart';

import '../protocol/navivox_event.dart';
import 'navivox_channel.dart';

final fakeNavivoxChannelProvider = Provider<FakeNavivoxChannel>((ref) {
  final channel = FakeNavivoxChannel();
  ref.onDispose(channel.dispose);
  return channel;
});

class FakeNavivoxChannel extends ChangeNotifier implements NavivoxChannel {
  FakeNavivoxChannel({Uuid? uuid}) : _uuid = uuid ?? const Uuid();

  final Uuid _uuid;
  NavivoxChannelState _state = const NavivoxChannelState();

  @override
  NavivoxChannelState get state => _state;

  @override
  void enterFakeServerMode() {
    final server = const NavivoxServer(
      id: 'fake-local',
      name: 'Fake Local Gormes',
      status: 'Server online',
    );
    final now = DateTime(2026, 5, 5, 12);

    _state = NavivoxChannelState(
      servers: [server],
      activeServerId: server.id,
      messages: [
        NavivoxChatMessage(
          id: _uuid.v4(),
          author: NavivoxMessageAuthor.system,
          kind: NavivoxMessageKind.text,
          createdAt: now,
          text: 'server.status: Server online',
        ),
        NavivoxChatMessage(
          id: _uuid.v4(),
          author: NavivoxMessageAuthor.assistant,
          kind: NavivoxMessageKind.text,
          createdAt: now.add(const Duration(seconds: 1)),
          text: 'Navivox fake channel is ready for local protocol UI work.',
        ),
        NavivoxChatMessage(
          id: _uuid.v4(),
          author: NavivoxMessageAuthor.assistant,
          kind: NavivoxMessageKind.toolCall,
          createdAt: now.add(const Duration(seconds: 2)),
          toolCall: const NavivoxToolCall(
            name: 'workspace.read',
            status: 'completed',
            summary: 'Read local project context without contacting a server.',
          ),
        ),
        NavivoxChatMessage(
          id: _uuid.v4(),
          author: NavivoxMessageAuthor.assistant,
          kind: NavivoxMessageKind.voice,
          createdAt: now.add(const Duration(seconds: 3)),
          voice: const NavivoxVoiceMessage(
            duration: Duration(seconds: 7),
            transcript: 'Connected to fake local mode.',
            confidence: 0.96,
          ),
        ),
      ],
    );
    notifyListeners();
  }

  @override
  void sendText(String text) {
    final trimmed = text.trim();
    if (trimmed.isEmpty) {
      return;
    }
    final now = DateTime.now();
    _state = _state.copyWith(
      messages: [
        ..._state.messages,
        NavivoxChatMessage(
          id: _uuid.v4(),
          author: NavivoxMessageAuthor.user,
          kind: NavivoxMessageKind.text,
          createdAt: now,
          text: trimmed,
        ),
        NavivoxChatMessage(
          id: _uuid.v4(),
          author: NavivoxMessageAuthor.assistant,
          kind: NavivoxMessageKind.text,
          createdAt: now.add(const Duration(milliseconds: 200)),
          text: 'Fake echo: $trimmed',
        ),
      ],
    );
    notifyListeners();
  }
}
