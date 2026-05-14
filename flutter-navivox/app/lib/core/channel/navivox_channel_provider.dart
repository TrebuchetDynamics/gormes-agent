import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'fake_navivox_channel.dart';
import 'gateway_navibox_channel.dart';
import 'navivox_channel.dart';

enum NavivoxChannelMode { fake, gateway }

final gatewayNaviboxChannelProvider = Provider<GatewayNaviboxChannel>((ref) {
  final channel = GatewayNaviboxChannel();
  ref.onDispose(channel.dispose);
  return channel;
});

final activeNavivoxChannelProvider = Provider<NavivoxChannelSwitcher>((ref) {
  final fake = ref.watch(fakeNavivoxChannelProvider);
  final gateway = ref.watch(gatewayNaviboxChannelProvider);
  final switcher = NavivoxChannelSwitcher(fake: fake, gateway: gateway);
  ref.onDispose(switcher.dispose);
  return switcher;
});

class NavivoxChannelSwitcher extends ChangeNotifier implements NavivoxChannel {
  NavivoxChannelSwitcher({
    required FakeNavivoxChannel fake,
    required GatewayNaviboxChannel gateway,
  }) : _fake = fake,
       _gateway = gateway {
    _fake.addListener(notifyListeners);
    _gateway.addListener(notifyListeners);
  }

  final FakeNavivoxChannel _fake;
  final GatewayNaviboxChannel _gateway;
  NavivoxChannelMode _mode = NavivoxChannelMode.fake;

  NavivoxChannelMode get mode => _mode;
  NavivoxChannel get _active =>
      _mode == NavivoxChannelMode.gateway ? _gateway : _fake;

  void useFake() {
    if (_mode == NavivoxChannelMode.fake) return;
    _mode = NavivoxChannelMode.fake;
    notifyListeners();
  }

  void useGateway() {
    if (_mode == NavivoxChannelMode.gateway) return;
    _mode = NavivoxChannelMode.gateway;
    notifyListeners();
  }

  @override
  NavivoxChannelState get state => _active.state;

  @override
  Stream<NavivoxApprovalRequest> get approvalRequests =>
      _active.approvalRequests;

  @override
  void enterFakeServerMode() {
    useFake();
    _fake.enterFakeServerMode();
  }

  @override
  void sendText(String text) => _active.sendText(text);

  @override
  void sendVoice({
    required Uint8List audio,
    required String transcript,
    required Duration duration,
    required double confidence,
  }) {
    _active.sendVoice(
      audio: audio,
      transcript: transcript,
      duration: duration,
      confidence: confidence,
    );
  }

  @override
  void respondToApproval({required String approvalId, required bool approved}) {
    _active.respondToApproval(approvalId: approvalId, approved: approved);
  }

  @override
  void requestAgentList() => _active.requestAgentList();

  @override
  void selectAgent(String agentId) => _active.selectAgent(agentId);

  @override
  void sendConfigSet({required String field, required Object? value}) {
    _active.sendConfigSet(field: field, value: value);
  }

  @override
  void sendConfigSecretSet({required String name, required String secret}) {
    _active.sendConfigSecretSet(name: name, secret: secret);
  }

  @override
  void dispose() {
    _fake.removeListener(notifyListeners);
    _gateway.removeListener(notifyListeners);
    super.dispose();
  }
}
