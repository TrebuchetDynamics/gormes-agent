import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/features/keys/services/openssh_key_validator.dart';

void main() {
  group('OpenSshKeyValidator', () {
    test('accepts a synthetic unencrypted OpenSSH key envelope', () {
      final result = OpenSshKeyValidator.validate(_openSshPem('none'));
      expect(result.isValid, isTrue);
      expect(result.format, OpenSshKeyFormat.openssh);
      expect(result.isEncrypted, isFalse);
      expect(result.error, isNull);
    });

    test('flags encrypted private keys', () {
      final result = OpenSshKeyValidator.validate(_openSshPem('aes256-ctr'));
      expect(result.isValid, isTrue);
      expect(result.isEncrypted, isTrue);
    });

    test('rejects empty input', () {
      final result = OpenSshKeyValidator.validate('');
      expect(result.isValid, isFalse);
      expect(result.error, isNotNull);
    });

    test('rejects PEM with wrong markers', () {
      final result = OpenSshKeyValidator.validate(
        '-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA...\n'
        '-----END RSA PRIVATE KEY-----\n',
      );
      expect(result.isValid, isFalse);
      expect(result.format, OpenSshKeyFormat.unknown);
      expect(result.error, contains('OpenSSH'));
    });

    test('rejects PEM with mismatched begin/end markers', () {
      final result = OpenSshKeyValidator.validate(
        '-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAA\n'
        '-----END RSA PRIVATE KEY-----\n',
      );
      expect(result.isValid, isFalse);
      expect(result.error, isNotNull);
    });

    test('rejects PEM whose body fails to base64-decode', () {
      final result = OpenSshKeyValidator.validate(
        '-----BEGIN OPENSSH PRIVATE KEY-----\n!!!not-base64!!!\n'
        '-----END OPENSSH PRIVATE KEY-----\n',
      );
      expect(result.isValid, isFalse);
      expect(result.error, isNotNull);
    });

    test('rejects PEM whose decoded body lacks the OpenSSH magic prefix', () {
      // "Hello, OpenSSH!" base64 → does not start with "openssh-key-v1\0"
      final result = OpenSshKeyValidator.validate(
        '-----BEGIN OPENSSH PRIVATE KEY-----\n'
        'SGVsbG8sIE9wZW5TU0gh\n'
        '-----END OPENSSH PRIVATE KEY-----\n',
      );
      expect(result.isValid, isFalse);
      expect(result.error, contains('magic'));
    });
  });
}

String _openSshPem(String cipherName) {
  final cipherBytes = utf8.encode(cipherName);
  final bytes = BytesBuilder(copy: false)
    ..add(utf8.encode('openssh-key-v1'))
    ..addByte(0)
    ..add(_uint32(cipherBytes.length))
    ..add(cipherBytes);

  return '-----BEGIN OPENSSH PRIVATE KEY-----\n'
      '${base64.encode(bytes.toBytes())}\n'
      '-----END OPENSSH PRIVATE KEY-----\n';
}

Uint8List _uint32(int value) {
  final data = ByteData(4)..setUint32(0, value, Endian.big);
  return data.buffer.asUint8List();
}
