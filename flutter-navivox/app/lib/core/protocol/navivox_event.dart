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
    this.artifacts = const [],
  });

  final String name;
  final String status;
  final String summary;
  final List<NavivoxToolArtifact> artifacts;

  NavivoxToolCall copyWith({
    String? name,
    String? status,
    String? summary,
    List<NavivoxToolArtifact>? artifacts,
  }) {
    return NavivoxToolCall(
      name: name ?? this.name,
      status: status ?? this.status,
      summary: summary ?? this.summary,
      artifacts: artifacts ?? this.artifacts,
    );
  }
}

class NavivoxToolArtifact {
  const NavivoxToolArtifact({
    required this.id,
    required this.kind,
    required this.title,
    this.summary,
    this.ref,
  });

  final String id;
  final String kind;
  final String title;
  final String? summary;
  final String? ref;
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
