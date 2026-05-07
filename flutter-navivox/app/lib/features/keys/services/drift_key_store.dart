import 'dart:convert';

import 'package:drift/drift.dart';

import '../data/key_database.dart';
import 'key_store.dart';

class DriftIdentityStore implements IdentityStore {
  DriftIdentityStore(this._db);
  final KeyDatabase _db;

  @override
  Future<void> upsert(StoredIdentity identity) async {
    await _db.into(_db.identityRows).insertOnConflictUpdate(_toRow(identity));
  }

  @override
  Future<void> delete(String termiusId) async {
    await (_db.delete(_db.identityRows)
          ..where((t) => t.termiusId.equals(termiusId)))
        .go();
  }

  @override
  Future<List<StoredIdentity>> getAll() async {
    final rows = await _db.select(_db.identityRows).get();
    return rows.map(_fromRow).toList(growable: false);
  }

  @override
  Stream<List<StoredIdentity>> watch() {
    return _db.select(_db.identityRows).watch().map(
          (rows) => rows.map(_fromRow).toList(growable: false),
        );
  }

  IdentityRowsCompanion _toRow(StoredIdentity i) {
    return IdentityRowsCompanion(
      termiusId: Value(i.termiusId),
      label: Value(i.label),
      publicKey: Value(i.publicKey),
      privateKeyBlob: Value(i.privateKeyBlob),
      isEncrypted: Value(i.isEncrypted),
    );
  }

  StoredIdentity _fromRow(IdentityRow r) {
    return StoredIdentity(
      termiusId: r.termiusId,
      label: r.label,
      publicKey: r.publicKey,
      privateKeyBlob: r.privateKeyBlob,
      isEncrypted: r.isEncrypted,
    );
  }
}

class DriftServerStore implements ServerStore {
  DriftServerStore(this._db);
  final KeyDatabase _db;

  @override
  Future<void> upsert(StoredServer server) async {
    await _db.into(_db.serverRows).insertOnConflictUpdate(_toRow(server));
  }

  @override
  Future<void> delete(String termiusId) async {
    await (_db.delete(_db.serverRows)
          ..where((t) => t.termiusId.equals(termiusId)))
        .go();
  }

  @override
  Future<List<StoredServer>> getAll() async {
    final rows = await _db.select(_db.serverRows).get();
    return rows.map(_fromRow).toList(growable: false);
  }

  @override
  Stream<List<StoredServer>> watch() {
    return _db.select(_db.serverRows).watch().map(
          (rows) => rows.map(_fromRow).toList(growable: false),
        );
  }

  ServerRowsCompanion _toRow(StoredServer s) {
    return ServerRowsCompanion(
      dedupKey: Value(s.dedupKey),
      termiusId: Value(s.termiusId),
      label: Value(s.label),
      hostname: Value(s.hostname),
      port: Value(s.port),
      username: Value(s.username),
      identityTermiusId: Value(s.identityTermiusId),
      groupName: Value(s.group),
      tags: Value(jsonEncode(s.tags)),
      knownHostKey: Value(s.knownHostKey),
    );
  }

  StoredServer _fromRow(ServerRow r) {
    final decodedTags = jsonDecode(r.tags);
    final tags = (decodedTags is List)
        ? decodedTags.whereType<String>().toList(growable: false)
        : const <String>[];
    return StoredServer(
      termiusId: r.termiusId,
      label: r.label,
      hostname: r.hostname,
      port: r.port,
      username: r.username,
      identityTermiusId: r.identityTermiusId,
      group: r.groupName,
      tags: tags,
      knownHostKey: r.knownHostKey,
    );
  }
}
