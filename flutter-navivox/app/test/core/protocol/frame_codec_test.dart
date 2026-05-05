import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/protocol/navivox_frame.dart';

void main() {
  test('protocol frame validates payload length while decoding', () {
    final header = <String, Object?>{
      'type': 'chat.submit',
      'message_id': 'frame-1',
      'timestamp': '2026-05-05T00:00:00Z',
      'payload_length': 5,
    };
    final headerBytes = utf8.encode(jsonEncode(header));
    final bytes = BytesBuilder()
      ..add(utf8.encode('NVOX'))
      ..add(_uint32(1))
      ..add(_uint32(headerBytes.length))
      ..add(headerBytes)
      ..add(utf8.encode('oops'));

    expect(
      () => NavivoxFrameCodec.decode(bytes.toBytes()),
      throwsA(isA<PayloadLengthMismatchException>()),
    );
  });
}

Uint8List _uint32(int value) {
  final data = ByteData(4)..setUint32(0, value, Endian.big);
  return data.buffer.asUint8List();
}
