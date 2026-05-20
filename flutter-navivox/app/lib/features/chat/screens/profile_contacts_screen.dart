import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../core/channel/navivox_channel.dart';
import '../../../core/channel/navivox_channel_provider.dart';

class ProfileContactsScreen extends ConsumerStatefulWidget {
  const ProfileContactsScreen({super.key});

  @override
  ConsumerState<ProfileContactsScreen> createState() =>
      _ProfileContactsScreenState();
}

class _ProfileContactsScreenState extends ConsumerState<ProfileContactsScreen> {
  NavivoxChannel? _subscribed;

  void _onChannelChanged() {
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    _subscribed?.removeListener(_onChannelChanged);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final channel = ref.watch(navivoxChannelProvider);
    if (!identical(_subscribed, channel)) {
      _subscribed?.removeListener(_onChannelChanged);
      channel.addListener(_onChannelChanged);
      _subscribed = channel;
    }

    final contacts = [...channel.state.profileContacts]
      ..sort((a, b) => a.displayName.compareTo(b.displayName));

    return Scaffold(
      appBar: AppBar(
        title: const Text('Navivox'),
        actions: [
          IconButton(
            tooltip: 'Search profiles',
            onPressed: () {},
            icon: const Icon(Icons.search),
          ),
        ],
      ),
      body: contacts.isEmpty
          ? const Center(child: Text('No profiles loaded'))
          : ListView.separated(
              itemCount: contacts.length,
              separatorBuilder: (context, index) => const Divider(height: 1),
              itemBuilder: (context, index) {
                final contact = contacts[index];
                return _ProfileContactTile(
                  contact: contact,
                  onTap: () {
                    channel.selectProfileContact(
                      serverId: contact.serverId,
                      profileId: contact.profileId,
                    );
                    context.go(
                      '/chats/${contact.serverId}/${contact.profileId}',
                    );
                  },
                  onLongPress: () => _showProfileDetails(context, contact),
                );
              },
            ),
      floatingActionButton: FloatingActionButton.small(
        tooltip: 'Add profile',
        onPressed: () => _showAddProfilePlaceholder(context),
        child: const Icon(Icons.add),
      ),
    );
  }

  void _showAddProfilePlaceholder(BuildContext context) {
    showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder: (context) => SafeArea(
        child: ListView(
          shrinkWrap: true,
          children: const [
            ListTile(
              leading: Icon(Icons.person_add_alt),
              title: Text('New profile'),
              subtitle: Text('Server-validated profile creation is next.'),
            ),
            ListTile(
              leading: Icon(Icons.dns),
              title: Text('Add server'),
              subtitle: Text('Import connect-info from Gormes.'),
            ),
          ],
        ),
      ),
    );
  }

  void _showProfileDetails(
    BuildContext context,
    NavivoxProfileContact contact,
  ) {
    showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder: (context) => SafeArea(
        child: ListView(
          shrinkWrap: true,
          children: [
            ListTile(
              leading: _ProfileAvatar(contact: contact),
              title: const Text('Profile details'),
              subtitle: Text('${contact.displayName}\n${contact.profileId}'),
            ),
            ListTile(
              leading: const Icon(Icons.edit),
              title: const Text('Edit profile'),
              subtitle: const Text('Schema-driven editor placeholder.'),
              onTap: () => Navigator.of(context).pop(),
            ),
          ],
        ),
      ),
    );
  }
}

class _ProfileContactTile extends StatelessWidget {
  const _ProfileContactTile({
    required this.contact,
    required this.onTap,
    required this.onLongPress,
  });

  final NavivoxProfileContact contact;
  final VoidCallback onTap;
  final VoidCallback onLongPress;

  @override
  Widget build(BuildContext context) {
    return ListTile(
      key: ValueKey('profile-contact-${contact.serverId}-${contact.profileId}'),
      leading: _ProfileAvatar(contact: contact),
      title: Row(
        children: [
          Expanded(child: Text(contact.displayName)),
          _ServerChip(label: contact.serverLabel),
        ],
      ),
      subtitle: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  _previewLabel(contact),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: contact.activeTurnState == 'streaming'
                      ? TextStyle(
                          color: Theme.of(context).colorScheme.primary,
                          fontStyle: FontStyle.italic,
                        )
                      : null,
                ),
              ),
              if (contact.activeTurnState == 'streaming')
                Container(
                  key: ValueKey(
                    'profile-active-turn-${contact.serverId}-${contact.profileId}',
                  ),
                  margin: const EdgeInsets.only(left: 6),
                  width: 8,
                  height: 8,
                  decoration: BoxDecoration(
                    color: Theme.of(context).colorScheme.primary,
                    shape: BoxShape.circle,
                  ),
                ),
            ],
          ),
          const SizedBox(height: 4),
          Wrap(
            spacing: 6,
            runSpacing: 4,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              _HealthChip(health: contact.health),
              Text(
                _workspaceLabel(contact),
                style: Theme.of(context).textTheme.labelSmall,
              ),
              for (final badge in contact.attentionBadges)
                Chip(visualDensity: VisualDensity.compact, label: Text(badge)),
            ],
          ),
        ],
      ),
      trailing: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            contact.micAvailable ? Icons.mic : Icons.mic_off,
            size: 18,
            color: contact.micAvailable
                ? Theme.of(context).colorScheme.primary
                : Theme.of(context).disabledColor,
          ),
          if (contact.latestAt != null)
            Text(
              DateFormat.Hm().format(contact.latestAt!),
              style: Theme.of(context).textTheme.labelSmall,
            ),
        ],
      ),
      onTap: onTap,
      onLongPress: onLongPress,
    );
  }

  String _previewLabel(NavivoxProfileContact contact) {
    if (contact.activeTurnState == 'streaming') return 'typing…';
    return contact.latestPreview;
  }

  String _workspaceLabel(NavivoxProfileContact contact) {
    if (!contact.workspaceRootsOk) return 'workspace issue';
    if (contact.workspaceRootCount == 1) return '1 root';
    return '${contact.workspaceRootCount} roots';
  }
}

class _ProfileAvatar extends StatelessWidget {
  const _ProfileAvatar({required this.contact});

  final NavivoxProfileContact contact;

  @override
  Widget build(BuildContext context) {
    final color =
        Colors.primaries[contact.avatarSeed.codeUnits.fold<int>(
              0,
              (a, b) => a + b,
            ) %
            Colors.primaries.length];
    return CircleAvatar(
      backgroundColor: color.shade700,
      foregroundColor: Colors.white,
      child: Text(contact.displayName.characters.first.toUpperCase()),
    );
  }
}

class _ServerChip extends StatelessWidget {
  const _ServerChip({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Chip(visualDensity: VisualDensity.compact, label: Text(label));
  }
}

class _HealthChip extends StatelessWidget {
  const _HealthChip({required this.health});

  final NavivoxProfileHealth health;

  @override
  Widget build(BuildContext context) {
    final label = switch (health) {
      NavivoxProfileHealth.online => 'online',
      NavivoxProfileHealth.offline => 'offline',
      NavivoxProfileHealth.needsAuth => 'auth',
      NavivoxProfileHealth.warning => 'warning',
    };
    return Text(label, style: Theme.of(context).textTheme.labelSmall);
  }
}
