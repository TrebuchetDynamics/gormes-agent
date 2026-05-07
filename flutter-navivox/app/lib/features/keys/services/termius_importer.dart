/// Pure-Dart Termius export parser. Produces import-ready models plus a list
/// of warnings the UI can show before the user commits the import.

class TermiusImportException implements Exception {
  const TermiusImportException(this.message);
  final String message;
  @override
  String toString() => 'TermiusImportException: $message';
}

class TermiusIdentity {
  const TermiusIdentity({
    required this.termiusId,
    required this.label,
    required this.privateKeyPem,
    required this.publicKey,
    required this.isEncrypted,
  });

  final String termiusId;
  final String label;
  final String? privateKeyPem;
  final String? publicKey;
  final bool isEncrypted;
}

class TermiusServer {
  const TermiusServer({
    required this.termiusId,
    required this.label,
    required this.hostname,
    required this.port,
    required this.username,
    this.termiusIdentity,
    this.group,
    this.tags = const [],
    this.knownHostKey,
  });

  final String termiusId;
  final String label;
  final String hostname;
  final int port;
  final String username;
  final String? termiusIdentity;
  final String? group;
  final List<String> tags;
  final String? knownHostKey;
}

class TermiusImportResult {
  const TermiusImportResult({
    required this.identities,
    required this.servers,
    required this.warnings,
  });

  final List<TermiusIdentity> identities;
  final List<TermiusServer> servers;
  final List<String> warnings;
}

class TermiusImporter {
  const TermiusImporter._();

  static TermiusImportResult parse(Object? json) {
    if (json is! Map<String, Object?>) {
      throw const TermiusImportException('export must be a JSON object');
    }

    final hostsRaw = json['hosts'] ?? const <Object?>[];
    final identitiesRaw = json['identities'] ?? const <Object?>[];

    if (hostsRaw is! List) {
      throw const TermiusImportException('"hosts" must be a list');
    }
    if (identitiesRaw is! List) {
      throw const TermiusImportException('"identities" must be a list');
    }

    final warnings = <String>[];
    final identities = <TermiusIdentity>[];
    final servers = <TermiusServer>[];

    for (final entry in identitiesRaw) {
      if (entry is! Map<String, Object?>) continue;
      final id = entry['id']?.toString() ?? '';
      final label = entry['label']?.toString() ?? '';
      final privateKey = entry['private_key'] as String?;
      final publicKey = entry['public_key'] as String?;
      final passphrase = entry['passphrase'];

      if ((privateKey == null || privateKey.isEmpty) &&
          (publicKey == null || publicKey.isEmpty)) {
        warnings.add('Skipped password-only identity "$label" ($id)');
        continue;
      }

      final isEncrypted = passphrase is String && passphrase.isNotEmpty;
      identities.add(
        TermiusIdentity(
          termiusId: id,
          label: label,
          privateKeyPem: privateKey,
          publicKey: publicKey,
          isEncrypted: isEncrypted,
        ),
      );
    }

    for (final entry in hostsRaw) {
      if (entry is! Map<String, Object?>) continue;
      final id = entry['id']?.toString() ?? '';
      final label = entry['label']?.toString() ?? '';
      final hostname = entry['hostname']?.toString() ?? '';
      final port = entry['port'] is int ? entry['port'] as int : 22;
      final username = entry['username']?.toString() ?? '';
      final identity = entry['identity']?.toString();
      final group = entry['group']?.toString();
      final tagsRaw = entry['tags'];
      final tags = (tagsRaw is List)
          ? tagsRaw.whereType<String>().toList(growable: false)
          : const <String>[];
      final knownHost = entry['known_host']?.toString();

      if (hostname.isEmpty) {
        warnings.add('Skipped host "$label" ($id): empty hostname');
        continue;
      }

      servers.add(
        TermiusServer(
          termiusId: id,
          label: label,
          hostname: hostname,
          port: port,
          username: username,
          termiusIdentity: (identity != null && identity.isNotEmpty)
              ? identity
              : null,
          group: (group != null && group.isNotEmpty) ? group : null,
          tags: tags,
          knownHostKey: (knownHost != null && knownHost.isNotEmpty)
              ? knownHost
              : null,
        ),
      );
    }

    return TermiusImportResult(
      identities: identities,
      servers: servers,
      warnings: warnings,
    );
  }
}
