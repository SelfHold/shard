import 'dart:typed_data';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../../core/i18n.dart';
import '../../core/theme/colors.dart';
import '../../core/theme/layout.dart';
import '../../core/theme/typography.dart';
import '../../data/rpc/core_client.dart';
import '../../data/rpc/gen/cloudsync.pbgrpc.dart' as pb;
import '../common/hm.dart';
import '../core/core_cubit.dart';

/// Group choice: the folders of the export become groups.
const _keep = '*keep';

/// The core reads at most this much (secrets.MaxImport).
const _maxFile = 32 << 20;

/// Imports an export of another password manager (KeePass, Chrome,
/// Firefox, Bitwarden, …). The core reads the file and shows what will be
/// added; nothing is saved before "Import". Returns how many were added.
Future<int?> showPasswordImport(BuildContext context, {required List<pb.SecretGroup> groups, String initialGroup = ''}) {
  final form = GlobalKey<_ImportState>();
  final action = ValueNotifier(const _Action('', busy: false, enabled: false));
  return showHmModal<int>(
    context,
    title: tr('Import passwords', 'Импорт паролей'),
    size: HmModalSize.lg,
    dismissible: false,
    body: (_) => _Import(key: form, groups: groups, initialGroup: initialGroup, action: action),
    actions: (ctx) => [
      HmButton(tr('Cancel', 'Отмена'), variant: HmVariant.neutral, expand: Hm.of(ctx).mobile, onPressed: () => Navigator.pop(ctx)),
      ValueListenableBuilder(
        valueListenable: action,
        builder: (_, a, _) => HmButton(
          a.label.isEmpty ? tr('Import', 'Импортировать') : a.label,
          variant: HmVariant.primary,
          loading: a.busy,
          expand: Hm.of(ctx).mobile,
          onPressed: a.enabled
              ? () async {
                  final n = await form.currentState?.primary();
                  if (n != null && ctx.mounted) Navigator.pop(ctx, n);
                }
              : null,
        ),
      ),
    ],
  );
}

/// The primary button of the dialog.
class _Action {
  const _Action(this.label, {required this.busy, required this.enabled});

  final String label;
  final bool busy;
  final bool enabled;
}

class _Import extends StatefulWidget {
  const _Import({super.key, required this.groups, required this.initialGroup, required this.action});

  final List<pb.SecretGroup> groups;
  final String initialGroup;
  final ValueNotifier<_Action> action;

  @override
  State<_Import> createState() => _ImportState();
}

class _ImportState extends State<_Import> {
  final _password = TextEditingController();
  late String _group = widget.groups.any((g) => g.id == widget.initialGroup) ? widget.initialGroup : _keep;
  String _fileName = '';
  Uint8List? _data;
  String _keyName = '';
  Uint8List? _key;
  bool _needPassword = false;
  bool _busy = false;
  String? _error;
  pb.SecretImportReply? _preview;
  int _seq = 0;

  late final CoreCubit _core;

  @override
  void initState() {
    super.initState();
    _core = context.read<CoreCubit>();
    WidgetsBinding.instance.addPostFrameCallback((_) => _sync());
  }

  @override
  void dispose() {
    _forget();
    _password.clear();
    _password.dispose();
    super.dispose();
  }

  /// The export holds every password in plain text: drop it once done.
  void _forget() {
    _data?.fillRange(0, _data!.length, 0);
    _key?.fillRange(0, _key!.length, 0);
    _data = _key = null;
    _preview?.clear();
  }

  void _sync() {
    final p = _preview;
    final label = p != null
        ? tr('Import ${p.added}', 'Импортировать ${p.added}')
        : _needPassword
        ? tr('Open', 'Открыть')
        : tr('Import', 'Импортировать');
    final enabled = !_busy && (p != null ? p.added > 0 : _needPassword && (_password.text.isNotEmpty || _key != null));
    widget.action.value = _Action(label, busy: _busy, enabled: enabled);
  }

  void _set(VoidCallback f) {
    if (!mounted) return;
    setState(f);
    _sync();
  }

  Future<void> _pickFile() async {
    final r = await FilePicker.platform.pickFiles(
      withData: true,
      dialogTitle: tr('Export of the password manager', 'Экспорт менеджера паролей'),
    );
    final f = r?.files.firstOrNull;
    if (f == null) return;
    if (f.size > _maxFile || f.bytes == null) {
      _set(
        () => _error = f.bytes == null
            ? tr('The file cannot be read', 'Файл не удалось прочитать')
            : tr('The file is larger than 32 MiB', 'Файл больше 32 МиБ'),
      );
      return;
    }
    _forget();
    _password.clear();
    _set(() {
      _fileName = f.name;
      _data = f.bytes;
      _keyName = '';
      _needPassword = false;
      _preview = null;
      _error = null;
    });
    await _read();
  }

  Future<void> _pickKey() async {
    final r = await FilePicker.platform.pickFiles(withData: true, dialogTitle: tr('KeePass key file', 'Ключевой файл KeePass'));
    final f = r?.files.firstOrNull;
    if (f?.bytes == null) return;
    _key?.fillRange(0, _key!.length, 0);
    _set(() {
      _key = f!.bytes;
      _keyName = f.name;
    });
  }

  pb.SecretImportRequest _request({required bool dryRun}) => pb.SecretImportRequest(
    data: _data,
    fileName: _fileName,
    password: _password.text,
    keyFile: _key,
    groupId: _group == _keep ? '' : _group,
    keepGroups: _group == _keep,
    dryRun: dryRun,
  );

  /// Asks the core what the file holds and what would be added.
  Future<void> _read() async {
    if (_data == null) return;
    final seq = ++_seq;
    _set(() => _busy = true);
    final req = _request(dryRun: true);
    try {
      final r = await _core.api.importSecrets(req);
      if (seq != _seq) return;
      _set(() {
        _needPassword = r.needPassword;
        _preview = r.needPassword ? null : r;
        _error = null;
      });
    } catch (e) {
      if (seq == _seq) _set(() => _error = errorText(e));
    } finally {
      req.clear();
      if (seq == _seq) _set(() => _busy = false);
    }
  }

  /// The primary button: open a protected file, or import.
  Future<int?> primary() async {
    if (_preview == null) {
      await _read();
      return null;
    }
    _set(() => _busy = true);
    final req = _request(dryRun: false);
    try {
      final r = await _core.api.importSecrets(req);
      Toasts.success(
        tr('Imported: ${r.added}', 'Импортировано: ${r.added}'),
        r.duplicates > 0 ? tr('Already in the safe: ${r.duplicates}', 'Уже были в сейфе: ${r.duplicates}') : null,
      );
      _forget();
      return r.added;
    } catch (e) {
      _set(() => _error = errorText(e));
      return null;
    } finally {
      req.clear();
      _set(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final hm = Hm.of(context);
    final gap = SizedBox(height: hm.gapSm);
    final p = _preview;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        HmFormField(
          label: tr('File', 'Файл'),
          required: true,
          hint: tr(
            'KeePass / KeePassXC (.kdbx, XML or CSV), Chrome, Edge, Brave, Opera, Firefox, Safari, Bitwarden '
                '(CSV or unencrypted JSON), 1Password, LastPass, Dashlane, Proton Pass, NordPass (CSV). '
                'Delete the export file afterwards: it holds the passwords unencrypted.',
            'KeePass / KeePassXC (.kdbx, XML или CSV), Chrome, Edge, Brave, Opera, Firefox, Safari, Bitwarden '
                '(CSV или незашифрованный JSON), 1Password, LastPass, Dashlane, Proton Pass, NordPass (CSV). '
                'Затем удалите файл экспорта: в нём пароли не зашифрованы.',
          ),
          child: Row(
            children: [
              HmButton(
                _fileName.isEmpty ? tr('Choose file…', 'Выбрать файл…') : tr('Another file…', 'Другой файл…'),
                icon: Icons.upload_file_outlined,
                onPressed: _busy ? null : _pickFile,
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  _fileName.isEmpty ? tr('No file chosen', 'Файл не выбран') : _fileName,
                  style: HmText.small,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          ),
        ),
        if (_needPassword) ...[
          gap,
          HmTextField(
            label: tr('KeePass master password', 'Мастер-пароль KeePass'),
            controller: _password,
            password: true,
            autofocus: true,
            onChanged: (_) => _sync(),
            onSubmitted: (_) => _read(),
          ),
          gap,
          Row(
            children: [
              HmButton(tr('Key file…', 'Ключевой файл…'), icon: Icons.key_outlined, small: true, onPressed: _busy ? null : _pickKey),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  _keyName.isEmpty ? tr('Only if the database uses one', 'Только если он используется') : _keyName,
                  style: HmText.small,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          ),
        ],
        if (_error != null) ...[gap, HmBanner(_error!, tone: HmTone.error)],
        if (p != null) ...[
          gap,
          HmSelect<String>(
            label: tr('Group', 'Группа'),
            options: {
              _keep: tr('Folders of the file as groups', 'Папки файла как группы'),
              '': tr('No group', 'Без группы'),
              for (final g in widget.groups) g.id: g.name,
            },
            value: _group,
            onChanged: (v) {
              _set(() => _group = v);
              _read();
            },
            hint: p.newGroups.isEmpty ? null : tr('New groups: ${p.newGroups.join(', ')}', 'Новые группы: ${p.newGroups.join(', ')}'),
          ),
          gap,
          HmBanner(
            [
              p.format,
              tr('to add: ${p.added}', 'добавится: ${p.added}'),
              if (p.duplicates > 0) tr('already in the safe: ${p.duplicates}', 'уже в сейфе: ${p.duplicates}'),
              if (p.failed > 0) tr('cannot be imported: ${p.failed}', 'не импортируются: ${p.failed}'),
            ].join(' · '),
            tone: p.added > 0 ? HmTone.info : HmTone.warning,
          ),
          gap,
          HmCard(
            recessed: true,
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            child: ConstrainedBox(
              constraints: BoxConstraints(maxHeight: hm.mobile ? 260 : 320),
              child: ListView.builder(shrinkWrap: true, itemCount: p.items.length, itemBuilder: (_, i) => _Row(p.items[i])),
            ),
          ),
        ],
      ],
    );
  }
}

class _Row extends StatelessWidget {
  const _Row(this.it);

  final pb.SecretImportItem it;

  @override
  Widget build(BuildContext context) {
    final (icon, color) = switch (it.status) {
      pb.SecretImportStatus.IMPORT_DUPLICATE => (Icons.done_all, HmColors.muted),
      pb.SecretImportStatus.IMPORT_FAILED => (Icons.error_outline, HmColors.error),
      _ => (it.kind == pb.SecretKind.SECRET_TOTP ? Icons.pin_outlined : Icons.password_outlined, HmColors.accent),
    };
    final sub = [
      if (it.username.isNotEmpty) it.username,
      if (it.kind == pb.SecretKind.SECRET_TOTP) tr('2FA code', '2FA-код'),
      if (it.status == pb.SecretImportStatus.IMPORT_DUPLICATE) tr('already in the safe', 'уже в сейфе'),
    ].join(' · ');
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 5),
      child: Row(
        children: [
          Icon(icon, size: 16, color: color),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(it.title, maxLines: 1, overflow: TextOverflow.ellipsis),
                if (it.error.isNotEmpty)
                  Text(it.error, style: HmText.small.copyWith(color: HmColors.error))
                else if (sub.isNotEmpty)
                  Text(sub, style: HmText.small, maxLines: 1, overflow: TextOverflow.ellipsis),
              ],
            ),
          ),
          if (it.group.isNotEmpty) ...[const SizedBox(width: 8), HmBadge(it.group)],
        ],
      ),
    );
  }
}
