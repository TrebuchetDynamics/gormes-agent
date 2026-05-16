import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/channel/gateway_navivox_channel.dart';
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
}

class _FakeGatewayServer {
  _FakeGatewayServer._(this._server, this.port);

  static const token = 'nvbx_test_token';

  final HttpServer _server;
  final int port;
  final Completer<Map<String, Object?>> _nextClientMessage = Completer();

  String get baseUrl => 'http://127.0.0.1:$port';
  Future<Map<String, Object?>> get nextClientMessage =>
      _nextClientMessage.future;

  static Future<_FakeGatewayServer> start() async {
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    final fake = _FakeGatewayServer._(server, server.port);
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
    if (request.uri.path == '/v1/navivox/stream') {
      final socket = await WebSocketTransformer.upgrade(request);
      socket.listen((raw) {
        final decoded = Map<String, Object?>.from(jsonDecode(raw as String));
        if (!_nextClientMessage.isCompleted) {
          _nextClientMessage.complete(decoded);
        }
        final requestId = decoded['request_id']?.toString() ?? 'req-test';
        socket.add(
          jsonEncode({
            'type': 'session_started',
            'request_id': requestId,
            'session_id': 's-test',
          }),
        );
        socket.add(
          jsonEncode({
            'type': 'assistant_delta',
            'request_id': requestId,
            'session_id': 's-test',
            'text': 'hello ',
          }),
        );
        socket.add(
          jsonEncode({
            'type': 'assistant_message',
            'request_id': requestId,
            'session_id': 's-test',
            'text': 'hello from gateway',
          }),
        );
        socket.add(
          jsonEncode({
            'type': 'done',
            'request_id': requestId,
            'session_id': 's-test',
          }),
        );
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
}
