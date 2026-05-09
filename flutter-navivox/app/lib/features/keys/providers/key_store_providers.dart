import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../services/key_store.dart';

/// In-memory by default; overridden in tests and (later) in the real app
/// startup with a Drift-backed store.
final identityStoreProvider = Provider<IdentityStore>(
  (_) => InMemoryIdentityStore(),
);

final serverStoreProvider = Provider<ServerStore>(
  (_) => InMemoryServerStore(),
);
