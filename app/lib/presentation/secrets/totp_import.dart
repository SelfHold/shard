import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../../core/i18n.dart';
import '../../core/theme/colors.dart';
import '../../core/theme/layout.dart';
import '../../core/theme/typography.dart';
import '../../data/rpc/core_client.dart';
import '../../data/rpc/gen/cloudsync.pbgrpc.dart' as pb;
import '../common/hm.dart';
import '../core/core_cubit.dart';
import 'totp.dart' show formatTotp;

/// Imports 2FA entries from otpauth://totp/ links, one a line. A link in
/// the clipboard is taken right away. Returns how many were saved.
Future<int?> showTotpImport(BuildContext context, {required List<pb.SecretGroup> groups, String initialGroup = ''}) {
  final form = GlobalKey<_ImportState>();
  final saving = ValueNotifier(false);
  return showHmModal<int>(
    context,
    title: tr('Import from link', 'Импорт из ссылки'),
    size: HmModalSize.md,
    dismissible: false,
    body: (_) => _Import(key: form, groups: groups, initialGroup: initialGroup, saving: saving),
    actions: (ctx) => [
      HmButton(tr('Cancel', 'Отмена'), variant: HmVariant.neutral, expand: Hm.of(ctx).mobile, onPressed: () => Navigator.pop(ctx)),
      ValueListenableBuilder(
        valueListenable: saving,
        builder: (_, busy, _) => HmButton(
          tr('Import', 'Импортировать'),
          variant: HmVariant.primary,
          loading: busy,
          expand: Hm.of(ctx).mobile,
          onPressed: () async {
            final n = await form.currentState?.save();
            if (n != null && ctx.mounted) Navigator.pop(ctx, n);
          },
        ),
      ),
    ],
  );
}

/// Whether a text holds an otpauth:// or otpauth-migration:// link.
bool _hasLink(String text) => text.toLowerCase().contains('otpauth');

class _Import extends StatefulWidget {
  const _Import({super.key, required this.groups, required this.initialGroup, required this.saving});

  final List<pb.SecretGroup> groups;
  final String initialGroup;
  final ValueNotifier<bool> saving;

  @override
  State<_Import> createState() => _ImportState();
}

class _ImportState extends State<_Import> {
  final _text = TextEditingController();
  late String _group = widget.groups.any((g) => g.id == widget.initialGroup) ? widget.initialGroup : '';
  pb.TotpLinksReply _read = pb.TotpLinksReply();
  String? _error;
  String? _fromClipboard; // the clipboard text the links came from
  Timer? _timer;
  int _seq = 0;

  late final CoreCubit _core;

  @override
  void initState() {
    super.initState();
    _core = context.read<CoreCubit>();
    _pasteClipboard();
  }

  @override
  void dispose() {
    _timer?.cancel();
    _text.clear();
    _text.dispose();
    super.dispose();
  }

  Future<void> _pasteClipboard() async {
    final clip = (await Clipboard.getData(Clipboard.kTextPlain))?.text ?? '';
    if (!mounted || _text.text.isNotEmpty || !_hasLink(clip)) return;
    _fromClipboard = clip;
    _text.text = clip.trim();
    _check();
  }

  void _checkSoon() {
    _timer?.cancel();
    _timer = Timer(const Duration(milliseconds: 300), _check);
  }

  /// Asks the core to find the links (and expand Google Authenticator
  /// exports): what will be saved and the code now.
  Future<void> _check() async {
    final seq = ++_seq;
    try {
      final r = await _core.api.readTotpLinks(pb.TotpLinksRequest(text: _text.text));
      if (mounted && seq == _seq) setState(() => _read = r);
    } catch (e) {
      if (mounted && seq == _seq) setState(() => _error = errorText(e));
    }
  }

  /// Parts of a Google Authenticator export that were not pasted.
  String? get _missingParts {
    final byId = <int, (int, Set<int>)>{};
    for (final b in _read.batches) {
      final got = byId[b.id]?.$2 ?? <int>{};
      byId[b.id] = (b.size, got..add(b.index));
    }
    final missing = [
      for (final (size, got) in byId.values)
        for (var i = 0; i < size; i++)
          if (!got.contains(i)) i + 1,
    ];
    if (missing.isEmpty) return null;
    final total = byId.values.map((v) => v.$1).fold(0, (a, b) => a > b ? a : b);
    return tr(
      'The export has $total QR codes; parts ${missing.join(', ')} are not pasted. Add their links too.',
      'В экспорте $total QR-кодов; части ${missing.join(', ')} не вставлены. Добавьте и их ссылки.',
    );
  }

  Future<int?> save() async {
    await _check();
    final ok = _read.links.where((l) => l.link.isNotEmpty).toList();
    if (ok.isEmpty) {
      setState(
        () => _error = _text.text.trim().isEmpty
            ? tr('Paste an otpauth://totp/ link', 'Вставьте ссылку otpauth://totp/')
            : tr('No link could be read', 'Ни одну ссылку не удалось прочитать'),
      );
      return null;
    }
    setState(() => _error = null);
    widget.saving.value = true;
    final done = <String>[];
    try {
      for (final e in ok) {
        final p = e.preview;
        final req = pb.SecretEntry(
          kind: pb.SecretKind.SECRET_TOTP,
          // The core takes the title and the account from the link.
          title: p.issuer.isNotEmpty || p.account.isNotEmpty ? '' : tr('2FA code', '2FA-код'),
          groupId: _group,
          totpSecret: e.link,
        );
        try {
          await _core.api.putSecret(req);
          done.add(e.link);
        } finally {
          req.clear();
        }
      }
    } catch (err) {
      // What was saved leaves the field, so trying again adds no copies.
      if (mounted) {
        setState(() {
          _error = errorText(err);
          // Exports are left as the links of their unsaved accounts.
          _text.text = ok.map((l) => l.link).where((l) => !done.contains(l)).join('\n');
        });
        _check();
      }
    } finally {
      widget.saving.value = false;
    }
    final saved = done.length;
    if (saved > 0) {
      // The link holds the secret: do not leave it in the clipboard.
      final clip = (await Clipboard.getData(Clipboard.kTextPlain))?.text;
      if (_fromClipboard != null && clip == _fromClipboard) await Clipboard.setData(const ClipboardData(text: ''));
      Toasts.success(tr('Imported: $saved', 'Импортировано: $saved'));
    }
    return saved == ok.length ? saved : null;
  }

  @override
  Widget build(BuildContext context) {
    final hm = Hm.of(context);
    final gap = SizedBox(height: hm.gapSm);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        HmTextField(
          label: tr('Links', 'Ссылки'),
          required: true,
          autofocus: true,
          controller: _text,
          identifier: true,
          lines: 4,
          error: _error,
          placeholder: 'otpauth://totp/Issuer:account?secret=…&issuer=Issuer',
          hint: tr(
            'The link behind the QR code, several links one a line, or Google Authenticator export links '
                '(otpauth-migration://). Taken from the clipboard if it holds one.',
            'Ссылка из QR-кода, несколько ссылок по одной на строку или ссылки экспорта Google Authenticator '
                '(otpauth-migration://). Берётся из буфера обмена, если она там есть.',
          ),
          onChanged: (_) => _checkSoon(),
        ),
        gap,
        HmSelect<String>(
          label: tr('Group', 'Группа'),
          options: {'': tr('No group', 'Без группы'), for (final g in widget.groups) g.id: g.name},
          value: _group,
          onChanged: (v) => setState(() => _group = v),
        ),
        if (_missingParts != null) ...[gap, HmBanner(_missingParts!, tone: HmTone.warning)],
        if (_read.links.isNotEmpty) ...[
          gap,
          HmCard(
            recessed: true,
            padding: const EdgeInsets.all(12),
            child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [for (final l in _read.links) _PreviewRow(l.preview)]),
          ),
        ],
      ],
    );
  }
}

class _PreviewRow extends StatelessWidget {
  const _PreviewRow(this.p);

  final pb.TotpPreview p;

  @override
  Widget build(BuildContext context) {
    final name = [p.issuer, p.account].where((s) => s.isNotEmpty).join(' · ');
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          Icon(
            p.error.isEmpty ? Icons.pin_outlined : Icons.error_outline,
            size: 16,
            color: p.error.isEmpty ? HmColors.accent : HmColors.error,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: p.error.isNotEmpty
                ? Text(p.error, style: HmText.small.copyWith(color: HmColors.error))
                : Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        name.isEmpty ? tr('No name in the link', 'В ссылке нет имени') : name,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                      Text('${p.algorithm} · ${p.digits} · ${tr('${p.period} s', '${p.period} с')}', style: HmText.small),
                    ],
                  ),
          ),
          if (p.error.isEmpty) Text(formatTotp(p.code), style: HmText.jb(14, weight: 700)),
        ],
      ),
    );
  }
}
