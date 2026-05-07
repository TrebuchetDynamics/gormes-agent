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

  NavivoxChannelState _state = const NavivoxChannelState();
  bool _closed = false;

  @override
  NavivoxChannelState get state => _state;

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

  Future<void> close() async {
    if (_closed) return;
    _closed = true;
    await _frameSubscription?.cancel();
    await _reader.close();
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
      default:
        // Unknown frames are ignored at the channel layer — higher-level
        // features (config, agent, tool) listen on their own bus.
        break;
    }
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
