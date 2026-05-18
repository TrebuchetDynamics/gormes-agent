class NavivoxGatewayConfig {
  const NavivoxGatewayConfig({required this.baseUri, this.token});

  factory NavivoxGatewayConfig.fromBaseUrl(String baseUrl, {String? token}) {
    return NavivoxGatewayConfig(baseUri: Uri.parse(baseUrl), token: token);
  }

  final Uri baseUri;
  final String? token;

  Uri get healthUri => _withPath('/healthz');
  Uri get statusUri => _withPath('/v1/navivox/status');
  Uri get profileContactsUri => _withPath('/v1/navivox/profile-contacts');
  Uri get sessionsUri => _withPath('/v1/navivox/sessions');
  Uri sessionUri(String sessionId) =>
      _withPath('/v1/navivox/sessions/$sessionId');
  Uri get turnUri => _withPath('/v1/navivox/turn');

  Uri get streamUri {
    final scheme = switch (baseUri.scheme) {
      'https' => 'wss',
      'http' => 'ws',
      'wss' || 'ws' => baseUri.scheme,
      _ => 'ws',
    };
    return _withPath('/v1/navivox/stream').replace(scheme: scheme);
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

class NavivoxGatewayMessage {
  const NavivoxGatewayMessage._(this.body);

  factory NavivoxGatewayMessage.ping({required String requestId}) {
    return NavivoxGatewayMessage._({'type': 'ping', 'request_id': requestId});
  }

  factory NavivoxGatewayMessage.startTurn({
    required String requestId,
    String? sessionId,
    required String text,
    Map<String, Object?> metadata = const {
      'client': 'navivox',
      'platform': 'flutter',
    },
  }) {
    return NavivoxGatewayMessage._({
      'type': 'start_turn',
      'request_id': requestId,
      if (sessionId != null && sessionId.trim().isNotEmpty)
        'session_id': sessionId,
      'text': text,
      'metadata': metadata,
    });
  }

  factory NavivoxGatewayMessage.cancelTurn({
    required String requestId,
    required String sessionId,
  }) {
    return NavivoxGatewayMessage._({
      'type': 'cancel_turn',
      'request_id': requestId,
      'session_id': sessionId,
    });
  }

  factory NavivoxGatewayMessage.subscribeSession({
    required String requestId,
    required String sessionId,
  }) {
    return NavivoxGatewayMessage._({
      'type': 'subscribe_session',
      'request_id': requestId,
      'session_id': sessionId,
    });
  }

  final Map<String, Object?> body;
}

class NavivoxGatewayEvent {
  const NavivoxGatewayEvent({
    required this.type,
    this.requestId,
    this.sessionId,
    this.text,
    this.code,
    this.message,
    this.toolName,
    this.toolCallId,
    this.status,
    this.contact,
  });

  factory NavivoxGatewayEvent.fromJson(Map<String, Object?> json) {
    final contact = json['contact'];
    return NavivoxGatewayEvent(
      type: json['type']?.toString() ?? '',
      requestId: json['request_id']?.toString(),
      sessionId: json['session_id']?.toString(),
      text: json['text']?.toString(),
      code: json['code']?.toString(),
      message: json['message']?.toString(),
      toolName: json['tool_name']?.toString(),
      toolCallId: json['tool_call_id']?.toString(),
      status: json['status']?.toString(),
      contact: contact is Map ? Map<String, Object?>.from(contact) : null,
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
  final Map<String, Object?>? contact;

  bool get isError => type == 'error';
}
