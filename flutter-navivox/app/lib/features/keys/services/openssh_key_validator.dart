import 'dart:convert';
import 'dart:typed_data';

enum OpenSshKeyFormat { openssh, unknown }

class OpenSshKeyValidationResult {
  const OpenSshKeyValidationResult({
    required this.isValid,
    required this.format,
    required this.isEncrypted,
    this.error,
  });

  final bool isValid;
  final OpenSshKeyFormat format;
  final bool isEncrypted;
  final String? error;
}

/// Lightweight syntactic validation of OpenSSH-format private keys. We only
/// need enough confidence to:
///   * accept or reject a pasted/imported key,
///   * tell whether it is passphrase-encrypted (so the user knows to expect a
///     passphrase prompt on first use).
/// Full cryptographic parsing is left to the SSH transport layer when the
/// caller actually opens a session.
class OpenSshKeyValidator {
  static const _beginMarker = '-----BEGIN OPENSSH PRIVATE KEY-----';
  static const _endMarker = '-----END OPENSSH PRIVATE KEY-----';
  static const _openSshMagic = 'openssh-key-v1\u0000';

  static OpenSshKeyValidationResult validate(String pem) {
    if (pem.trim().isEmpty) {
      return const OpenSshKeyValidationResult(
        isValid: false,
        format: OpenSshKeyFormat.unknown,
        isEncrypted: false,
        error: 'empty key',
      );
    }

    if (!pem.contains(_beginMarker) || !pem.contains(_endMarker)) {
      // Look for any "PRIVATE KEY" markers to give a useful hint.
      final hasOtherPrivateKeyMarker =
          pem.contains('-----BEGIN ') && pem.contains('PRIVATE KEY-----');
      return OpenSshKeyValidationResult(
        isValid: false,
        format: OpenSshKeyFormat.unknown,
        isEncrypted: false,
        error: hasOtherPrivateKeyMarker
            ? 'expected an OpenSSH-format private key (use ssh-keygen -p -m RFC4716)'
            : 'no OpenSSH PEM markers found',
      );
    }

    final beginIndex = pem.indexOf(_beginMarker);
    final endIndex = pem.indexOf(_endMarker);
    if (endIndex < beginIndex) {
      return const OpenSshKeyValidationResult(
        isValid: false,
        format: OpenSshKeyFormat.openssh,
        isEncrypted: false,
        error: 'mismatched begin/end markers',
      );
    }

    final body = pem
        .substring(beginIndex + _beginMarker.length, endIndex)
        .replaceAll(RegExp(r'\s'), '');

    final Uint8List decoded;
    try {
      decoded = base64.decode(body);
    } catch (_) {
      return const OpenSshKeyValidationResult(
        isValid: false,
        format: OpenSshKeyFormat.openssh,
        isEncrypted: false,
        error: 'body is not valid base64',
      );
    }

    if (decoded.length < _openSshMagic.length) {
      return const OpenSshKeyValidationResult(
        isValid: false,
        format: OpenSshKeyFormat.openssh,
        isEncrypted: false,
        error: 'decoded body too short for OpenSSH magic prefix',
      );
    }

    final magic = String.fromCharCodes(
      decoded.sublist(0, _openSshMagic.length),
    );
    if (magic != _openSshMagic) {
      return const OpenSshKeyValidationResult(
        isValid: false,
        format: OpenSshKeyFormat.openssh,
        isEncrypted: false,
        error: 'OpenSSH magic prefix missing',
      );
    }

    // After the magic, the next length-prefixed string is the cipher name.
    // "none" means unencrypted; anything else (aes256-ctr, etc.) means
    // passphrase-encrypted.
    final isEncrypted = !_cipherIsNone(decoded);

    return OpenSshKeyValidationResult(
      isValid: true,
      format: OpenSshKeyFormat.openssh,
      isEncrypted: isEncrypted,
    );
  }

  static bool _cipherIsNone(Uint8List body) {
    final cipherStart = _openSshMagic.length;
    if (body.length < cipherStart + 4) return false;
    final view = ByteData.sublistView(body);
    final cipherLen = view.getUint32(cipherStart, Endian.big);
    final cipherEnd = cipherStart + 4 + cipherLen;
    if (cipherLen <= 0 || cipherEnd > body.length) return false;
    final cipherName = utf8.decode(body.sublist(cipherStart + 4, cipherEnd));
    return cipherName == 'none';
  }
}
