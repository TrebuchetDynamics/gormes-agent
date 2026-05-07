import 'package:drift/drift.dart';

part 'key_database.g.dart';

class IdentityRows extends Table {
  @override
  String get tableName => 'identities';

  TextColumn get termiusId => text()();
  TextColumn get label => text()();
  TextColumn get publicKey => text().nullable()();
  TextColumn get privateKeyBlob => text().nullable()();
  BoolColumn get isEncrypted => boolean()();

  @override
  Set<Column<Object>> get primaryKey => {termiusId};
}

class ServerRows extends Table {
  @override
  String get tableName => 'servers';

  TextColumn get dedupKey => text()();
  TextColumn get termiusId => text()();
  TextColumn get label => text()();
  TextColumn get hostname => text()();
  IntColumn get port => integer()();
  TextColumn get username => text()();
  TextColumn get identityTermiusId => text().nullable()();
  TextColumn get groupName => text().nullable()();
  TextColumn get tags => text()(); // JSON-encoded list of strings
  TextColumn get knownHostKey => text().nullable()();

  @override
  Set<Column<Object>> get primaryKey => {dedupKey};
}

@DriftDatabase(tables: [IdentityRows, ServerRows])
class KeyDatabase extends _$KeyDatabase {
  KeyDatabase(QueryExecutor e) : super(e);

  @override
  int get schemaVersion => 1;
}
