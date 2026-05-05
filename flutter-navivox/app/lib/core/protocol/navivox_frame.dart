import 'dart:convert';
import 'dart:typed_data';

const navivoxMagic = 'NVOX';
const navivoxProtocolVersion = 1;
const maxNavivoxFrameBytes = 1024 * 1024;

class NavivoxFrame {
  const NavivoxFrame({
    required this.type,
    required this.messageId,
    required this.timestamp,
    required this.payload,
    this.correlationId,
    this.turnId,
    this.agentId,
    this.contentType,
    this.metadata = const {},
  });

  final String type;
  final String messageId;
  final DateTime timestamp;
  final Uint8List payload;
  final String? correlationId;
  final String? turnId;
  final String? agentId;
  final String? contentType;
  final Map<String, Object?> metadata;
}

class NavivoxFrameCodec {
  const NavivoxFrameCodec._();

  static NavivoxFrame decode(Uint8List bytes) {
    if (bytes.length < 12) {
      throw const InvalidFrameException('frame prelude is incomplete');
    }
    if (bytes.length > maxNavivoxFrameBytes) {
      throw const FrameSizeExceededException();
    }

    final magic = ascii.decode(bytes.sublist(0, 4));
    if (magic != navivoxMagic) {
      throw const InvalidFrameException('invalid magic bytes');
    }

    final data = ByteData.sublistView(bytes);
    final version = data.getUint32(4, Endian.big);
    if (version != navivoxProtocolVersion) {
      throw UnsupportedVersionException(version);
    }

    final headerLength = data.getUint32(8, Endian.big);
    final payloadOffset = 12 + headerLength;
    if (payloadOffset > bytes.length) {
      throw const InvalidFrameException('header exceeds frame length');
    }

    final headerText = utf8.decode(bytes.sublist(12, payloadOffset));
    final header = jsonDecode(headerText);
    if (header is! Map<String, Object?>) {
      throw const InvalidFrameException('header must be a JSON object');
    }

    final payloadLength = header['payload_length'];
    if (payloadLength is! int || payloadLength < 0) {
      throw const InvalidFrameException('payload_length must be non-negative');
    }

    final actualPayloadLength = bytes.length - payloadOffset;
    if (payloadLength != actualPayloadLength) {
      throw PayloadLengthMismatchException(
        expected: payloadLength,
        actual: actualPayloadLength,
      );
    }

    return NavivoxFrame(
      type: _requiredString(header, 'type'),
      messageId: _requiredString(header, 'message_id'),
      timestamp: DateTime.parse(_requiredString(header, 'timestamp')),
      correlationId: header['correlation_id'] as String?,
      turnId: header['turn_id'] as String?,
      agentId: header['agent_id'] as String?,
      contentType: header['content_type'] as String?,
      metadata: (header['metadata'] as Map?)?.cast<String, Object?>() ?? {},
      payload: Uint8List.sublistView(bytes, payloadOffset),
    );
  }

  static String _requiredString(Map<String, Object?> header, String key) {
    final value = header[key];
    if (value is! String || value.isEmpty) {
      throw InvalidFrameException('$key must be a non-empty string');
    }
    return value;
  }
}

class InvalidFrameException implements Exception {
  const InvalidFrameException(this.message);

  final String message;

  @override
  String toString() => 'InvalidFrameException: $message';
}

class PayloadLengthMismatchException implements Exception {
  const PayloadLengthMismatchException({
    required this.expected,
    required this.actual,
  });

  final int expected;
  final int actual;

  @override
  String toString() =>
      'PayloadLengthMismatchException: expected $expected bytes, got $actual';
}

class FrameSizeExceededException implements Exception {
  const FrameSizeExceededException();
}

class UnsupportedVersionException implements Exception {
  const UnsupportedVersionException(this.version);

  final int version;
}
