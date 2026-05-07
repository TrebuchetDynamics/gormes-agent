import 'dart:async';
import 'dart:typed_data';

import 'package:dartssh2/dartssh2.dart';

import 'byte_transport.dart';

/// [ByteTransport] backed by a dartssh2 [SSHSession]. The session's `stdout`
/// stream becomes [bytes]; writes to [sink] flow into the session's `stdin`.
///
/// The adapter intentionally keeps a [fromParts] constructor so unit tests can
/// drive the channel with in-memory streams instead of opening a real SSH
/// session. Production code uses [Dartssh2ByteTransport.wrap].
class Dartssh2ByteTransport implements ByteTransport {
  Dartssh2ByteTransport.fromParts({
    required Stream<Uint8List> stdout,
    required StreamSink<List<int>> stdin,
    required Future<void> Function() onClose,
  })  : _stdout = stdout,
        _stdinSink = _Uint8ListSink(stdin),
        _onClose = onClose;

  /// Wrap a live [SSHSession] (the result of `client.execute(...)`).
  factory Dartssh2ByteTransport.wrap(SSHSession session) {
    return Dartssh2ByteTransport.fromParts(
      stdout: session.stdout,
      stdin: session.stdin,
      onClose: () async {
        // Send EOF, wait for the remote process to exit, then drop the session.
        await session.stdin.close();
        await session.done;
      },
    );
  }

  final Stream<Uint8List> _stdout;
  final StreamSink<Uint8List> _stdinSink;
  final Future<void> Function() _onClose;
  bool _closed = false;

  @override
  Stream<Uint8List> get bytes => _stdout;

  @override
  StreamSink<Uint8List> get sink => _stdinSink;

  @override
  Future<void> close() async {
    if (_closed) return;
    _closed = true;
    await _onClose();
  }
}

/// dartssh2's `SSHSession.stdin` is a `StreamSink<List<int>>` while our
/// [ByteTransport] contract is `StreamSink<Uint8List>`. This thin adapter
/// pushes byte chunks through without copying.
class _Uint8ListSink implements StreamSink<Uint8List> {
  _Uint8ListSink(this._inner);
  final StreamSink<List<int>> _inner;

  @override
  void add(Uint8List event) => _inner.add(event);

  @override
  void addError(Object error, [StackTrace? stackTrace]) =>
      _inner.addError(error, stackTrace);

  @override
  Future<void> addStream(Stream<Uint8List> stream) => _inner.addStream(stream);

  @override
  Future<void> close() => _inner.close();

  @override
  Future<void> get done => _inner.done;
}
