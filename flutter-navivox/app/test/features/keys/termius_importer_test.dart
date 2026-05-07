import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/features/keys/services/termius_importer.dart';

const _sampleExport = '''
{
  "hosts": [
    {
      "id": "host-1",
      "label": "prod-web",
      "hostname": "web.example.com",
      "port": 22,
      "username": "deploy",
      "identity": "id-1",
      "group": "Production",
      "tags": ["web", "api"],
      "known_host": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5..."
    },
    {
      "id": "host-2",
      "label": "no-host",
      "hostname": "",
      "port": 22,
      "username": "root"
    },
    {
      "id": "host-3",
      "label": "dev",
      "hostname": "dev.example.com",
      "port": 2222,
      "username": "dev",
      "identity": "id-2"
    }
  ],
  "identities": [
    {
      "id": "id-1",
      "label": "Production key",
      "private_key": "fixture-private-key-id-1",
      "public_key": "ssh-ed25519 AAAAC3...prod",
      "passphrase": null
    },
    {
      "id": "id-2",
      "label": "Encrypted key",
      "private_key": "fixture-private-key-id-2",
      "public_key": "ssh-ed25519 AAAAC3...dev",
      "passphrase": "test-passphrase"
    },
    {
      "id": "id-3",
      "label": "password-only",
      "private_key": null,
      "public_key": null,
      "passphrase": null
    }
  ]
}
''';

void main() {
  group('TermiusImporter', () {
    test('parses identities and skips password-only entries', () {
      final result = TermiusImporter.parse(jsonDecode(_sampleExport));

      expect(result.identities.length, 2);
      final ids = result.identities.map((i) => i.termiusId).toList();
      expect(ids, ['id-1', 'id-2']);

      final encrypted = result.identities.firstWhere(
        (i) => i.termiusId == 'id-2',
      );
      expect(encrypted.isEncrypted, isTrue);

      final unencrypted = result.identities.firstWhere(
        (i) => i.termiusId == 'id-1',
      );
      expect(unencrypted.isEncrypted, isFalse);
      expect(unencrypted.publicKey, contains('prod'));

      expect(result.warnings.any((w) => w.contains('password-only')), isTrue);
    });

    test('parses hosts and rejects entries with empty hostname', () {
      final result = TermiusImporter.parse(jsonDecode(_sampleExport));

      expect(result.servers.length, 2);
      final labels = result.servers.map((s) => s.label).toList();
      expect(labels, containsAll(['prod-web', 'dev']));
      expect(labels, isNot(contains('no-host')));

      expect(result.warnings.any((w) => w.contains('hostname')), isTrue);
    });

    test('preserves group, tags, and known_host when present', () {
      final result = TermiusImporter.parse(jsonDecode(_sampleExport));
      final prodWeb = result.servers.firstWhere((s) => s.label == 'prod-web');
      expect(prodWeb.hostname, 'web.example.com');
      expect(prodWeb.port, 22);
      expect(prodWeb.username, 'deploy');
      expect(prodWeb.termiusIdentity, 'id-1');
      expect(prodWeb.group, 'Production');
      expect(prodWeb.tags, ['web', 'api']);
      expect(prodWeb.knownHostKey, contains('ssh-ed25519'));
    });

    test('throws on malformed top-level JSON', () {
      expect(
        () => TermiusImporter.parse('not a map'),
        throwsA(isA<TermiusImportException>()),
      );
      expect(
        () => TermiusImporter.parse(jsonDecode('{"hosts": "not-a-list"}')),
        throwsA(isA<TermiusImportException>()),
      );
    });

    test('handles an export with no hosts or identities', () {
      final result = TermiusImporter.parse(
        jsonDecode('{"hosts": [], "identities": []}'),
      );
      expect(result.servers, isEmpty);
      expect(result.identities, isEmpty);
      expect(result.warnings, isEmpty);
    });
  });
}
