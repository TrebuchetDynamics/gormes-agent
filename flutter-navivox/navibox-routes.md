# Navivox Route Design

Status: planning draft
Source: derived from navibox-prd.md

## 1. Route Architecture

GoRouter with Riverpod-powered redirect guards and shell routes.

### 1.1 Route Constants

```dart
abstract class AppRoutes {
  static const setup = '/setup';
  static const chats = '/chats';
  static const chatThread = '/chats/:serverId/:threadId';
  static const servers = '/servers';
  static const serverDetail = '/servers/:id';
  static const agents = '/agents';
  static const agentEditor = '/agents/:id/edit';
  static const agentCreate = '/agents/create';
  static const keys = '/keys';
  static const keyImport = '/keys/import';
  static const keyGenerate = '/keys/generate';
  static const config = '/config';
  static const configSection = '/config/:section';
  static const secretEditor = '/config/secrets/:key';
  static const terminal = '/terminal';
  static const terminalSession = '/terminal/:serverId';
  static const settings = '/settings';
}
```

## 2. Full Route Table

### 2.1 Setup (First Run)

| Path | Screen | Guard | Notes |
|------|--------|-------|-------|
| `/setup` | FirstRunWizard | None (shown when no servers exist) | 10-step first-run journey grouped into wizard screens |
| `/setup/import` | TermiusImportScreen | None | File picker + preview |
| `/setup/manual` | ManualServerScreen | None | Direct server entry |
| `/setup/keys` | KeyImportOrGenerateScreen | None | Import or generate Ed25519 |
| `/setup/verify/:serverId` | HostVerificationScreen | None | Fingerprint display + pin |
| `/setup/probe/:serverId` | GormesProbeScreen | None | Check for Gormes binary |
| `/setup/pair/:serverId` | DevicePairingScreen | None | Role assignment |
| `/setup/agents/:serverId` | AgentSelectOrCreateScreen | None | First agent selection |

### 2.2 Main Shell (Authenticated)

Shell route wraps all main screens with bottom nav (mobile) or left rail (desktop).

| Tab | Path | Screen | Icon | Guard |
|-----|------|--------|------|-------|
| Chats | `/chats` | ChatsScreen | chat_bubble | Operator+ |
| Servers | `/servers` | ServersScreen | dns | Operator+ |
| Agents | `/agents` | AgentsScreen | smart_toy | Operator+ |
| Config | `/config` | ConfigOverviewScreen | settings | Admin+ |
| Keys | `/keys` | KeysScreen | key | Operator+ |
| Terminal | `/terminal` | TerminalScreen | terminal | Operator+ |
| Settings | `/settings` | SettingsScreen | tune | Operator+ |

### 2.3 Detail Routes (pushed on top of shell)

| Path | Screen | Guard | Notes |
|------|--------|-------|-------|
| `/chats/:serverId/:threadId` | ChatScreen | Paired device | Main chat view |
| `/servers/:id` | ServerDetailScreen | Operator+ | Edit server, view status |
| `/servers/:id/gormes-probe` | GormesProbeScreen | Operator+ | Re-probe for Gormes |
| `/agents/:id/edit` | AgentEditorScreen | Admin+ | Full agent CRUD |
| `/agents/create/:serverId` | AgentCreateScreen | Admin+ | New agent wizard |
| `/keys/import` | KeyImportScreen | Operator+ | File-based SSH key import |
| `/keys/generate` | KeyGenerateScreen | Operator+ | Ed25519 key generation |
| `/config/:section` | ConfigSectionScreen | Admin+ | Section-specific config editor |
| `/config/secrets/:key` | SecretEditorScreen (biometric-gated) | Admin+ | Set/rotate/delete secrets |
| `/terminal/:serverId` | TerminalSessionScreen | Operator+ | Full-screen SSH terminal |

## 3. Router Configuration

### 3.1 Router Provider (Riverpod)

```dart
final routerProvider = Provider<GoRouter>((ref) {
  final pairingState = ref.watch(pairingStateProvider);
  final serverCount = ref.watch(serverCountProvider);

  return GoRouter(
    initialLocation: '/chats',
    debugLogDiagnostics: kDebugMode,
    refreshListenable: pairingState,

    redirect: (context, state) {
      // First run: no servers configured
      final hasServers = serverCount > 0;
      final isOnSetup = state.matchedLocation.startsWith('/setup');
      final isOnSettings = state.matchedLocation == '/settings';

      if (!hasServers && !isOnSetup) return '/setup';
      if (hasServers && isOnSetup) return '/chats';

      return null; // no redirect
    },

    routes: [
      // Setup flow (no shell)
      GoRoute(
        path: '/setup',
        builder: (_, __) => const FirstRunWizard(),
        routes: [
          GoRoute(path: 'import', builder: (_, __) => const TermiusImportScreen()),
          GoRoute(path: 'manual', builder: (_, __) => const ManualServerScreen()),
          GoRoute(path: 'keys', builder: (_, __) => const KeyImportOrGenerateScreen()),
          GoRoute(path: 'verify/:serverId', builder: (_, state) =>
            HostVerificationScreen(serverId: state.pathParameters['serverId']!)),
          GoRoute(path: 'probe/:serverId', builder: (_, state) =>
            GormesProbeScreen(serverId: state.pathParameters['serverId']!)),
          GoRoute(path: 'pair/:serverId', builder: (_, state) =>
            DevicePairingScreen(serverId: state.pathParameters['serverId']!)),
          GoRoute(path: 'agents/:serverId', builder: (_, state) =>
            AgentSelectOrCreateScreen(serverId: state.pathParameters['serverId']!)),
        ],
      ),

      // Main app shell
      ShellRoute(
        builder: (context, state, child) {
          return AdaptiveScaffold(
            body: child,
            currentLocation: state.matchedLocation,
          );
        },
        routes: [
          // Chats
          GoRoute(
            path: '/chats',
            builder: (_, __) => const ChatsScreen(),
          ),
          GoRoute(
            path: '/chats/:serverId/:threadId',
            builder: (_, state) => ChatScreen(
              serverId: state.pathParameters['serverId']!,
              threadId: state.pathParameters['threadId']!,
            ),
          ),

          // Servers
          GoRoute(
            path: '/servers',
            builder: (_, __) => const ServersScreen(),
            routes: [
              GoRoute(
                path: ':id',
                builder: (_, state) => ServerDetailScreen(
                  id: state.pathParameters['id']!,
                ),
              ),
            ],
          ),

          // Agents
          GoRoute(
            path: '/agents',
            builder: (_, __) => const AgentsScreen(),
            routes: [
              GoRoute(
                path: ':id/edit',
                builder: (_, state) => AgentEditorScreen(
                  agentId: state.pathParameters['id']!,
                ),
              ),
              GoRoute(
                path: 'create/:serverId',
                builder: (_, state) => AgentCreateScreen(
                  serverId: state.pathParameters['serverId']!,
                ),
              ),
            ],
          ),

          // Keys
          GoRoute(
            path: '/keys',
            builder: (_, __) => const KeysScreen(),
            routes: [
              GoRoute(path: 'import', builder: (_, __) => const KeyImportScreen()),
              GoRoute(path: 'generate', builder: (_, __) => const KeyGenerateScreen()),
            ],
          ),

          // Config (Admin+ only with redirect guard)
          GoRoute(
            path: '/config',
            redirect: (context, state) {
              final serverId = ref.read(activeServerIdProvider);
              final role = ref.read(pairingRoleProvider(serverId));
              if (role == null || (role != PairingRole.owner && role != PairingRole.admin)) {
                return '/chats'; // redirect non-admins
              }
              return null;
            },
            builder: (_, __) => const ConfigOverviewScreen(),
            routes: [
              GoRoute(
                path: ':section',
                builder: (_, state) => ConfigSectionScreen(
                  section: state.pathParameters['section']!,
                ),
              ),
              GoRoute(
                path: 'secrets/:key',
                builder: (_, state) => SecretEditorScreen(
                  configKey: state.pathParameters['key']!,
                ),
              ),
            ],
          ),

          // Terminal
          GoRoute(
            path: '/terminal',
            builder: (_, __) => const TerminalScreen(),
            routes: [
              GoRoute(
                path: ':serverId',
                builder: (_, state) => TerminalSessionScreen(
                  serverId: state.pathParameters['serverId']!,
                ),
              ),
            ],
          ),

          // Settings
          GoRoute(
            path: '/settings',
            builder: (_, __) => const SettingsScreen(),
          ),
        ],
      ),
    ],

    errorBuilder: (context, state) => NotFoundScreen(error: state.error),
  );
});
```

## 4. Route Guards

### 4.1 Role-Based Guards

```dart
// Config mutation guard
final canMutateConfigProvider = Provider.family<bool, String>((ref, serverId) {
  final role = ref.watch(pairingRoleProvider(serverId));
  return role == PairingRole.owner || role == PairingRole.admin;
});

// Agent mutation guard
final canManageAgentsProvider = Provider.family<bool, String>((ref, serverId) {
  final role = ref.watch(pairingRoleProvider(serverId));
  return role == PairingRole.owner || role == PairingRole.admin;
});

// View-only guard
final canViewProvider = Provider.family<bool, String>((ref, serverId) {
  final role = ref.watch(pairingRoleProvider(serverId));
  return role != null; // any paired role can view
});
```

### 4.2 Inline Widget Guards

For fine-grained UI control within screens:

```dart
class ConfigGuard extends ConsumerWidget {
  final String serverId;
  final Widget child;

  const ConfigGuard({required this.serverId, required this.child});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final canMutate = ref.watch(canMutateConfigProvider(serverId));
    if (!canMutate) {
      return const ReadOnlyConfigView();
    }
    return child;
  }
}
```

## 5. Navigation Patterns

### 5.1 Server Switcher (in chat top bar)

Changes active server context without leaving chat:

```dart
void switchServer(BuildContext context, String newServerId) {
  ref.read(activeServerIdProvider.notifier).state = newServerId;
  // Chat screen rebuilds with new server context
  // Navivox channel reconnects if needed
}
```

### 5.2 Agent Switcher (in chat overlay)

Bottom sheet or popup that switches agent for current thread:

```dart
void switchAgent(BuildContext context, String serverId, String agentId) {
  ref.read(selectedAgentProvider(serverId).notifier).state = agentId;
  // Sends agent.select event over navivox channel
  ref.read(navivoxChannelProvider(serverId)).requireValue!.selectAgent(agentId);
}
```

### 5.3 Deep Link Support

```
navivox://chat/<serverId>/<threadId>
navivox://server/<serverId>
navivox://config/<serverId>/<section>
```

## 6. Mobile Navigation Layout

```
┌─────────────────────┐
│  App Bar (context)  │  ← Server name, agent pill, connection status
│                     │
│  Content Area       │
│  (GoRouter child)   │
│                     │
│                     │
│                     │
├─────────────────────┤
│ Chats │ Srv │ Agt │  ← Bottom Navigation Bar
│       │     │     │
│ Config│Keys │Term │  ← Scrollable on mobile
└─────────────────────┘
```

## 7. Desktop Navigation Layout

```
┌──────┬──────────────────────────────────┐
│      │  Top Bar                         │
│      │  Server: my-server  Agent: mineru│
│ Left ├──────────────────────────────────┤
│ Rail │                                  │
│      │  Content Area                    │
│ 💬   │  (GoRouter child)                │
│ 🖥️   │                                  │
│ 🤖   │                                  │
│ ⚙️   │                                  │
│ 🔑   │                                  │
│ ⌨️   │                                  │
│ ⚡   │                                  │
│      │                                  │
├──────┴──────────────────────────────────┤
│  Status Bar (connection, version, etc.) │
└─────────────────────────────────────────┘
```

## 8. First-Run Flow Navigation

The first-run wizard uses a `PageController` or `Stepper` widget internally, not separate routes:

```
Step 1: Import from Termius or add server manually
        │
        ▼
Step 2: Select or generate SSH key
        │
        ▼
Step 3: Connect + verify host fingerprint
        │
        ▼
Step 4: Probe for Gormes
        │
        ▼
Step 5: Pair device (owner for first device)
        │
        ▼
Step 6: Select existing agent or create one
        │
        ▼
        └──→ Navigate to /chats
```

The wizard can be skipped at certain steps (e.g., skip Gormes probe and use as generic SSH terminal).

## 9. Route Transition Animations

| Transition | Use Case |
|------------|----------|
| Slide right → left | Push to detail screens |
| Slide left → right | Pop back |
| Fade through | Tab switches in shell |
| Modal bottom sheet | Agent switcher, quick actions |
| Full-screen dialog | Secret editor, host key verification |

```dart
// Custom page builder for GoRouter transitions
CustomTransitionPage(
  child: child,
  transitionsBuilder: (context, animation, secondaryAnimation, child) {
    return SlideTransition(
      position: Tween<Offset>(
        begin: const Offset(1.0, 0.0),
        end: Offset.zero,
      ).animate(animation),
      child: child,
    );
  },
);
```
