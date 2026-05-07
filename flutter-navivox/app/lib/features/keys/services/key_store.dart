import 'dart:async';

import 'termius_importer.dart';

class StoredIdentity {
  const StoredIdentity({
    required this.termiusId,
    required this.label,
    required this.publicKey,
    required this.privateKeyBlob,
    required this.isEncrypted,
  });

  final String termiusId;
  final String label;
  final String? publicKey;
  final String? privateKeyBlob;
  final bool isEncrypted;
}

class StoredServer {
  const StoredServer({
    required this.termiusId,
    required this.label,
    required this.hostname,
    required this.port,
    required this.username,
    this.identityTermiusId,
    this.group,
    this.tags = const [],
    this.knownHostKey,
  });

  final String termiusId;
  final String label;
  final String hostname;
  final int port;
  final String username;
  final String? identityTermiusId;
  final String? group;
  final List<String> tags;
  final String? knownHostKey;

  String get dedupKey =>
      '$hostname|$port|$username|${identityTermiusId ?? ""}';
}

abstract interface class IdentityStore {
  Future<void> upsert(StoredIdentity identity);
  Future<void> delete(String termiusId);
  Future<List<StoredIdentity>> getAll();
  Stream<List<StoredIdentity>> watch();
}

abstract interface class ServerStore {
  Future<void> upsert(StoredServer server);
  Future<void> delete(String termiusId);
  Future<List<StoredServer>> getAll();
  Stream<List<StoredServer>> watch();
}

class InMemoryIdentityStore implements IdentityStore {
  final Map<String, StoredIdentity> _byId = {};
  final StreamController<List<StoredIdentity>> _watch =
      StreamController<List<StoredIdentity>>.broadcast(sync: false);

  @override
  Future<void> upsert(StoredIdentity identity) async {
    _byId[identity.termiusId] = identity;
    _emit();
  }

  @override
  Future<void> delete(String termiusId) async {
    _byId.remove(termiusId);
    _emit();
  }

  @override
  Future<List<StoredIdentity>> getAll() async => List.unmodifiable(_byId.values);

  @override
  Stream<List<StoredIdentity>> watch() {
    // Replay current snapshot to new listeners.
    final controller = StreamController<List<StoredIdentity>>();
    controller.add(List.unmodifiable(_byId.values));
    final sub = _watch.stream.listen(controller.add);
    controller.onCancel = sub.cancel;
    return controller.stream;
  }

  void _emit() => _watch.add(List.unmodifiable(_byId.values));
}

class InMemoryServerStore implements ServerStore {
  final Map<String, StoredServer> _byDedupKey = {};
  final StreamController<List<StoredServer>> _watch =
      StreamController<List<StoredServer>>.broadcast(sync: false);

  @override
  Future<void> upsert(StoredServer server) async {
    _byDedupKey[server.dedupKey] = server;
    _emit();
  }

  @override
  Future<void> delete(String termiusId) async {
    _byDedupKey.removeWhere((_, s) => s.termiusId == termiusId);
    _emit();
  }

  @override
  Future<List<StoredServer>> getAll() async =>
      List.unmodifiable(_byDedupKey.values);

  @override
  Stream<List<StoredServer>> watch() {
    final controller = StreamController<List<StoredServer>>();
    controller.add(List.unmodifiable(_byDedupKey.values));
    final sub = _watch.stream.listen(controller.add);
    controller.onCancel = sub.cancel;
    return controller.stream;
  }

  void _emit() => _watch.add(List.unmodifiable(_byDedupKey.values));
}

class TermiusImportSummary {
  const TermiusImportSummary({
    required this.identitiesImported,
    required this.serversImported,
  });

  final int identitiesImported;
  final int serversImported;
}

class TermiusImportService {
  TermiusImportService({
    required this.identityStore,
    required this.serverStore,
  });

  final IdentityStore identityStore;
  final ServerStore serverStore;

  TermiusImportSummary importParsed(TermiusImportResult parsed) {
    for (final id in parsed.identities) {
      identityStore.upsert(StoredIdentity(
        termiusId: id.termiusId,
        label: id.label,
        publicKey: id.publicKey,
        privateKeyBlob: id.privateKeyPem,
        isEncrypted: id.isEncrypted,
      ));
    }
    for (final s in parsed.servers) {
      serverStore.upsert(StoredServer(
        termiusId: s.termiusId,
        label: s.label,
        hostname: s.hostname,
        port: s.port,
        username: s.username,
        identityTermiusId: s.termiusIdentity,
        group: s.group,
        tags: s.tags,
        knownHostKey: s.knownHostKey,
      ));
    }
    return TermiusImportSummary(
      identitiesImported: parsed.identities.length,
      serversImported: parsed.servers.length,
    );
  }
}
