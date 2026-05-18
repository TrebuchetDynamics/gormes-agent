import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/channel/gateway_navivox_channel.dart';
import 'package:navivox/core/channel/navivox_channel.dart';
import 'package:navivox/core/gateway/navivox_gateway_protocol.dart';
import 'package:navivox/core/protocol/navivox_event.dart';

void main() {
  test('connects to gateway and streams a chat turn', () async {
    final server = await _FakeGatewayServer.start();
    addTearDown(server.close);

    final channel = GatewayNavivoxChannel();
    addTearDown(channel.dispose);

    await channel.connect(
      NavivoxGatewayConfig.fromBaseUrl(
        server.baseUrl,
        token: _FakeGatewayServer.token,
      ),
    );

    expect(channel.state.activeServer?.name, 'Gormes Gateway');
    expect(channel.state.activeServer?.status, contains('Gateway online'));

    final completed = Completer<void>();
    channel.addListener(() {
      final texts = channel.state.messages.map((m) => m.text).toList();
      if (texts.contains('hello from gateway') && !completed.isCompleted) {
        completed.complete();
      }
    });

    channel.sendText('hello gateway');

    final sent = await server.nextClientMessage;
    expect(sent['type'], 'start_turn');
    expect(sent['text'], 'hello gateway');
    await completed.future.timeout(const Duration(seconds: 2));

    final messages = channel.state.messages;
    expect(messages.where((m) => m.text == 'hello gateway'), hasLength(1));
    expect(messages.where((m) => m.text == 'hello from gateway'), hasLength(1));
  });

  test('selected profile scope is included in gateway turn metadata', () async {
    final server = await _FakeGatewayServer.start();
    addTearDown(server.close);

    final channel = GatewayNavivoxChannel();
    addTearDown(channel.dispose);

    await channel.connect(
      NavivoxGatewayConfig.fromBaseUrl(
        server.baseUrl,
        token: _FakeGatewayServer.token,
      ),
    );
    channel.selectProfileContact(
      serverId: 'navivox-gateway',
      profileId: 'default',
    );

    channel.sendText('hello scoped gateway');

    final sent = await server.nextClientMessage;
    final metadata = Map<String, Object?>.from(sent['metadata'] as Map);
    expect(metadata['server_id'], 'navivox-gateway');
    expect(metadata['profile_id'], 'default');
    expect(metadata['client'], 'navivox');
  });

  test(
    'loads profile contacts from snapshot and applies gateway updates',
    () async {
      final server = await _FakeGatewayServer.start(
        contacts: [
          {
            'server_id': 'local-gormes',
            'profile_id': 'mineru',
            'display_name': 'Mineru Builder',
            'server_label': 'local',
            'avatar_seed': 'local-gormes:mineru',
            'latest_preview': 'Ready to work',
            'latest_preview_kind': 'status',
            'latest_preview_at': '2026-05-18T06:30:00Z',
            'health': 'online',
            'workspace_root_count': 2,
            'workspace_roots_ok': true,
            'workspace_roots_warning': 0,
            'workspace_roots_error': 0,
            'attention_badges': ['approval'],
            'mic_available': true,
            'active_turn_state': 'idle',
          },
        ],
        streamEvents: (requestId) => [
          {
            'type': 'profile_contact_update',
            'request_id': requestId,
            'contact': {
              'server_id': 'local-gormes',
              'profile_id': 'mineru',
              'display_name': 'Mineru Builder',
              'server_label': 'local',
              'avatar_seed': 'local-gormes:mineru',
              'latest_preview': 'Checking status',
              'latest_preview_kind': 'user',
              'latest_preview_at': '2026-05-18T06:31:00Z',
              'health': 'warning',
              'workspace_root_count': 2,
              'workspace_roots_ok': false,
              'workspace_roots_warning': 1,
              'workspace_roots_error': 0,
              'attention_badges': ['workspace'],
              'mic_available': false,
              'active_turn_state': 'active',
            },
          },
        ],
      );
      addTearDown(server.close);

      final channel = GatewayNavivoxChannel();
      addTearDown(channel.dispose);

      await channel.connect(
        NavivoxGatewayConfig.fromBaseUrl(
          server.baseUrl,
          token: _FakeGatewayServer.token,
        ),
      );

      expect(channel.state.profileContacts, hasLength(1));
      expect(channel.state.profileContacts.single.profileId, 'mineru');
      expect(
        channel.state.profileContacts.single.latestPreview,
        'Ready to work',
      );
      expect(channel.state.profileContacts.single.workspaceRootCount, 2);
      expect(channel.state.profileContacts.single.attentionBadges, [
        'approval',
      ]);

      final updated = Completer<void>();
      channel.addListener(() {
        final contacts = channel.state.profileContacts;
        if (contacts.length == 1 &&
            contacts.single.latestPreview == 'Checking status' &&
            !updated.isCompleted) {
          updated.complete();
        }
      });

      channel.selectProfileContact(
        serverId: 'local-gormes',
        profileId: 'mineru',
      );
      channel.sendText('Checking status');
      await server.nextClientMessage;
      await updated.future.timeout(const Duration(seconds: 2));

      final contact = channel.state.profileContacts.single;
      expect(contact.health, NavivoxProfileHealth.warning);
      expect(contact.workspaceRootsOk, isFalse);
      expect(contact.workspaceRootsWarning, 1);
      expect(contact.micAvailable, isFalse);
      expect(contact.activeTurnState, 'active');
    },
  );

  test(
    'voice transcript renders locally and submits as a gateway turn',
    () async {
      final server = await _FakeGatewayServer.start();
      addTearDown(server.close);

      final channel = GatewayNavivoxChannel();
      addTearDown(channel.dispose);

      await channel.connect(
        NavivoxGatewayConfig.fromBaseUrl(
          server.baseUrl,
          token: _FakeGatewayServer.token,
        ),
      );

      channel.sendVoice(
        audio: Uint8List.fromList([1, 2, 3]),
        transcript: 'hello by voice',
        duration: const Duration(milliseconds: 1200),
        confidence: 0.91,
      );

      final sent = await server.nextClientMessage;
      expect(sent['type'], 'start_turn');
      expect(sent['text'], 'hello by voice');

      final voiceMessages = channel.state.messages
          .where((message) => message.kind == NavivoxMessageKind.voice)
          .toList();
      expect(voiceMessages, hasLength(1));
      expect(voiceMessages.single.author, NavivoxMessageAuthor.user);
      expect(voiceMessages.single.voice?.transcript, 'hello by voice');
      expect(
        voiceMessages.single.voice?.duration,
        const Duration(milliseconds: 1200),
      );
      expect(voiceMessages.single.voice?.confidence, 0.91);
    },
  );

  test('tool progress events render as one durable tool-call card', () async {
    final server = await _FakeGatewayServer.start(
      streamEvents: (requestId) => [
        {
          'type': 'session_started',
          'request_id': requestId,
          'session_id': 's-test',
        },
        {
          'type': 'tool_call_started',
          'request_id': requestId,
          'session_id': 's-test',
          'tool_call_id': 'req-tool-browser-1',
          'tool_name': 'browser_navigate',
          'status': 'started',
          'message': 'browser_navigate started',
        },
        {
          'type': 'tool_call_finished',
          'request_id': requestId,
          'session_id': 's-test',
          'tool_call_id': 'req-tool-browser-1',
          'tool_name': 'browser_navigate',
          'status': 'finished',
          'message': 'browser_navigate finished',
        },
        {'type': 'done', 'request_id': requestId, 'session_id': 's-test'},
      ],
    );
    addTearDown(server.close);

    final channel = GatewayNavivoxChannel();
    addTearDown(channel.dispose);

    await channel.connect(
      NavivoxGatewayConfig.fromBaseUrl(
        server.baseUrl,
        token: _FakeGatewayServer.token,
      ),
    );

    final completed = Completer<void>();
    channel.addListener(() {
      final cards = channel.state.messages
          .where((message) => message.kind == NavivoxMessageKind.toolCall)
          .toList();
      if (cards.length == 1 &&
          cards.single.toolCall?.status == 'finished' &&
          !completed.isCompleted) {
        completed.complete();
      }
    });

    channel.sendText('open dashboard');
    await server.nextClientMessage;
    await completed.future.timeout(const Duration(seconds: 2));

    final cards = channel.state.messages
        .where((message) => message.kind == NavivoxMessageKind.toolCall)
        .toList();
    expect(cards, hasLength(1));
    expect(cards.single.toolCall?.name, 'browser_navigate');
    expect(cards.single.toolCall?.status, 'finished');
    expect(cards.single.toolCall?.summary, 'browser_navigate finished');
  });
}

class _FakeGatewayServer {
  _FakeGatewayServer._(
    this._server,
    this.port,
    this._streamEvents,
    this._contacts,
  );

  static const token = 'nvbx_test_token';

  final HttpServer _server;
  final int port;
  final List<Map<String, Object?>> Function(String requestId)? _streamEvents;
  final List<Map<String, Object?>>? _contacts;
  final Completer<Map<String, Object?>> _nextClientMessage = Completer();

  String get baseUrl => 'http://127.0.0.1:$port';
  Future<Map<String, Object?>> get nextClientMessage =>
      _nextClientMessage.future;

  static Future<_FakeGatewayServer> start({
    List<Map<String, Object?>> Function(String requestId)? streamEvents,
    List<Map<String, Object?>>? contacts,
  }) async {
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    final fake = _FakeGatewayServer._(
      server,
      server.port,
      streamEvents,
      contacts,
    );
    server.listen(fake._handle);
    return fake;
  }

  Future<void> close() async {
    await _server.close(force: true);
  }

  Future<void> _handle(HttpRequest request) async {
    if (request.uri.path == '/healthz') {
      _writeJson(request.response, {'status': 'ok'});
      return;
    }
    if (!_authorized(request)) {
      request.response.statusCode = HttpStatus.unauthorized;
      await request.response.close();
      return;
    }
    if (request.uri.path == '/v1/navivox/status') {
      _writeJson(request.response, {'enabled': true});
      return;
    }
    if (request.uri.path == '/v1/navivox/profile-contacts') {
      _writeJson(request.response, {
        'contacts':
            _contacts ??
            [
              {
                'server_id': 'navivox-gateway',
                'profile_id': 'default',
                'display_name': 'Default profile',
                'server_label': 'Gormes Gateway',
                'health': 'online',
                'latest_preview': 'Gateway online',
                'latest_preview_kind': 'status',
                'workspace_root_count': 1,
                'workspace_roots_ok': true,
                'workspace_roots_warning': 0,
                'workspace_roots_error': 0,
                'attention_badges': <String>[],
                'mic_available': true,
                'active_turn_state': 'idle',
              },
            ],
      });
      return;
    }
    if (request.uri.path == '/v1/navivox/stream') {
      final socket = await WebSocketTransformer.upgrade(request);
      socket.listen((raw) {
        final decoded = Map<String, Object?>.from(jsonDecode(raw as String));
        if (!_nextClientMessage.isCompleted) {
          _nextClientMessage.complete(decoded);
        }
        final requestId = decoded['request_id']?.toString() ?? 'req-test';
        for (final event in _eventsFor(requestId)) {
          socket.add(jsonEncode(event));
        }
      });
      return;
    }
    request.response.statusCode = HttpStatus.notFound;
    await request.response.close();
  }

  bool _authorized(HttpRequest request) {
    return request.headers.value(HttpHeaders.authorizationHeader) ==
        'Bearer $token';
  }

  void _writeJson(HttpResponse response, Map<String, Object?> body) {
    response.headers.contentType = ContentType.json;
    response.write(jsonEncode(body));
    unawaited(response.close());
  }

  List<Map<String, Object?>> _eventsFor(String requestId) {
    final custom = _streamEvents;
    if (custom != null) {
      return custom(requestId);
    }
    return [
      {
        'type': 'session_started',
        'request_id': requestId,
        'session_id': 's-test',
      },
      {
        'type': 'assistant_delta',
        'request_id': requestId,
        'session_id': 's-test',
        'text': 'hello ',
      },
      {
        'type': 'assistant_message',
        'request_id': requestId,
        'session_id': 's-test',
        'text': 'hello from gateway',
      },
      {'type': 'done', 'request_id': requestId, 'session_id': 's-test'},
    ];
  }
}
