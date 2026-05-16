// ignore_for_file: avoid_web_libraries_in_flutter, deprecated_member_use

import 'dart:async';
import 'dart:convert';
import 'dart:html' as html;

const navivoxWebSocketProtocol = 'gormes.navivox.v1';
const _navivoxWebSocketTokenProtocolPrefix = 'gormes.navivox.token.';

class NavivoxGatewaySocket {
  NavivoxGatewaySocket._(this._socket) {
    _messageSub = _socket.onMessage.listen((event) => _events.add(event.data));
    _errorSub = _socket.onError.listen(
      (event) => _events.addError(StateError('Gateway stream error')),
    );
    _closeSub = _socket.onClose.listen((event) {
      if (!_events.isClosed) unawaited(_events.close());
    });
  }

  final html.WebSocket _socket;
  final StreamController<dynamic> _events = StreamController<dynamic>();
  late final StreamSubscription<html.MessageEvent> _messageSub;
  late final StreamSubscription<html.Event> _errorSub;
  late final StreamSubscription<html.CloseEvent> _closeSub;

  Stream<dynamic> get events => _events.stream;

  void add(String message) => _socket.sendString(message);

  Future<void> close() async {
    _socket.close();
    await _messageSub.cancel();
    await _errorSub.cancel();
    await _closeSub.cancel();
    if (!_events.isClosed) await _events.close();
  }
}

Future<String> defaultGet(Uri uri, Map<String, String> headers) async {
  final response = await html.HttpRequest.request(
    uri.toString(),
    method: 'GET',
    requestHeaders: headers,
  );
  final status = response.status ?? 0;
  if (status < 200 || status >= 300) {
    throw StateError('Navivox gateway returned HTTP $status');
  }
  return response.responseText ?? '';
}

Future<NavivoxGatewaySocket> defaultConnectWebSocket(
  Uri uri,
  Map<String, String> headers,
) async {
  final protocols = <String>[navivoxWebSocketProtocol];
  final token = _bearerToken(headers);
  if (token != null && token.isNotEmpty) {
    protocols.add(
      '$_navivoxWebSocketTokenProtocolPrefix'
      '${base64Url.encode(utf8.encode(token)).replaceAll('=', '')}',
    );
  }

  final socket = html.WebSocket(uri.toString(), protocols);
  final completer = Completer<NavivoxGatewaySocket>();
  late final StreamSubscription<html.Event> openSub;
  late final StreamSubscription<html.Event> errorSub;
  late final StreamSubscription<html.CloseEvent> closeSub;

  Future<void> cleanup() async {
    await openSub.cancel();
    await errorSub.cancel();
    await closeSub.cancel();
  }

  openSub = socket.onOpen.listen((event) {
    unawaited(cleanup());
    if (!completer.isCompleted) {
      completer.complete(NavivoxGatewaySocket._(socket));
    }
  });
  errorSub = socket.onError.listen((event) {
    unawaited(cleanup());
    if (!completer.isCompleted) {
      completer.completeError(StateError('Navivox gateway WebSocket failed'));
    }
  });
  closeSub = socket.onClose.listen((event) {
    unawaited(cleanup());
    if (!completer.isCompleted) {
      completer.completeError(StateError('Navivox gateway WebSocket closed'));
    }
  });

  return completer.future;
}

String? _bearerToken(Map<String, String> headers) {
  final auth = headers.entries
      .where((entry) => entry.key.toLowerCase() == 'authorization')
      .map((entry) => entry.value.trim())
      .firstOrNull;
  if (auth == null || !auth.startsWith('Bearer ')) return null;
  return auth.substring('Bearer '.length).trim();
}
