import 'package:flutter_riverpod/flutter_riverpod.dart';

class NavivoxVoiceSettings {
  const NavivoxVoiceSettings({
    this.continuousVoiceEnabled = true,
    this.profileSwitchingEnabled = true,
    this.commandWord = 'navi',
    this.trustedServerIds = const {},
  });

  final bool continuousVoiceEnabled;
  final bool profileSwitchingEnabled;
  final String commandWord;
  final Set<String> trustedServerIds;

  bool isTrusted(String serverId) => trustedServerIds.contains(serverId);

  NavivoxVoiceSettings copyWith({
    bool? continuousVoiceEnabled,
    bool? profileSwitchingEnabled,
    String? commandWord,
    Set<String>? trustedServerIds,
  }) {
    return NavivoxVoiceSettings(
      continuousVoiceEnabled:
          continuousVoiceEnabled ?? this.continuousVoiceEnabled,
      profileSwitchingEnabled:
          profileSwitchingEnabled ?? this.profileSwitchingEnabled,
      commandWord: commandWord ?? this.commandWord,
      trustedServerIds: trustedServerIds ?? this.trustedServerIds,
    );
  }
}

class NavivoxVoiceSettingsController extends Notifier<NavivoxVoiceSettings> {
  @override
  NavivoxVoiceSettings build() => const NavivoxVoiceSettings();

  void setContinuousVoiceEnabled(bool enabled) {
    state = state.copyWith(continuousVoiceEnabled: enabled);
  }

  void setProfileSwitchingEnabled(bool enabled) {
    state = state.copyWith(profileSwitchingEnabled: enabled);
  }

  void setCommandWord(String value) {
    final normalized = value.trim().toLowerCase();
    if (normalized.isEmpty || normalized.contains(RegExp(r'\s'))) return;
    state = state.copyWith(commandWord: normalized);
  }

  void setServerTrusted(String serverId, bool trusted) {
    final trimmed = serverId.trim();
    if (trimmed.isEmpty) return;
    final next = {...state.trustedServerIds};
    if (trusted) {
      next.add(trimmed);
    } else {
      next.remove(trimmed);
    }
    state = state.copyWith(trustedServerIds: next);
  }
}

final navivoxVoiceSettingsProvider =
    NotifierProvider<NavivoxVoiceSettingsController, NavivoxVoiceSettings>(
      NavivoxVoiceSettingsController.new,
    );
