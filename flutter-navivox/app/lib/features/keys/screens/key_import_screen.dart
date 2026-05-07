import 'dart:convert';

import 'package:flutter/material.dart';

import '../services/key_store.dart';
import '../services/termius_importer.dart';

class KeyImportScreen extends StatefulWidget {
  const KeyImportScreen({
    required this.identityStore,
    required this.serverStore,
    super.key,
  });

  final IdentityStore identityStore;
  final ServerStore serverStore;

  @override
  State<KeyImportScreen> createState() => _KeyImportScreenState();
}

class _KeyImportScreenState extends State<KeyImportScreen> {
  final _controller = TextEditingController();
  String? _error;
  TermiusImportSummary? _summary;
  List<StoredServer> _imported = const [];

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _import() async {
    setState(() {
      _error = null;
      _summary = null;
    });
    Object? json;
    try {
      json = jsonDecode(_controller.text);
    } catch (e) {
      setState(() => _error = 'Could not parse JSON: $e');
      return;
    }
    try {
      final parsed = TermiusImporter.parse(json);
      final service = TermiusImportService(
        identityStore: widget.identityStore,
        serverStore: widget.serverStore,
      );
      final summary = service.importParsed(parsed);
      final imported = await widget.serverStore.getAll();
      setState(() {
        _summary = summary;
        _imported = imported;
      });
    } on TermiusImportException catch (e) {
      setState(() => _error = 'Could not parse export: $e');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Import keys')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Text('Paste a Termius JSON export'),
            const SizedBox(height: 8),
            TextField(
              controller: _controller,
              maxLines: 10,
              decoration: const InputDecoration(
                border: OutlineInputBorder(),
                hintText: '{ "hosts": [...], "identities": [...] }',
              ),
            ),
            const SizedBox(height: 12),
            FilledButton(
              onPressed: _import,
              child: const Text('Import'),
            ),
            if (_error != null) ...[
              const SizedBox(height: 12),
              Text(_error!, style: const TextStyle(color: Colors.red)),
            ],
            if (_summary != null) ...[
              const SizedBox(height: 12),
              Text(
                'Imported ${_summary!.identitiesImported} identities and '
                '${_summary!.serversImported} hosts',
              ),
              const SizedBox(height: 8),
              Expanded(
                child: ListView.builder(
                  key: const Key('imported-hosts-list'),
                  itemCount: _imported.length,
                  itemBuilder: (context, i) {
                    final s = _imported[i];
                    return ListTile(
                      title: Text(s.label),
                      subtitle: Text('${s.username}@${s.hostname}:${s.port}'),
                    );
                  },
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
