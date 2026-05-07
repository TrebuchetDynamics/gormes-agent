import 'dart:async';
import 'dart:typed_data';

/// A duplex byte stream — the minimum surface a navivox channel needs from any
/// underlying transport (SSH session, WebSocket, in-memory test pair).
abstract interface class ByteTransport {
  /// Bytes received from the remote peer.
  Stream<Uint8List> get bytes;

  /// Bytes to send to the remote peer.
  StreamSink<Uint8List> get sink;

  /// Closes the transport and releases resources.
  Future<void> close();
}
