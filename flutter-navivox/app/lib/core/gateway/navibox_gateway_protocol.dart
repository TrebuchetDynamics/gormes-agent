class NaviboxGatewayConfig {
  const NaviboxGatewayConfig({required this.baseUri, this.token});

  factory NaviboxGatewayConfig.fromBaseUrl(String baseUrl, {String? token}) {
    return NaviboxGatewayConfig(baseUri: Uri.parse(baseUrl), token: token);
  }

  final Uri baseUri;
  final String? token;

  Uri get healthUri => _withPath('/healthz');
  Uri get statusUri => _withPath('/v1/navibox/status');
  Uri get sessionsUri => _withPath('/v1/navibox/sessions');
  Uri sessionUri(String sessionId) =>
      _withPath('/v1/navibox/sessions/$sessionId');
  Uri get turnUri => _withPath('/v1/navibox/turn');

  Uri get streamUri {
    final scheme = switch (baseUri.scheme) {
      'https' => 'wss',
      'http' => 'ws',
      'wss' || 'ws' => baseUri.scheme,
      _ => 'ws',
    };
    return _withPath('/v1/navibox/stream').replace(scheme: scheme);
  }

  Map<String, String> get headers {
    final value = token?.trim();
    if (value == null || value.isEmpty) {
      return const {};
    }
    return {'Authorization': 'Bearer $value'};
  }

  Uri _withPath(String path) {
    return baseUri.replace(path: path, query: null);
  }
}

class NaviboxGatewayMessage {
  const NaviboxGatewayMessage._(this.body);

  factory NaviboxGatewayMessage.ping({required String requestId}) {
    return NaviboxGatewayMessage._({'type': 'ping', 'request_id': requestId});
  }

  factory NaviboxGatewayMessage.startTurn({
    required String requestId,
    String? sessionId,
    required String text,
    Map<String, Object?> metadata = const {
      'client': 'navibox',
      'platform': 'flutter',
    },
  }) {
    return NaviboxGatewayMessage._({
      'type': 'start_turn',
      'request_id': requestId,
      if (sessionId != null && sessionId.trim().isNotEmpty)
        'session_id': sessionId,
      'text': text,
      'metadata': metadata,
    });
  }

  factory NaviboxGatewayMessage.cancelTurn({
    required String requestId,
    required String sessionId,
  }) {
    return NaviboxGatewayMessage._({
      'type': 'cancel_turn',
      'request_id': requestId,
      'session_id': sessionId,
    });
  }

  factory NaviboxGatewayMessage.subscribeSession({
    required String requestId,
    required String sessionId,
  }) {
    return NaviboxGatewayMessage._({
      'type': 'subscribe_session',
      'request_id': requestId,
      'session_id': sessionId,
    });
  }

  final Map<String, Object?> body;
}

class NaviboxGatewayEvent {
  const NaviboxGatewayEvent({
    required this.type,
    this.requestId,
    this.sessionId,
    this.text,
    this.code,
    this.message,
    this.toolName,
    this.toolCallId,
    this.status,
  });

  factory NaviboxGatewayEvent.fromJson(Map<String, Object?> json) {
    return NaviboxGatewayEvent(
      type: json['type']?.toString() ?? '',
      requestId: json['request_id']?.toString(),
      sessionId: json['session_id']?.toString(),
      text: json['text']?.toString(),
      code: json['code']?.toString(),
      message: json['message']?.toString(),
      toolName: json['tool_name']?.toString(),
      toolCallId: json['tool_call_id']?.toString(),
      status: json['status']?.toString(),
    );
  }

  final String type;
  final String? requestId;
  final String? sessionId;
  final String? text;
  final String? code;
  final String? message;
  final String? toolName;
  final String? toolCallId;
  final String? status;

  bool get isError => type == 'error';
}
