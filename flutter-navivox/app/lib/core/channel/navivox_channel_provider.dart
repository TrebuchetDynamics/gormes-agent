import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'gateway_navivox_channel.dart';
import 'navivox_channel.dart';

/// Production-only Riverpod provider for the active Navivox channel. The
/// HTTP/WebSocket gateway is the sole supported transport; the in-memory
/// fake-server mode and SSH wire protocol were deleted with the Navivox
/// HTTP-only hardening rows under phase 9.E.
///
/// Tests override this provider with their own [NavivoxChannel] mock.
final navivoxChannelProvider = Provider<NavivoxChannel>((ref) {
  final channel = GatewayNavivoxChannel();
  ref.onDispose(channel.dispose);
  return channel;
});

/// Existing call sites still import `gatewayNavivoxChannelProvider` directly
/// to drive `.connect(config)` from the setup screen. It stays a separate
/// provider so the setup flow keeps a typed handle to the gateway.
final gatewayNavivoxChannelProvider = Provider<GatewayNavivoxChannel>((ref) {
  return ref.watch(navivoxChannelProvider) as GatewayNavivoxChannel;
});
