# Profile-rooted database paths probe `profiles/main/` existence

`MemoryDBPath()` and `SessionDBPath()` probe whether `$GORMES_HOME/profiles/main/` already exists on disk and, if so, return paths under it instead of the `$GORMES_HOME` root. This bridges the gap between single-profile users (default, no materialised profile dir → root DBs, unchanged) and multi-profile setups where every profile, including "main", has isolated DBs under its profile root. The migration path is: `gormes setup profiles` materialises `profiles/main/`, and on next startup the DB paths auto-redirect.

**Status**: accepted

**Considered options**: (a) always write to root (simple but blocks profile isolation), (b) always write to `profiles/main/` (clean but breaks existing installs on upgrade), (c) probe `profiles/main/` existence and redirect only when materialised (chosen — zero-change for existing users, clean profile isolation when setup materialises the dir).

**Consequences**: The probe is a one-time stat call — not a directory-creation trigger — so stale installs without `profiles/main/` never accidentally create one. Migration tools must materialise the dir AND move the DB files; the path redirect is not a migration by itself.