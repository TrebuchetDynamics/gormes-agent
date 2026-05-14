import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'navibox_gateway_protocol.dart';

typedef NaviboxGatewayGet =
    Future<String> Function(Uri uri, Map<String, String> headers);

typedef NaviboxGatewayWebSocketConnector =
    Future<WebSocket> Function(Uri uri, Map<String, String> headers);

class NaviboxGatewayClient {
  NaviboxGatewayClient({
    required this.config,
    NaviboxGatewayGet? get,
    NaviboxGatewayWebSocketConnector? connectWebSocket,
  }) : _get = get ?? _defaultGet,
       _connectWebSocket = connectWebSocket ?? _defaultConnectWebSocket;

  final NaviboxGatewayConfig config;
  final NaviboxGatewayGet _get;
  final NaviboxGatewayWebSocketConnector _connectWebSocket;

  Future<Map<String, Object?>> health() => _getJson(config.healthUri);
  Future<Map<String, Object?>> status() => _getJson(config.statusUri);

  Future<WebSocket> connectStream() {
    return _connectWebSocket(config.streamUri, config.headers);
  }

  Duration reconnectDelay(int attempt) {
    final bounded = attempt.clamp(0, 6).toInt();
    return Duration(milliseconds: 250 * (1 << bounded));
  }

  Stream<NaviboxGatewayEvent> decodeEvents(Stream<dynamic> wireEvents) {
    return wireEvents.map((event) {
      final decoded = event is String ? jsonDecode(event) : event;
      if (decoded is! Map) {
        return const NaviboxGatewayEvent(
          type: 'error',
          code: 'bad_response',
          message: 'Invalid gateway event',
        );
      }
      return NaviboxGatewayEvent.fromJson(Map<String, Object?>.from(decoded));
    });
  }

  Future<Map<String, Object?>> _getJson(Uri uri) async {
    final body = await _get(uri, config.headers);
    final decoded = jsonDecode(body);
    if (decoded is! Map) {
      throw const FormatException('expected JSON object');
    }
    return Map<String, Object?>.from(decoded);
  }

  static Future<String> _defaultGet(
    Uri uri,
    Map<String, String> headers,
  ) async {
    final client = HttpClient();
    try {
      final request = await client.getUrl(uri);
      headers.forEach(request.headers.set);
      final response = await request.close();
      final body = await utf8.decoder.bind(response).join();
      if (response.statusCode < 200 || response.statusCode >= 300) {
        throw HttpException(
          'Navibox gateway returned HTTP ${response.statusCode}',
          uri: uri,
        );
      }
      return body;
    } finally {
      client.close();
    }
  }

  static Future<WebSocket> _defaultConnectWebSocket(
    Uri uri,
    Map<String, String> headers,
  ) {
    return WebSocket.connect(uri.toString(), headers: headers);
  }
}
