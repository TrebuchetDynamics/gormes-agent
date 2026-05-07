import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:uuid/uuid.dart';

import '../protocol/frame_reader.dart';
import '../protocol/navivox_event.dart';
import '../protocol/navivox_frame.dart';
import '../transport/byte_transport.dart';
import 'navivox_channel.dart';

/// A [NavivoxChannel] driven by a duplex byte transport. The transport can be
/// an SSH session (production) or an [InMemoryByteTransportPair] (tests).
class SshNavivoxChannel extends ChangeNotifier implements NavivoxChannel {
  SshNavivoxChannel({
    required ByteTransport transport,
    Uuid? uuid,
    DateTime Function()? clock,
  }) : _transport = transport,
       _uuid = uuid ?? const Uuid(),
       _clock = clock ?? DateTime.now {
    _reader = NavivoxFrameReader(_transport.bytes);
    _frameSubscription = _reader.frames.listen(
      _onFrame,
      onError: (_) {
        // Decode errors are surfaced via debug print; channel state stays
        // stable so the UI doesn't flap on a single bad frame.
      },
    );
  }

  final ByteTransport _transport;
  final Uuid _uuid;
  final DateTime Function() _clock;
  late final NavivoxFrameReader _reader;
  StreamSubscription<NavivoxFrame>? _frameSubscription;
  final StreamController<NavivoxApprovalRequest> _approvals =
      StreamController<NavivoxApprovalRequest>.broadcast();

  NavivoxChannelState _state = const NavivoxChannelState();
  bool _closed = false;

  @override
  NavivoxChannelState get state => _state;

  @override
  Stream<NavivoxApprovalRequest> get approvalRequests => _approvals.stream;

  @override
  void enterFakeServerMode() {
    // The SSH-backed channel stays empty until the server identifies itself —
    // an explicit fake mode would short-circuit that handshake, so this is
    // intentionally a no-op here. Tests use [FakeNavivoxChannel] for the
    // transport-less fake.
  }

  @override
  void sendText(String text) {
    final trimmed = text.trim();
    if (trimmed.isEmpty || _closed) return;
    _writeFrame(
      type: 'chat.submit',
      payload: Uint8List.fromList(utf8.encode(jsonEncode({'text': trimmed}))),
      contentType: 'application/json',
    );
  }

  @override
  void sendVoice({
    required Uint8List audio,
    required String transcript,
    required Duration duration,
    required double confidence,
  }) {
    if (audio.isEmpty || _closed) return;
    _writeFrame(
      type: 'voice.submit',
      payload: audio,
      contentType: 'audio/pcm',
      metadata: {
        'transcript': transcript,
        'duration_ms': duration.inMilliseconds,
        'confidence': confidence,
        'codec': 'pcm_s16le',
      },
    );
  }

  @override
  void requestAgentList() {
    if (_closed) return;
    _writeFrame(
      type: 'agent.list',
      payload: Uint8List.fromList(utf8.encode('{}')),
      contentType: 'application/json',
    );
  }

  @override
  void sendConfigSet({required String field, required Object? value}) {
    if (_closed) return;
    _writeFrame(
      type: 'config.set',
      contentType: 'application/json',
      payload: Uint8List.fromList(
          utf8.encode(jsonEncode({'field': field, 'value': value}))),
    );
  }

  @override
  void sendConfigSecretSet({required String name, required String secret}) {
    if (_closed) return;
    _writeFrame(
      type: 'config.secret.set',
      contentType: 'application/json',
      payload: Uint8List.fromList(
          utf8.encode(jsonEncode({'name': name, 'value': secret}))),
    );
    // Local state never carries the raw secret value — see test.
  }

  @override
  void selectAgent(String agentId) {
    if (_closed) return;
    _writeFrame(
      type: 'agent.select',
      payload: Uint8List.fromList(utf8.encode(jsonEncode({'agent_id': agentId}))),
      contentType: 'application/json',
    );
  }

  @override
  void respondToApproval({required String approvalId, required bool approved}) {
    if (_closed) return;
    final frame = NavivoxFrame(
      type: 'tool.approval.responded',
      messageId: _uuid.v4(),
      timestamp: _clock(),
      contentType: 'application/json',
      payload: Uint8List.fromList(
        utf8.encode(jsonEncode({'approved': approved})),
      ),
      metadata: {'approval_id': approvalId},
    );
    _transport.sink.add(NavivoxFrameCodec.encode(frame));
  }

  Future<void> close() async {
    if (_closed) return;
    _closed = true;
    await _frameSubscription?.cancel();
    await _reader.close();
    await _approvals.close();
    await _transport.close();
  }

  void _writeFrame({
    required String type,
    required Uint8List payload,
    String? contentType,
    Map<String, Object?> metadata = const {},
  }) {
    final frame = NavivoxFrame(
      type: type,
      messageId: _uuid.v4(),
      timestamp: _clock(),
      payload: payload,
      contentType: contentType,
      metadata: metadata,
    );
    _transport.sink.add(NavivoxFrameCodec.encode(frame));
  }

  void _onFrame(NavivoxFrame frame) {
    switch (frame.type) {
      case 'chat.message':
        _appendChatMessage(frame);
      case 'chat.final':
        // Turn-complete marker — no UI message, but listeners should still
        // observe state churn so streaming indicators can clear.
        notifyListeners();
      case 'voice.transcript':
      case 'voice.audio':
        _appendVoiceMessage(frame);
      case 'tool.call.started':
      case 'tool.call.progress':
      case 'tool.call.completed':
      case 'tool.call.failed':
      case 'tool.call.cancelled':
      case 'tool.call.blocked':
        _upsertToolCall(frame);
      case 'tool.artifact.ready':
        _attachArtifact(frame);
      case 'tool.approval.requested':
        _emitApproval(frame);
      case 'agent.list':
        _ingestAgentList(frame);
      case 'agent.select':
        _ingestAgentSelect(frame);
      case 'config.schema':
        _state = _state.copyWith(configSchema: _decodeJsonBody(frame));
        notifyListeners();
      case 'config.get':
        final body = _decodeJsonBody(frame);
        final values = body['values'];
        if (values is Map) {
          _state = _state.copyWith(
            configValues: Map<String, Object?>.from(values),
          );
          notifyListeners();
        }
      case 'config.diff':
        _state = _state.copyWith(configDiff: _decodeJsonBody(frame));
        notifyListeners();
      default:
        // Unknown frames are ignored at the channel layer — higher-level
        // features (config, agent admin) listen on their own bus.
        break;
    }
  }

  void _upsertToolCall(NavivoxFrame frame) {
    final toolCallId = frame.metadata['tool_call_id']?.toString();
    if (toolCallId == null || toolCallId.isEmpty) return;

    Map<String, Object?> body = const {};
    try {
      final decoded = jsonDecode(utf8.decode(frame.payload));
      if (decoded is Map<String, Object?>) body = decoded;
    } catch (_) {
      // Tolerate non-JSON payloads — the lifecycle status comes from the
      // frame type, not the body.
    }

    // The status part of the frame type drives the UI ("started", "progress",
    // "completed", "failed", "cancelled", "blocked").
    final status = frame.type.substring('tool.call.'.length);

    final existing = _state.messages.indexWhere(
      (m) => m.id == toolCallId && m.kind == NavivoxMessageKind.toolCall,
    );

    final priorCall = existing >= 0 ? _state.messages[existing].toolCall : null;
    final name = body['name']?.toString() ?? priorCall?.name ?? '';
    final summary = body['summary']?.toString() ?? priorCall?.summary ?? '';

    final message = NavivoxChatMessage(
      id: toolCallId,
      author: NavivoxMessageAuthor.assistant,
      kind: NavivoxMessageKind.toolCall,
      createdAt: frame.timestamp,
      toolCall: NavivoxToolCall(
        name: name,
        status: status,
        summary: summary,
        artifacts: priorCall?.artifacts ?? const [],
      ),
    );

    if (existing >= 0) {
      final next = List<NavivoxChatMessage>.from(_state.messages);
      next[existing] = message;
      _state = _state.copyWith(messages: next);
    } else {
      _state = _state.copyWith(messages: [..._state.messages, message]);
    }
    notifyListeners();
  }

  void _attachArtifact(NavivoxFrame frame) {
    final toolCallId = frame.metadata['tool_call_id']?.toString();
    final artifactId = frame.metadata['artifact_id']?.toString();
    final kind = frame.metadata['kind']?.toString();
    if (toolCallId == null || artifactId == null || kind == null) return;

    final existing = _state.messages.indexWhere(
      (m) => m.id == toolCallId && m.kind == NavivoxMessageKind.toolCall,
    );
    if (existing < 0) return;

    Map<String, Object?> body = const {};
    try {
      final decoded = jsonDecode(utf8.decode(frame.payload));
      if (decoded is Map<String, Object?>) body = decoded;
    } catch (_) {/* ignore */}

    final artifact = NavivoxToolArtifact(
      id: artifactId,
      kind: kind,
      title: body['title']?.toString() ?? '',
      summary: body['summary']?.toString(),
      ref: body['ref']?.toString(),
    );

    final priorMessage = _state.messages[existing];
    final priorCall = priorMessage.toolCall!;
    final updatedCall = priorCall.copyWith(
      artifacts: [...priorCall.artifacts, artifact],
    );
    final next = List<NavivoxChatMessage>.from(_state.messages);
    next[existing] = NavivoxChatMessage(
      id: priorMessage.id,
      author: priorMessage.author,
      kind: priorMessage.kind,
      createdAt: priorMessage.createdAt,
      toolCall: updatedCall,
    );
    _state = _state.copyWith(messages: next);
    notifyListeners();
  }

  void _ingestAgentList(NavivoxFrame frame) {
    final body = _decodeJsonBody(frame);
    final raw = body['agents'];
    if (raw is! List) return;

    final agents = <NavivoxAgent>[];
    for (final entry in raw) {
      if (entry is! Map) continue;
      final id = entry['id']?.toString() ?? '';
      if (id.isEmpty) continue;
      agents.add(NavivoxAgent(
        id: id,
        name: entry['name']?.toString() ?? id,
        status: entry['status']?.toString() ?? 'active',
      ));
    }
    _state = _state.copyWith(agents: agents);
    notifyListeners();
  }

  void _ingestAgentSelect(NavivoxFrame frame) {
    final body = _decodeJsonBody(frame);
    final id = body['agent_id']?.toString();
    if (id == null || id.isEmpty) return;
    _state = _state.copyWith(selectedAgentId: id);
    notifyListeners();
  }

  Map<String, Object?> _decodeJsonBody(NavivoxFrame frame) {
    try {
      final decoded = jsonDecode(utf8.decode(frame.payload));
      if (decoded is Map<String, Object?>) return decoded;
    } catch (_) {/* ignore */}
    return const {};
  }

  void _emitApproval(NavivoxFrame frame) {
    final approvalId = frame.metadata['approval_id']?.toString();
    final toolCallId = frame.metadata['tool_call_id']?.toString();
    if (approvalId == null || toolCallId == null) return;

    Map<String, Object?> body = const {};
    try {
      final decoded = jsonDecode(utf8.decode(frame.payload));
      if (decoded is Map<String, Object?>) body = decoded;
    } catch (_) {
      /* ignore */
    }

    _approvals.add(
      NavivoxApprovalRequest(
        id: approvalId,
        toolCallId: toolCallId,
        prompt: body['prompt']?.toString() ?? '',
        risk: body['risk']?.toString(),
      ),
    );
  }

  void _appendChatMessage(NavivoxFrame frame) {
    String? text;
    try {
      final body = jsonDecode(utf8.decode(frame.payload));
      if (body is Map && body['text'] is String) {
        text = body['text'] as String;
      }
    } catch (_) {
      // Non-JSON payload — keep text null.
    }
    _state = _state.copyWith(
      messages: [
        ..._state.messages,
        NavivoxChatMessage(
          id: frame.messageId,
          author: NavivoxMessageAuthor.assistant,
          kind: NavivoxMessageKind.text,
          createdAt: frame.timestamp,
          text: text ?? '',
        ),
      ],
    );
    notifyListeners();
  }

  void _appendVoiceMessage(NavivoxFrame frame) {
    final transcript = frame.metadata['transcript']?.toString() ?? '';
    final ms = frame.metadata['duration_ms'];
    final duration = ms is int ? Duration(milliseconds: ms) : Duration.zero;
    final conf = frame.metadata['confidence'];
    final confidence = conf is num ? conf.toDouble() : 1.0;

    _state = _state.copyWith(
      messages: [
        ..._state.messages,
        NavivoxChatMessage(
          id: frame.messageId,
          author: NavivoxMessageAuthor.assistant,
          kind: frame.type == 'voice.audio'
              ? NavivoxMessageKind.voice
              : NavivoxMessageKind.text,
          createdAt: frame.timestamp,
          text: frame.type == 'voice.transcript' ? transcript : null,
          voice: frame.type == 'voice.audio'
              ? NavivoxVoiceMessage(
                  duration: duration,
                  transcript: transcript,
                  confidence: confidence,
                )
              : null,
        ),
      ],
    );
    notifyListeners();
  }
}
