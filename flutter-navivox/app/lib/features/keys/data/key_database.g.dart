// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'key_database.dart';

// ignore_for_file: type=lint
class $IdentityRowsTable extends IdentityRows
    with TableInfo<$IdentityRowsTable, IdentityRow> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $IdentityRowsTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _termiusIdMeta = const VerificationMeta(
    'termiusId',
  );
  @override
  late final GeneratedColumn<String> termiusId = GeneratedColumn<String>(
    'termius_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _labelMeta = const VerificationMeta('label');
  @override
  late final GeneratedColumn<String> label = GeneratedColumn<String>(
    'label',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _publicKeyMeta = const VerificationMeta(
    'publicKey',
  );
  @override
  late final GeneratedColumn<String> publicKey = GeneratedColumn<String>(
    'public_key',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  static const VerificationMeta _privateKeyBlobMeta = const VerificationMeta(
    'privateKeyBlob',
  );
  @override
  late final GeneratedColumn<String> privateKeyBlob = GeneratedColumn<String>(
    'private_key_blob',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  static const VerificationMeta _isEncryptedMeta = const VerificationMeta(
    'isEncrypted',
  );
  @override
  late final GeneratedColumn<bool> isEncrypted = GeneratedColumn<bool>(
    'is_encrypted',
    aliasedName,
    false,
    type: DriftSqlType.bool,
    requiredDuringInsert: true,
    defaultConstraints: GeneratedColumn.constraintIsAlways(
      'CHECK ("is_encrypted" IN (0, 1))',
    ),
  );
  @override
  List<GeneratedColumn> get $columns => [
    termiusId,
    label,
    publicKey,
    privateKeyBlob,
    isEncrypted,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'identities';
  @override
  VerificationContext validateIntegrity(
    Insertable<IdentityRow> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('termius_id')) {
      context.handle(
        _termiusIdMeta,
        termiusId.isAcceptableOrUnknown(data['termius_id']!, _termiusIdMeta),
      );
    } else if (isInserting) {
      context.missing(_termiusIdMeta);
    }
    if (data.containsKey('label')) {
      context.handle(
        _labelMeta,
        label.isAcceptableOrUnknown(data['label']!, _labelMeta),
      );
    } else if (isInserting) {
      context.missing(_labelMeta);
    }
    if (data.containsKey('public_key')) {
      context.handle(
        _publicKeyMeta,
        publicKey.isAcceptableOrUnknown(data['public_key']!, _publicKeyMeta),
      );
    }
    if (data.containsKey('private_key_blob')) {
      context.handle(
        _privateKeyBlobMeta,
        privateKeyBlob.isAcceptableOrUnknown(
          data['private_key_blob']!,
          _privateKeyBlobMeta,
        ),
      );
    }
    if (data.containsKey('is_encrypted')) {
      context.handle(
        _isEncryptedMeta,
        isEncrypted.isAcceptableOrUnknown(
          data['is_encrypted']!,
          _isEncryptedMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_isEncryptedMeta);
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {termiusId};
  @override
  IdentityRow map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return IdentityRow(
      termiusId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}termius_id'],
      )!,
      label: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}label'],
      )!,
      publicKey: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}public_key'],
      ),
      privateKeyBlob: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}private_key_blob'],
      ),
      isEncrypted: attachedDatabase.typeMapping.read(
        DriftSqlType.bool,
        data['${effectivePrefix}is_encrypted'],
      )!,
    );
  }

  @override
  $IdentityRowsTable createAlias(String alias) {
    return $IdentityRowsTable(attachedDatabase, alias);
  }
}

class IdentityRow extends DataClass implements Insertable<IdentityRow> {
  final String termiusId;
  final String label;
  final String? publicKey;
  final String? privateKeyBlob;
  final bool isEncrypted;
  const IdentityRow({
    required this.termiusId,
    required this.label,
    this.publicKey,
    this.privateKeyBlob,
    required this.isEncrypted,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['termius_id'] = Variable<String>(termiusId);
    map['label'] = Variable<String>(label);
    if (!nullToAbsent || publicKey != null) {
      map['public_key'] = Variable<String>(publicKey);
    }
    if (!nullToAbsent || privateKeyBlob != null) {
      map['private_key_blob'] = Variable<String>(privateKeyBlob);
    }
    map['is_encrypted'] = Variable<bool>(isEncrypted);
    return map;
  }

  IdentityRowsCompanion toCompanion(bool nullToAbsent) {
    return IdentityRowsCompanion(
      termiusId: Value(termiusId),
      label: Value(label),
      publicKey: publicKey == null && nullToAbsent
          ? const Value.absent()
          : Value(publicKey),
      privateKeyBlob: privateKeyBlob == null && nullToAbsent
          ? const Value.absent()
          : Value(privateKeyBlob),
      isEncrypted: Value(isEncrypted),
    );
  }

  factory IdentityRow.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return IdentityRow(
      termiusId: serializer.fromJson<String>(json['termiusId']),
      label: serializer.fromJson<String>(json['label']),
      publicKey: serializer.fromJson<String?>(json['publicKey']),
      privateKeyBlob: serializer.fromJson<String?>(json['privateKeyBlob']),
      isEncrypted: serializer.fromJson<bool>(json['isEncrypted']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'termiusId': serializer.toJson<String>(termiusId),
      'label': serializer.toJson<String>(label),
      'publicKey': serializer.toJson<String?>(publicKey),
      'privateKeyBlob': serializer.toJson<String?>(privateKeyBlob),
      'isEncrypted': serializer.toJson<bool>(isEncrypted),
    };
  }

  IdentityRow copyWith({
    String? termiusId,
    String? label,
    Value<String?> publicKey = const Value.absent(),
    Value<String?> privateKeyBlob = const Value.absent(),
    bool? isEncrypted,
  }) => IdentityRow(
    termiusId: termiusId ?? this.termiusId,
    label: label ?? this.label,
    publicKey: publicKey.present ? publicKey.value : this.publicKey,
    privateKeyBlob: privateKeyBlob.present
        ? privateKeyBlob.value
        : this.privateKeyBlob,
    isEncrypted: isEncrypted ?? this.isEncrypted,
  );
  IdentityRow copyWithCompanion(IdentityRowsCompanion data) {
    return IdentityRow(
      termiusId: data.termiusId.present ? data.termiusId.value : this.termiusId,
      label: data.label.present ? data.label.value : this.label,
      publicKey: data.publicKey.present ? data.publicKey.value : this.publicKey,
      privateKeyBlob: data.privateKeyBlob.present
          ? data.privateKeyBlob.value
          : this.privateKeyBlob,
      isEncrypted: data.isEncrypted.present
          ? data.isEncrypted.value
          : this.isEncrypted,
    );
  }

  @override
  String toString() {
    return (StringBuffer('IdentityRow(')
          ..write('termiusId: $termiusId, ')
          ..write('label: $label, ')
          ..write('publicKey: $publicKey, ')
          ..write('privateKeyBlob: $privateKeyBlob, ')
          ..write('isEncrypted: $isEncrypted')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode =>
      Object.hash(termiusId, label, publicKey, privateKeyBlob, isEncrypted);
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is IdentityRow &&
          other.termiusId == this.termiusId &&
          other.label == this.label &&
          other.publicKey == this.publicKey &&
          other.privateKeyBlob == this.privateKeyBlob &&
          other.isEncrypted == this.isEncrypted);
}

class IdentityRowsCompanion extends UpdateCompanion<IdentityRow> {
  final Value<String> termiusId;
  final Value<String> label;
  final Value<String?> publicKey;
  final Value<String?> privateKeyBlob;
  final Value<bool> isEncrypted;
  final Value<int> rowid;
  const IdentityRowsCompanion({
    this.termiusId = const Value.absent(),
    this.label = const Value.absent(),
    this.publicKey = const Value.absent(),
    this.privateKeyBlob = const Value.absent(),
    this.isEncrypted = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  IdentityRowsCompanion.insert({
    required String termiusId,
    required String label,
    this.publicKey = const Value.absent(),
    this.privateKeyBlob = const Value.absent(),
    required bool isEncrypted,
    this.rowid = const Value.absent(),
  }) : termiusId = Value(termiusId),
       label = Value(label),
       isEncrypted = Value(isEncrypted);
  static Insertable<IdentityRow> custom({
    Expression<String>? termiusId,
    Expression<String>? label,
    Expression<String>? publicKey,
    Expression<String>? privateKeyBlob,
    Expression<bool>? isEncrypted,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (termiusId != null) 'termius_id': termiusId,
      if (label != null) 'label': label,
      if (publicKey != null) 'public_key': publicKey,
      if (privateKeyBlob != null) 'private_key_blob': privateKeyBlob,
      if (isEncrypted != null) 'is_encrypted': isEncrypted,
      if (rowid != null) 'rowid': rowid,
    });
  }

  IdentityRowsCompanion copyWith({
    Value<String>? termiusId,
    Value<String>? label,
    Value<String?>? publicKey,
    Value<String?>? privateKeyBlob,
    Value<bool>? isEncrypted,
    Value<int>? rowid,
  }) {
    return IdentityRowsCompanion(
      termiusId: termiusId ?? this.termiusId,
      label: label ?? this.label,
      publicKey: publicKey ?? this.publicKey,
      privateKeyBlob: privateKeyBlob ?? this.privateKeyBlob,
      isEncrypted: isEncrypted ?? this.isEncrypted,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (termiusId.present) {
      map['termius_id'] = Variable<String>(termiusId.value);
    }
    if (label.present) {
      map['label'] = Variable<String>(label.value);
    }
    if (publicKey.present) {
      map['public_key'] = Variable<String>(publicKey.value);
    }
    if (privateKeyBlob.present) {
      map['private_key_blob'] = Variable<String>(privateKeyBlob.value);
    }
    if (isEncrypted.present) {
      map['is_encrypted'] = Variable<bool>(isEncrypted.value);
    }
    if (rowid.present) {
      map['rowid'] = Variable<int>(rowid.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('IdentityRowsCompanion(')
          ..write('termiusId: $termiusId, ')
          ..write('label: $label, ')
          ..write('publicKey: $publicKey, ')
          ..write('privateKeyBlob: $privateKeyBlob, ')
          ..write('isEncrypted: $isEncrypted, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

class $ServerRowsTable extends ServerRows
    with TableInfo<$ServerRowsTable, ServerRow> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $ServerRowsTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _dedupKeyMeta = const VerificationMeta(
    'dedupKey',
  );
  @override
  late final GeneratedColumn<String> dedupKey = GeneratedColumn<String>(
    'dedup_key',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _termiusIdMeta = const VerificationMeta(
    'termiusId',
  );
  @override
  late final GeneratedColumn<String> termiusId = GeneratedColumn<String>(
    'termius_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _labelMeta = const VerificationMeta('label');
  @override
  late final GeneratedColumn<String> label = GeneratedColumn<String>(
    'label',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _hostnameMeta = const VerificationMeta(
    'hostname',
  );
  @override
  late final GeneratedColumn<String> hostname = GeneratedColumn<String>(
    'hostname',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _portMeta = const VerificationMeta('port');
  @override
  late final GeneratedColumn<int> port = GeneratedColumn<int>(
    'port',
    aliasedName,
    false,
    type: DriftSqlType.int,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _usernameMeta = const VerificationMeta(
    'username',
  );
  @override
  late final GeneratedColumn<String> username = GeneratedColumn<String>(
    'username',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _identityTermiusIdMeta = const VerificationMeta(
    'identityTermiusId',
  );
  @override
  late final GeneratedColumn<String> identityTermiusId =
      GeneratedColumn<String>(
        'identity_termius_id',
        aliasedName,
        true,
        type: DriftSqlType.string,
        requiredDuringInsert: false,
      );
  static const VerificationMeta _groupNameMeta = const VerificationMeta(
    'groupName',
  );
  @override
  late final GeneratedColumn<String> groupName = GeneratedColumn<String>(
    'group_name',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  static const VerificationMeta _tagsMeta = const VerificationMeta('tags');
  @override
  late final GeneratedColumn<String> tags = GeneratedColumn<String>(
    'tags',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _knownHostKeyMeta = const VerificationMeta(
    'knownHostKey',
  );
  @override
  late final GeneratedColumn<String> knownHostKey = GeneratedColumn<String>(
    'known_host_key',
    aliasedName,
    true,
    type: DriftSqlType.string,
    requiredDuringInsert: false,
  );
  @override
  List<GeneratedColumn> get $columns => [
    dedupKey,
    termiusId,
    label,
    hostname,
    port,
    username,
    identityTermiusId,
    groupName,
    tags,
    knownHostKey,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'servers';
  @override
  VerificationContext validateIntegrity(
    Insertable<ServerRow> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('dedup_key')) {
      context.handle(
        _dedupKeyMeta,
        dedupKey.isAcceptableOrUnknown(data['dedup_key']!, _dedupKeyMeta),
      );
    } else if (isInserting) {
      context.missing(_dedupKeyMeta);
    }
    if (data.containsKey('termius_id')) {
      context.handle(
        _termiusIdMeta,
        termiusId.isAcceptableOrUnknown(data['termius_id']!, _termiusIdMeta),
      );
    } else if (isInserting) {
      context.missing(_termiusIdMeta);
    }
    if (data.containsKey('label')) {
      context.handle(
        _labelMeta,
        label.isAcceptableOrUnknown(data['label']!, _labelMeta),
      );
    } else if (isInserting) {
      context.missing(_labelMeta);
    }
    if (data.containsKey('hostname')) {
      context.handle(
        _hostnameMeta,
        hostname.isAcceptableOrUnknown(data['hostname']!, _hostnameMeta),
      );
    } else if (isInserting) {
      context.missing(_hostnameMeta);
    }
    if (data.containsKey('port')) {
      context.handle(
        _portMeta,
        port.isAcceptableOrUnknown(data['port']!, _portMeta),
      );
    } else if (isInserting) {
      context.missing(_portMeta);
    }
    if (data.containsKey('username')) {
      context.handle(
        _usernameMeta,
        username.isAcceptableOrUnknown(data['username']!, _usernameMeta),
      );
    } else if (isInserting) {
      context.missing(_usernameMeta);
    }
    if (data.containsKey('identity_termius_id')) {
      context.handle(
        _identityTermiusIdMeta,
        identityTermiusId.isAcceptableOrUnknown(
          data['identity_termius_id']!,
          _identityTermiusIdMeta,
        ),
      );
    }
    if (data.containsKey('group_name')) {
      context.handle(
        _groupNameMeta,
        groupName.isAcceptableOrUnknown(data['group_name']!, _groupNameMeta),
      );
    }
    if (data.containsKey('tags')) {
      context.handle(
        _tagsMeta,
        tags.isAcceptableOrUnknown(data['tags']!, _tagsMeta),
      );
    } else if (isInserting) {
      context.missing(_tagsMeta);
    }
    if (data.containsKey('known_host_key')) {
      context.handle(
        _knownHostKeyMeta,
        knownHostKey.isAcceptableOrUnknown(
          data['known_host_key']!,
          _knownHostKeyMeta,
        ),
      );
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {dedupKey};
  @override
  ServerRow map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return ServerRow(
      dedupKey: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}dedup_key'],
      )!,
      termiusId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}termius_id'],
      )!,
      label: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}label'],
      )!,
      hostname: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}hostname'],
      )!,
      port: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}port'],
      )!,
      username: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}username'],
      )!,
      identityTermiusId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}identity_termius_id'],
      ),
      groupName: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}group_name'],
      ),
      tags: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}tags'],
      )!,
      knownHostKey: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}known_host_key'],
      ),
    );
  }

  @override
  $ServerRowsTable createAlias(String alias) {
    return $ServerRowsTable(attachedDatabase, alias);
  }
}

class ServerRow extends DataClass implements Insertable<ServerRow> {
  final String dedupKey;
  final String termiusId;
  final String label;
  final String hostname;
  final int port;
  final String username;
  final String? identityTermiusId;
  final String? groupName;
  final String tags;
  final String? knownHostKey;
  const ServerRow({
    required this.dedupKey,
    required this.termiusId,
    required this.label,
    required this.hostname,
    required this.port,
    required this.username,
    this.identityTermiusId,
    this.groupName,
    required this.tags,
    this.knownHostKey,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['dedup_key'] = Variable<String>(dedupKey);
    map['termius_id'] = Variable<String>(termiusId);
    map['label'] = Variable<String>(label);
    map['hostname'] = Variable<String>(hostname);
    map['port'] = Variable<int>(port);
    map['username'] = Variable<String>(username);
    if (!nullToAbsent || identityTermiusId != null) {
      map['identity_termius_id'] = Variable<String>(identityTermiusId);
    }
    if (!nullToAbsent || groupName != null) {
      map['group_name'] = Variable<String>(groupName);
    }
    map['tags'] = Variable<String>(tags);
    if (!nullToAbsent || knownHostKey != null) {
      map['known_host_key'] = Variable<String>(knownHostKey);
    }
    return map;
  }

  ServerRowsCompanion toCompanion(bool nullToAbsent) {
    return ServerRowsCompanion(
      dedupKey: Value(dedupKey),
      termiusId: Value(termiusId),
      label: Value(label),
      hostname: Value(hostname),
      port: Value(port),
      username: Value(username),
      identityTermiusId: identityTermiusId == null && nullToAbsent
          ? const Value.absent()
          : Value(identityTermiusId),
      groupName: groupName == null && nullToAbsent
          ? const Value.absent()
          : Value(groupName),
      tags: Value(tags),
      knownHostKey: knownHostKey == null && nullToAbsent
          ? const Value.absent()
          : Value(knownHostKey),
    );
  }

  factory ServerRow.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return ServerRow(
      dedupKey: serializer.fromJson<String>(json['dedupKey']),
      termiusId: serializer.fromJson<String>(json['termiusId']),
      label: serializer.fromJson<String>(json['label']),
      hostname: serializer.fromJson<String>(json['hostname']),
      port: serializer.fromJson<int>(json['port']),
      username: serializer.fromJson<String>(json['username']),
      identityTermiusId: serializer.fromJson<String?>(
        json['identityTermiusId'],
      ),
      groupName: serializer.fromJson<String?>(json['groupName']),
      tags: serializer.fromJson<String>(json['tags']),
      knownHostKey: serializer.fromJson<String?>(json['knownHostKey']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'dedupKey': serializer.toJson<String>(dedupKey),
      'termiusId': serializer.toJson<String>(termiusId),
      'label': serializer.toJson<String>(label),
      'hostname': serializer.toJson<String>(hostname),
      'port': serializer.toJson<int>(port),
      'username': serializer.toJson<String>(username),
      'identityTermiusId': serializer.toJson<String?>(identityTermiusId),
      'groupName': serializer.toJson<String?>(groupName),
      'tags': serializer.toJson<String>(tags),
      'knownHostKey': serializer.toJson<String?>(knownHostKey),
    };
  }

  ServerRow copyWith({
    String? dedupKey,
    String? termiusId,
    String? label,
    String? hostname,
    int? port,
    String? username,
    Value<String?> identityTermiusId = const Value.absent(),
    Value<String?> groupName = const Value.absent(),
    String? tags,
    Value<String?> knownHostKey = const Value.absent(),
  }) => ServerRow(
    dedupKey: dedupKey ?? this.dedupKey,
    termiusId: termiusId ?? this.termiusId,
    label: label ?? this.label,
    hostname: hostname ?? this.hostname,
    port: port ?? this.port,
    username: username ?? this.username,
    identityTermiusId: identityTermiusId.present
        ? identityTermiusId.value
        : this.identityTermiusId,
    groupName: groupName.present ? groupName.value : this.groupName,
    tags: tags ?? this.tags,
    knownHostKey: knownHostKey.present ? knownHostKey.value : this.knownHostKey,
  );
  ServerRow copyWithCompanion(ServerRowsCompanion data) {
    return ServerRow(
      dedupKey: data.dedupKey.present ? data.dedupKey.value : this.dedupKey,
      termiusId: data.termiusId.present ? data.termiusId.value : this.termiusId,
      label: data.label.present ? data.label.value : this.label,
      hostname: data.hostname.present ? data.hostname.value : this.hostname,
      port: data.port.present ? data.port.value : this.port,
      username: data.username.present ? data.username.value : this.username,
      identityTermiusId: data.identityTermiusId.present
          ? data.identityTermiusId.value
          : this.identityTermiusId,
      groupName: data.groupName.present ? data.groupName.value : this.groupName,
      tags: data.tags.present ? data.tags.value : this.tags,
      knownHostKey: data.knownHostKey.present
          ? data.knownHostKey.value
          : this.knownHostKey,
    );
  }

  @override
  String toString() {
    return (StringBuffer('ServerRow(')
          ..write('dedupKey: $dedupKey, ')
          ..write('termiusId: $termiusId, ')
          ..write('label: $label, ')
          ..write('hostname: $hostname, ')
          ..write('port: $port, ')
          ..write('username: $username, ')
          ..write('identityTermiusId: $identityTermiusId, ')
          ..write('groupName: $groupName, ')
          ..write('tags: $tags, ')
          ..write('knownHostKey: $knownHostKey')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(
    dedupKey,
    termiusId,
    label,
    hostname,
    port,
    username,
    identityTermiusId,
    groupName,
    tags,
    knownHostKey,
  );
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is ServerRow &&
          other.dedupKey == this.dedupKey &&
          other.termiusId == this.termiusId &&
          other.label == this.label &&
          other.hostname == this.hostname &&
          other.port == this.port &&
          other.username == this.username &&
          other.identityTermiusId == this.identityTermiusId &&
          other.groupName == this.groupName &&
          other.tags == this.tags &&
          other.knownHostKey == this.knownHostKey);
}

class ServerRowsCompanion extends UpdateCompanion<ServerRow> {
  final Value<String> dedupKey;
  final Value<String> termiusId;
  final Value<String> label;
  final Value<String> hostname;
  final Value<int> port;
  final Value<String> username;
  final Value<String?> identityTermiusId;
  final Value<String?> groupName;
  final Value<String> tags;
  final Value<String?> knownHostKey;
  final Value<int> rowid;
  const ServerRowsCompanion({
    this.dedupKey = const Value.absent(),
    this.termiusId = const Value.absent(),
    this.label = const Value.absent(),
    this.hostname = const Value.absent(),
    this.port = const Value.absent(),
    this.username = const Value.absent(),
    this.identityTermiusId = const Value.absent(),
    this.groupName = const Value.absent(),
    this.tags = const Value.absent(),
    this.knownHostKey = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  ServerRowsCompanion.insert({
    required String dedupKey,
    required String termiusId,
    required String label,
    required String hostname,
    required int port,
    required String username,
    this.identityTermiusId = const Value.absent(),
    this.groupName = const Value.absent(),
    required String tags,
    this.knownHostKey = const Value.absent(),
    this.rowid = const Value.absent(),
  }) : dedupKey = Value(dedupKey),
       termiusId = Value(termiusId),
       label = Value(label),
       hostname = Value(hostname),
       port = Value(port),
       username = Value(username),
       tags = Value(tags);
  static Insertable<ServerRow> custom({
    Expression<String>? dedupKey,
    Expression<String>? termiusId,
    Expression<String>? label,
    Expression<String>? hostname,
    Expression<int>? port,
    Expression<String>? username,
    Expression<String>? identityTermiusId,
    Expression<String>? groupName,
    Expression<String>? tags,
    Expression<String>? knownHostKey,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (dedupKey != null) 'dedup_key': dedupKey,
      if (termiusId != null) 'termius_id': termiusId,
      if (label != null) 'label': label,
      if (hostname != null) 'hostname': hostname,
      if (port != null) 'port': port,
      if (username != null) 'username': username,
      if (identityTermiusId != null) 'identity_termius_id': identityTermiusId,
      if (groupName != null) 'group_name': groupName,
      if (tags != null) 'tags': tags,
      if (knownHostKey != null) 'known_host_key': knownHostKey,
      if (rowid != null) 'rowid': rowid,
    });
  }

  ServerRowsCompanion copyWith({
    Value<String>? dedupKey,
    Value<String>? termiusId,
    Value<String>? label,
    Value<String>? hostname,
    Value<int>? port,
    Value<String>? username,
    Value<String?>? identityTermiusId,
    Value<String?>? groupName,
    Value<String>? tags,
    Value<String?>? knownHostKey,
    Value<int>? rowid,
  }) {
    return ServerRowsCompanion(
      dedupKey: dedupKey ?? this.dedupKey,
      termiusId: termiusId ?? this.termiusId,
      label: label ?? this.label,
      hostname: hostname ?? this.hostname,
      port: port ?? this.port,
      username: username ?? this.username,
      identityTermiusId: identityTermiusId ?? this.identityTermiusId,
      groupName: groupName ?? this.groupName,
      tags: tags ?? this.tags,
      knownHostKey: knownHostKey ?? this.knownHostKey,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (dedupKey.present) {
      map['dedup_key'] = Variable<String>(dedupKey.value);
    }
    if (termiusId.present) {
      map['termius_id'] = Variable<String>(termiusId.value);
    }
    if (label.present) {
      map['label'] = Variable<String>(label.value);
    }
    if (hostname.present) {
      map['hostname'] = Variable<String>(hostname.value);
    }
    if (port.present) {
      map['port'] = Variable<int>(port.value);
    }
    if (username.present) {
      map['username'] = Variable<String>(username.value);
    }
    if (identityTermiusId.present) {
      map['identity_termius_id'] = Variable<String>(identityTermiusId.value);
    }
    if (groupName.present) {
      map['group_name'] = Variable<String>(groupName.value);
    }
    if (tags.present) {
      map['tags'] = Variable<String>(tags.value);
    }
    if (knownHostKey.present) {
      map['known_host_key'] = Variable<String>(knownHostKey.value);
    }
    if (rowid.present) {
      map['rowid'] = Variable<int>(rowid.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('ServerRowsCompanion(')
          ..write('dedupKey: $dedupKey, ')
          ..write('termiusId: $termiusId, ')
          ..write('label: $label, ')
          ..write('hostname: $hostname, ')
          ..write('port: $port, ')
          ..write('username: $username, ')
          ..write('identityTermiusId: $identityTermiusId, ')
          ..write('groupName: $groupName, ')
          ..write('tags: $tags, ')
          ..write('knownHostKey: $knownHostKey, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

abstract class _$KeyDatabase extends GeneratedDatabase {
  _$KeyDatabase(QueryExecutor e) : super(e);
  $KeyDatabaseManager get managers => $KeyDatabaseManager(this);
  late final $IdentityRowsTable identityRows = $IdentityRowsTable(this);
  late final $ServerRowsTable serverRows = $ServerRowsTable(this);
  @override
  Iterable<TableInfo<Table, Object?>> get allTables =>
      allSchemaEntities.whereType<TableInfo<Table, Object?>>();
  @override
  List<DatabaseSchemaEntity> get allSchemaEntities => [
    identityRows,
    serverRows,
  ];
}

typedef $$IdentityRowsTableCreateCompanionBuilder =
    IdentityRowsCompanion Function({
      required String termiusId,
      required String label,
      Value<String?> publicKey,
      Value<String?> privateKeyBlob,
      required bool isEncrypted,
      Value<int> rowid,
    });
typedef $$IdentityRowsTableUpdateCompanionBuilder =
    IdentityRowsCompanion Function({
      Value<String> termiusId,
      Value<String> label,
      Value<String?> publicKey,
      Value<String?> privateKeyBlob,
      Value<bool> isEncrypted,
      Value<int> rowid,
    });

class $$IdentityRowsTableFilterComposer
    extends Composer<_$KeyDatabase, $IdentityRowsTable> {
  $$IdentityRowsTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get termiusId => $composableBuilder(
    column: $table.termiusId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get label => $composableBuilder(
    column: $table.label,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get publicKey => $composableBuilder(
    column: $table.publicKey,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get privateKeyBlob => $composableBuilder(
    column: $table.privateKeyBlob,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<bool> get isEncrypted => $composableBuilder(
    column: $table.isEncrypted,
    builder: (column) => ColumnFilters(column),
  );
}

class $$IdentityRowsTableOrderingComposer
    extends Composer<_$KeyDatabase, $IdentityRowsTable> {
  $$IdentityRowsTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get termiusId => $composableBuilder(
    column: $table.termiusId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get label => $composableBuilder(
    column: $table.label,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get publicKey => $composableBuilder(
    column: $table.publicKey,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get privateKeyBlob => $composableBuilder(
    column: $table.privateKeyBlob,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<bool> get isEncrypted => $composableBuilder(
    column: $table.isEncrypted,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$IdentityRowsTableAnnotationComposer
    extends Composer<_$KeyDatabase, $IdentityRowsTable> {
  $$IdentityRowsTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get termiusId =>
      $composableBuilder(column: $table.termiusId, builder: (column) => column);

  GeneratedColumn<String> get label =>
      $composableBuilder(column: $table.label, builder: (column) => column);

  GeneratedColumn<String> get publicKey =>
      $composableBuilder(column: $table.publicKey, builder: (column) => column);

  GeneratedColumn<String> get privateKeyBlob => $composableBuilder(
    column: $table.privateKeyBlob,
    builder: (column) => column,
  );

  GeneratedColumn<bool> get isEncrypted => $composableBuilder(
    column: $table.isEncrypted,
    builder: (column) => column,
  );
}

class $$IdentityRowsTableTableManager
    extends
        RootTableManager<
          _$KeyDatabase,
          $IdentityRowsTable,
          IdentityRow,
          $$IdentityRowsTableFilterComposer,
          $$IdentityRowsTableOrderingComposer,
          $$IdentityRowsTableAnnotationComposer,
          $$IdentityRowsTableCreateCompanionBuilder,
          $$IdentityRowsTableUpdateCompanionBuilder,
          (
            IdentityRow,
            BaseReferences<_$KeyDatabase, $IdentityRowsTable, IdentityRow>,
          ),
          IdentityRow,
          PrefetchHooks Function()
        > {
  $$IdentityRowsTableTableManager(_$KeyDatabase db, $IdentityRowsTable table)
    : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$IdentityRowsTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$IdentityRowsTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer: () =>
              $$IdentityRowsTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<String> termiusId = const Value.absent(),
                Value<String> label = const Value.absent(),
                Value<String?> publicKey = const Value.absent(),
                Value<String?> privateKeyBlob = const Value.absent(),
                Value<bool> isEncrypted = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => IdentityRowsCompanion(
                termiusId: termiusId,
                label: label,
                publicKey: publicKey,
                privateKeyBlob: privateKeyBlob,
                isEncrypted: isEncrypted,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String termiusId,
                required String label,
                Value<String?> publicKey = const Value.absent(),
                Value<String?> privateKeyBlob = const Value.absent(),
                required bool isEncrypted,
                Value<int> rowid = const Value.absent(),
              }) => IdentityRowsCompanion.insert(
                termiusId: termiusId,
                label: label,
                publicKey: publicKey,
                privateKeyBlob: privateKeyBlob,
                isEncrypted: isEncrypted,
                rowid: rowid,
              ),
          withReferenceMapper: (p0) => p0
              .map((e) => (e.readTable(table), BaseReferences(db, table, e)))
              .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$IdentityRowsTableProcessedTableManager =
    ProcessedTableManager<
      _$KeyDatabase,
      $IdentityRowsTable,
      IdentityRow,
      $$IdentityRowsTableFilterComposer,
      $$IdentityRowsTableOrderingComposer,
      $$IdentityRowsTableAnnotationComposer,
      $$IdentityRowsTableCreateCompanionBuilder,
      $$IdentityRowsTableUpdateCompanionBuilder,
      (
        IdentityRow,
        BaseReferences<_$KeyDatabase, $IdentityRowsTable, IdentityRow>,
      ),
      IdentityRow,
      PrefetchHooks Function()
    >;
typedef $$ServerRowsTableCreateCompanionBuilder =
    ServerRowsCompanion Function({
      required String dedupKey,
      required String termiusId,
      required String label,
      required String hostname,
      required int port,
      required String username,
      Value<String?> identityTermiusId,
      Value<String?> groupName,
      required String tags,
      Value<String?> knownHostKey,
      Value<int> rowid,
    });
typedef $$ServerRowsTableUpdateCompanionBuilder =
    ServerRowsCompanion Function({
      Value<String> dedupKey,
      Value<String> termiusId,
      Value<String> label,
      Value<String> hostname,
      Value<int> port,
      Value<String> username,
      Value<String?> identityTermiusId,
      Value<String?> groupName,
      Value<String> tags,
      Value<String?> knownHostKey,
      Value<int> rowid,
    });

class $$ServerRowsTableFilterComposer
    extends Composer<_$KeyDatabase, $ServerRowsTable> {
  $$ServerRowsTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get dedupKey => $composableBuilder(
    column: $table.dedupKey,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get termiusId => $composableBuilder(
    column: $table.termiusId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get label => $composableBuilder(
    column: $table.label,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get hostname => $composableBuilder(
    column: $table.hostname,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<int> get port => $composableBuilder(
    column: $table.port,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get username => $composableBuilder(
    column: $table.username,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get identityTermiusId => $composableBuilder(
    column: $table.identityTermiusId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get groupName => $composableBuilder(
    column: $table.groupName,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get tags => $composableBuilder(
    column: $table.tags,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<String> get knownHostKey => $composableBuilder(
    column: $table.knownHostKey,
    builder: (column) => ColumnFilters(column),
  );
}

class $$ServerRowsTableOrderingComposer
    extends Composer<_$KeyDatabase, $ServerRowsTable> {
  $$ServerRowsTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get dedupKey => $composableBuilder(
    column: $table.dedupKey,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get termiusId => $composableBuilder(
    column: $table.termiusId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get label => $composableBuilder(
    column: $table.label,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get hostname => $composableBuilder(
    column: $table.hostname,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<int> get port => $composableBuilder(
    column: $table.port,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get username => $composableBuilder(
    column: $table.username,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get identityTermiusId => $composableBuilder(
    column: $table.identityTermiusId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get groupName => $composableBuilder(
    column: $table.groupName,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get tags => $composableBuilder(
    column: $table.tags,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<String> get knownHostKey => $composableBuilder(
    column: $table.knownHostKey,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$ServerRowsTableAnnotationComposer
    extends Composer<_$KeyDatabase, $ServerRowsTable> {
  $$ServerRowsTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get dedupKey =>
      $composableBuilder(column: $table.dedupKey, builder: (column) => column);

  GeneratedColumn<String> get termiusId =>
      $composableBuilder(column: $table.termiusId, builder: (column) => column);

  GeneratedColumn<String> get label =>
      $composableBuilder(column: $table.label, builder: (column) => column);

  GeneratedColumn<String> get hostname =>
      $composableBuilder(column: $table.hostname, builder: (column) => column);

  GeneratedColumn<int> get port =>
      $composableBuilder(column: $table.port, builder: (column) => column);

  GeneratedColumn<String> get username =>
      $composableBuilder(column: $table.username, builder: (column) => column);

  GeneratedColumn<String> get identityTermiusId => $composableBuilder(
    column: $table.identityTermiusId,
    builder: (column) => column,
  );

  GeneratedColumn<String> get groupName =>
      $composableBuilder(column: $table.groupName, builder: (column) => column);

  GeneratedColumn<String> get tags =>
      $composableBuilder(column: $table.tags, builder: (column) => column);

  GeneratedColumn<String> get knownHostKey => $composableBuilder(
    column: $table.knownHostKey,
    builder: (column) => column,
  );
}

class $$ServerRowsTableTableManager
    extends
        RootTableManager<
          _$KeyDatabase,
          $ServerRowsTable,
          ServerRow,
          $$ServerRowsTableFilterComposer,
          $$ServerRowsTableOrderingComposer,
          $$ServerRowsTableAnnotationComposer,
          $$ServerRowsTableCreateCompanionBuilder,
          $$ServerRowsTableUpdateCompanionBuilder,
          (
            ServerRow,
            BaseReferences<_$KeyDatabase, $ServerRowsTable, ServerRow>,
          ),
          ServerRow,
          PrefetchHooks Function()
        > {
  $$ServerRowsTableTableManager(_$KeyDatabase db, $ServerRowsTable table)
    : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$ServerRowsTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$ServerRowsTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer: () =>
              $$ServerRowsTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<String> dedupKey = const Value.absent(),
                Value<String> termiusId = const Value.absent(),
                Value<String> label = const Value.absent(),
                Value<String> hostname = const Value.absent(),
                Value<int> port = const Value.absent(),
                Value<String> username = const Value.absent(),
                Value<String?> identityTermiusId = const Value.absent(),
                Value<String?> groupName = const Value.absent(),
                Value<String> tags = const Value.absent(),
                Value<String?> knownHostKey = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => ServerRowsCompanion(
                dedupKey: dedupKey,
                termiusId: termiusId,
                label: label,
                hostname: hostname,
                port: port,
                username: username,
                identityTermiusId: identityTermiusId,
                groupName: groupName,
                tags: tags,
                knownHostKey: knownHostKey,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String dedupKey,
                required String termiusId,
                required String label,
                required String hostname,
                required int port,
                required String username,
                Value<String?> identityTermiusId = const Value.absent(),
                Value<String?> groupName = const Value.absent(),
                required String tags,
                Value<String?> knownHostKey = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => ServerRowsCompanion.insert(
                dedupKey: dedupKey,
                termiusId: termiusId,
                label: label,
                hostname: hostname,
                port: port,
                username: username,
                identityTermiusId: identityTermiusId,
                groupName: groupName,
                tags: tags,
                knownHostKey: knownHostKey,
                rowid: rowid,
              ),
          withReferenceMapper: (p0) => p0
              .map((e) => (e.readTable(table), BaseReferences(db, table, e)))
              .toList(),
          prefetchHooksCallback: null,
        ),
      );
}

typedef $$ServerRowsTableProcessedTableManager =
    ProcessedTableManager<
      _$KeyDatabase,
      $ServerRowsTable,
      ServerRow,
      $$ServerRowsTableFilterComposer,
      $$ServerRowsTableOrderingComposer,
      $$ServerRowsTableAnnotationComposer,
      $$ServerRowsTableCreateCompanionBuilder,
      $$ServerRowsTableUpdateCompanionBuilder,
      (ServerRow, BaseReferences<_$KeyDatabase, $ServerRowsTable, ServerRow>),
      ServerRow,
      PrefetchHooks Function()
    >;

class $KeyDatabaseManager {
  final _$KeyDatabase _db;
  $KeyDatabaseManager(this._db);
  $$IdentityRowsTableTableManager get identityRows =>
      $$IdentityRowsTableTableManager(_db, _db.identityRows);
  $$ServerRowsTableTableManager get serverRows =>
      $$ServerRowsTableTableManager(_db, _db.serverRows);
}
