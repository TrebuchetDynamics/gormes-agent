import 'dart:async';
import 'dart:typed_data';

import 'byte_transport.dart';

/// A pair of [ByteTransport]s wired so client writes appear on the server's
/// read stream and vice versa. Used in tests to drive the channel without a
/// real SSH session.
class InMemoryByteTransportPair {
  InMemoryByteTransportPair() {
    _clientToServer = StreamController<Uint8List>();
    _serverToClient = StreamController<Uint8List>();
    client = _Endpoint(
      bytes: _serverToClient.stream,
      sink: _clientToServer.sink,
      onClose: _close,
    );
  }

  late final StreamController<Uint8List> _clientToServer;
  late final StreamController<Uint8List> _serverToClient;
  late final ByteTransport client;

  StreamSubscription<Uint8List>? _serverSubscription;
  bool _closed = false;

  /// Writes raw bytes from the simulated server to the client side.
  void serverWrite(Uint8List bytes) {
    if (_closed) return;
    _serverToClient.add(bytes);
  }

  /// Reads exactly one chunk from the client→server pipe.
  Future<Uint8List> serverReadOne() {
    return _clientToServer.stream.first;
  }

  /// Reads up to [count] bytes from the client→server pipe.
  Future<Uint8List> serverReadBytes(int count) async {
    final out = BytesBuilder(copy: false);
    await for (final chunk in _clientToServer.stream) {
      out.add(chunk);
      if (out.length >= count) break;
    }
    return out.toBytes();
  }

  /// Drains everything the client has written so far.
  Future<List<int>> serverReadAll() async {
    final out = <int>[];
    await for (final chunk in _clientToServer.stream) {
      out.addAll(chunk);
    }
    return out;
  }

  Future<void> _close() async {
    if (_closed) return;
    _closed = true;
    await _serverSubscription?.cancel();
    // Single-subscription StreamController.close() waits for the consumer to
    // drain buffered events. With no listener attached yet (e.g. close-before-
    // read in tests) that hangs forever. Fire and forget — late listeners
    // still receive buffered events plus done.
    unawaited(_clientToServer.close());
    unawaited(_serverToClient.close());
  }
}

class _Endpoint implements ByteTransport {
  _Endpoint({
    required Stream<Uint8List> bytes,
    required StreamSink<Uint8List> sink,
    required Future<void> Function() onClose,
  }) : _bytes = bytes,
       _sink = sink,
       _onClose = onClose;

  final Stream<Uint8List> _bytes;
  final StreamSink<Uint8List> _sink;
  final Future<void> Function() _onClose;

  @override
  Stream<Uint8List> get bytes => _bytes;

  @override
  StreamSink<Uint8List> get sink => _sink;

  @override
  Future<void> close() => _onClose();
}
