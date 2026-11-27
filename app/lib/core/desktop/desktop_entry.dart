import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:path/path.dart' as p;

/// GTK application id (linux/CMakeLists.txt APPLICATION_ID). GNOME on
/// Wayland matches windows to `<id>.desktop` for the taskbar icon.
const _appId = 'dev.selfhold.shard';

/// Registers the app with the Linux desktop (menu entry and taskbar icon):
/// writes `~/.local/share/applications/<id>.desktop` pointing at this build
/// and copies the icon next to it. Only release builds do it, so a debug
/// run does not take over the entry. Rewrites only when something changed.
///
/// A Linux package installs the entry and icon system-wide. Then nothing is
/// written, and the copies an earlier unpackaged run left are removed:
/// user-level files take precedence and would hide the packaged ones.
Future<void> registerDesktopEntry() async {
  if (!Platform.isLinux || !kReleaseMode) return;
  try {
    final env = Platform.environment;
    final xdg = env['XDG_DATA_HOME'];
    final data = xdg != null && xdg.isNotEmpty ? xdg : p.join(env['HOME'] ?? '', '.local', 'share');
    final userEntry = File(p.join(data, 'applications', '$_appId.desktop'));
    final userIcon = File(p.join(data, 'icons', 'hicolor', '256x256', 'apps', '$_appId.png'));
    if (await _installedSystemWide(env)) {
      for (final f in [userEntry, userIcon]) {
        if (await f.exists()) await f.delete();
      }
      return;
    }
    final exe = Platform.resolvedExecutable;
    final src = File(p.join(p.dirname(exe), 'data', 'flutter_assets', 'assets', 'icon', 'shard.png'));
    if (!await src.exists()) return;

    final bytes = await src.readAsBytes();
    if (!await userIcon.exists() || !listEquals(await userIcon.readAsBytes(), bytes)) {
      await userIcon.parent.create(recursive: true);
      await userIcon.writeAsBytes(bytes, flush: true);
    }

    final text = [
      '[Desktop Entry]',
      'Type=Application',
      'Name=Shard',
      'Comment=Encrypted file sync over several cloud storages',
      'Exec="$exe"',
      'Icon=${userIcon.path}',
      'Terminal=false',
      'Categories=Network;FileTransfer;',
      'StartupWMClass=$_appId',
      'StartupNotify=true',
      '',
    ].join('\n');
    if (!await userEntry.exists() || await userEntry.readAsString() != text) {
      await userEntry.parent.create(recursive: true);
      await userEntry.writeAsString(text, flush: true);
    }
  } catch (_) {
    // Cosmetic only: the app works without it.
  }
}

/// Whether a package installed `<id>.desktop` into one of the system data
/// dirs (XDG Base Directory spec default: /usr/local/share:/usr/share).
Future<bool> _installedSystemWide(Map<String, String> env) async {
  final dirs = env['XDG_DATA_DIRS'];
  final list = dirs != null && dirs.isNotEmpty ? dirs.split(':') : const ['/usr/local/share', '/usr/share'];
  for (final dir in list) {
    if (dir.isEmpty) continue;
    if (await File(p.join(dir, 'applications', '$_appId.desktop')).exists()) return true;
  }
  return false;
}
