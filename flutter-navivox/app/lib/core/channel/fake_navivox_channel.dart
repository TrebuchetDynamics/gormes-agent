import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:uuid/uuid.dart';

import '../protocol/navivox_event.dart';
import '../protocol/navivox_frame.dart';
import 'navivox_channel.dart';

final fakeNavivoxChannelProvider = Provider<FakeNavivoxChannel>((ref) {
  final channel = FakeNavivoxChannel();
  ref.onDispose(channel.dispose);
  return channel;
});

class FakeNavivoxChannel extends ChangeNotifier implements NavivoxChannel {
  FakeNavivoxChannel({Uuid? uuid, DateTime Function()? clock})
    : _uuid = uuid ?? const Uuid(),
      _clock = clock ?? DateTime.now;

  final Uuid _uuid;
  final DateTime Function() _clock;
  NavivoxChannelState _state = const NavivoxChannelState();
  NavivoxFrame? _lastSentFrame;
  Uint8List? _lastSentFrameBytes;
  final List<NavivoxFrame> _lastReceivedFrames = [];

  final StreamController<NavivoxApprovalRequest> _approvals =
      StreamController<NavivoxApprovalRequest>.broadcast();
  final List<({String approvalId, bool approved})> _approvalResponses = [];

  @override
  NavivoxChannelState get state => _state;

  @override
  Stream<NavivoxApprovalRequest> get approvalRequests => _approvals.stream;

  /// Last frame produced by [sendText], encoded once and decoded again so tests
  /// can inspect the wire-level shape.
  NavivoxFrame? get lastSentFrame => _lastSentFrame;
  Uint8List? get lastSentFrameBytes => _lastSentFrameBytes;

  /// Frames synthesised as a mock server response to the last [sendText].
  List<NavivoxFrame> get lastReceivedFrames =>
      List.unmodifiable(_lastReceivedFrames);

  /// Approval responses captured by [respondToApproval] for tests to inspect.
  List<({String approvalId, bool approved})> get approvalResponses =>
      List.unmodifiable(_approvalResponses);

  /// Test-side hook: pretend the server requested approval.
  void emitApprovalRequest(NavivoxApprovalRequest request) {
    _approvals.add(request);
  }

  /// Test-side hook: pretend the server returned the config schema.
  void emitConfigSchema(Map<String, Object?> schema) {
    _state = _state.copyWith(configSchema: schema);
    notifyListeners();
  }

  /// Test-side hook: pretend the server returned the current config values.
  void emitConfigValues(Map<String, Object?> values) {
    _state = _state.copyWith(configValues: values);
    notifyListeners();
  }

  @override
  void respondToApproval({required String approvalId, required bool approved}) {
    _approvalResponses.add((approvalId: approvalId, approved: approved));
  }

  @override
  void requestAgentList() {
    // The fake channel publishes a built-in trio so the UI can render without
    // a real server.
    _state = _state.copyWith(agents: const [
      NavivoxAgent(id: 'default', name: 'Default', status: 'active'),
      NavivoxAgent(id: 'arch', name: 'Architect', status: 'active'),
    ]);
    notifyListeners();
  }

  @override
  void selectAgent(String agentId) {
    _state = _state.copyWith(selectedAgentId: agentId);
    notifyListeners();
  }

  @override
  void sendConfigSet({required String field, required Object? value}) {
    _state = _state.copyWith(
      configValues: {..._state.configValues, field: value},
    );
    notifyListeners();
  }

  @override
  void sendConfigSecretSet({required String name, required String secret}) {
    // The fake never stores the secret — UI must rely on server confirmation.
  }

  @override
  void dispose() {
    _approvals.close();
    super.dispose();
  }

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
    final now = _clock();
    final submitId = _uuid.v4();

    final submit = NavivoxFrame(
      type: 'chat.submit',
      messageId: submitId,
      timestamp: now,
      contentType: 'application/json',
      payload: Uint8List.fromList(utf8.encode(jsonEncode({'text': trimmed}))),
    );
    final wire = NavivoxFrameCodec.encode(submit);
    _lastSentFrame = NavivoxFrameCodec.decode(wire);
    _lastSentFrameBytes = wire;

    final assistantText = 'Mock response to: $trimmed';
    final messageFrame = NavivoxFrame(
      type: 'chat.message',
      messageId: _uuid.v4(),
      timestamp: now.add(const Duration(milliseconds: 150)),
      correlationId: submitId,
      contentType: 'application/json',
      payload: Uint8List.fromList(
        utf8.encode(jsonEncode({'text': assistantText})),
      ),
    );
    final finalFrame = NavivoxFrame(
      type: 'chat.final',
      messageId: _uuid.v4(),
      timestamp: now.add(const Duration(milliseconds: 200)),
      correlationId: submitId,
      contentType: 'application/json',
      payload: Uint8List.fromList(
        utf8.encode(jsonEncode({'turn_complete': true})),
      ),
    );

    _lastReceivedFrames
      ..clear()
      ..add(NavivoxFrameCodec.decode(NavivoxFrameCodec.encode(messageFrame)))
      ..add(NavivoxFrameCodec.decode(NavivoxFrameCodec.encode(finalFrame)));

    _state = _state.copyWith(
      messages: [
        ..._state.messages,
        NavivoxChatMessage(
          id: submitId,
          author: NavivoxMessageAuthor.user,
          kind: NavivoxMessageKind.text,
          createdAt: now,
          text: trimmed,
        ),
        NavivoxChatMessage(
          id: messageFrame.messageId,
          author: NavivoxMessageAuthor.assistant,
          kind: NavivoxMessageKind.text,
          createdAt: messageFrame.timestamp,
          text: assistantText,
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
    if (audio.isEmpty) {
      return;
    }
    final now = _clock();
    final submitId = _uuid.v4();

    final submit = NavivoxFrame(
      type: 'voice.submit',
      messageId: submitId,
      timestamp: now,
      contentType: 'audio/pcm',
      payload: audio,
      metadata: {
        'transcript': transcript,
        'duration_ms': duration.inMilliseconds,
        'confidence': confidence,
        'codec': 'pcm_s16le',
      },
    );
    final wire = NavivoxFrameCodec.encode(submit);
    _lastSentFrame = NavivoxFrameCodec.decode(wire);
    _lastSentFrameBytes = wire;

    final mockTranscript = transcript;
    final mockTtsBytes = Uint8List.fromList(
      List<int>.generate(audio.length.clamp(16, 256), (i) => (i * 7) % 256),
    );

    final transcriptFrame = NavivoxFrame(
      type: 'voice.transcript',
      messageId: _uuid.v4(),
      timestamp: now.add(const Duration(milliseconds: 100)),
      correlationId: submitId,
      contentType: 'application/json',
      payload: Uint8List.fromList(
        utf8.encode(jsonEncode({'transcript': mockTranscript, 'final': true})),
      ),
    );
    final audioFrame = NavivoxFrame(
      type: 'voice.audio',
      messageId: _uuid.v4(),
      timestamp: now.add(const Duration(milliseconds: 220)),
      correlationId: submitId,
      contentType: 'audio/pcm',
      payload: mockTtsBytes,
      metadata: {
        'codec': 'pcm_s16le',
        'duration_ms': duration.inMilliseconds + 200,
        'transcript': 'Acknowledged: $mockTranscript',
      },
    );

    _lastReceivedFrames
      ..clear()
      ..add(NavivoxFrameCodec.decode(NavivoxFrameCodec.encode(transcriptFrame)))
      ..add(NavivoxFrameCodec.decode(NavivoxFrameCodec.encode(audioFrame)));

    _state = _state.copyWith(
      messages: [
        ..._state.messages,
        NavivoxChatMessage(
          id: submitId,
          author: NavivoxMessageAuthor.user,
          kind: NavivoxMessageKind.voice,
          createdAt: now,
          voice: NavivoxVoiceMessage(
            duration: duration,
            transcript: transcript,
            confidence: confidence,
          ),
        ),
        NavivoxChatMessage(
          id: audioFrame.messageId,
          author: NavivoxMessageAuthor.assistant,
          kind: NavivoxMessageKind.voice,
          createdAt: audioFrame.timestamp,
          voice: NavivoxVoiceMessage(
            duration: duration + const Duration(milliseconds: 200),
            transcript: 'Acknowledged: $mockTranscript',
            confidence: 0.99,
          ),
        ),
      ],
    );
    notifyListeners();
  }
}
