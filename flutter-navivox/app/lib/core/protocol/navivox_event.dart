enum NavivoxMessageKind { text, toolCall, voice }

enum NavivoxMessageAuthor { user, assistant, system }

class NavivoxChatMessage {
  const NavivoxChatMessage({
    required this.id,
    required this.author,
    required this.kind,
    required this.createdAt,
    this.text,
    this.toolCall,
    this.voice,
  });

  final String id;
  final NavivoxMessageAuthor author;
  final NavivoxMessageKind kind;
  final DateTime createdAt;
  final String? text;
  final NavivoxToolCall? toolCall;
  final NavivoxVoiceMessage? voice;
}

class NavivoxToolCall {
  const NavivoxToolCall({
    required this.name,
    required this.status,
    required this.summary,
  });

  final String name;
  final String status;
  final String summary;
}

class NavivoxVoiceMessage {
  const NavivoxVoiceMessage({
    required this.duration,
    required this.transcript,
    required this.confidence,
  });

  final Duration duration;
  final String transcript;
  final double confidence;
}
