import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'navivox_frame.dart';

/// Buffers an incoming byte stream and emits one [NavivoxFrame] per fully
/// received frame. Handles partial reads (one frame split across many chunks)
/// and concatenated frames (multiple frames in a single chunk). Decode errors
/// are exposed on a separate [errors] stream so callers can observe them
/// without entangling them with the frame stream's data path.
class NavivoxFrameReader {
  NavivoxFrameReader(Stream<Uint8List> source) {
    _output = StreamController<NavivoxFrame>.broadcast();
    _errors = StreamController<Object>.broadcast();
    _subscription = source.listen(
      _onBytes,
      onError: _errors.add,
      onDone: _onDone,
      cancelOnError: false,
    );
  }

  late final StreamController<NavivoxFrame> _output;
  late final StreamController<Object> _errors;
  StreamSubscription<Uint8List>? _subscription;
  final BytesBuilder _buffer = BytesBuilder(copy: false);

  Stream<NavivoxFrame> get frames => _output.stream;
  Stream<Object> get errors => _errors.stream;

  Future<void> close() async {
    await _subscription?.cancel();
    await _output.close();
    await _errors.close();
  }

  void _onDone() {
    _output.close();
    _errors.close();
  }

  void _onBytes(Uint8List chunk) {
    _buffer.add(chunk);
    _drain();
  }

  void _drain() {
    while (true) {
      final view = _buffer.toBytes();
      if (view.length < 12) return;

      // Validate magic before trusting the header-length field. If magic is
      // wrong, drop one byte and resync.
      if (view[0] != 0x4E ||
          view[1] != 0x56 ||
          view[2] != 0x4F ||
          view[3] != 0x58) {
        _errors.add(const InvalidFrameException('invalid magic bytes'));
        _buffer.clear();
        if (view.length > 1) _buffer.add(Uint8List.sublistView(view, 1));
        continue;
      }

      final headerLength = ByteData.sublistView(view).getUint32(8, Endian.big);
      final headerEnd = 12 + headerLength;
      if (view.length < headerEnd) return;

      final int payloadLength;
      try {
        final headerJson = json.decode(
          utf8.decode(view.sublist(12, headerEnd)),
        );
        if (headerJson is! Map) {
          throw const InvalidFrameException('header must be a JSON object');
        }
        final v = headerJson['payload_length'];
        if (v is! int || v < 0) {
          throw const InvalidFrameException(
            'payload_length must be non-negative',
          );
        }
        payloadLength = v;
      } catch (e) {
        _errors.add(
          e is Exception ? e : InvalidFrameException('header parse failed: $e'),
        );
        _buffer.clear();
        if (view.length > 1) _buffer.add(Uint8List.sublistView(view, 1));
        continue;
      }

      final frameLength = headerEnd + payloadLength;
      if (view.length < frameLength) return;

      final frameBytes = Uint8List.sublistView(view, 0, frameLength);
      _buffer.clear();
      if (frameLength < view.length) {
        _buffer.add(Uint8List.sublistView(view, frameLength));
      }

      try {
        _output.add(NavivoxFrameCodec.decode(frameBytes));
      } catch (e) {
        _errors.add(
          e is Exception ? e : InvalidFrameException('decode failed: $e'),
        );
      }
    }
  }
}
