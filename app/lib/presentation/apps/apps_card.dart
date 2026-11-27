import 'dart:async';

import 'package:fixnum/fixnum.dart';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../../core/format.dart';
import '../../core/i18n.dart';
import '../../core/theme/layout.dart';
import '../../core/theme/typography.dart';
import '../../data/rpc/core_client.dart';
import '../../data/rpc/gen/cloudsync.pbgrpc.dart' as pb;
import '../common/hm.dart';
import '../core/core_cubit.dart';

/// Connected apps (docs/CONNECTOR.md): other applications that keep their
/// settings in the vault. Shows pairing requests, the apps paired on this
/// device and what each app stores (on every device).
class AppsCard extends StatefulWidget {
  const AppsCard({super.key});

  @override
  State<AppsCard> createState() => _AppsCardState();
}

class _AppsCardState extends State<AppsCard> {
  late final CoreCubit _core;
  pb.AppsReply? _apps;
  String? _error;
  int _rev = -1;

  @override
  void initState() {
    super.initState();
    _core = context.read<CoreCubit>();
    unawaited(_load());
  }

  Future<void> _load() async {
    _rev = _core.state.appsRev + _core.state.treeRev;
    try {
      final r = await _core.api.listApps(pb.Empty());
      if (mounted) {
        setState(() {
          _apps = r;
          _error = null;
        });
      }
    } catch (e) {
      if (mounted) setState(() => _error = errorText(e));
    }
  }

  Future<void> _run(Future<void> Function() fn, String failed) async {
    try {
      await fn();
    } catch (e) {
      Toasts.error(failed, errorText(e));
    }
    await _load();
  }

  Future<void> _revoke(pb.AppPairing p) async {
    final ok = await confirm(
      context,
      title: tr('Disconnect ${p.appName}?', 'Отключить ${p.appName}?'),
      message: tr(
        'The app loses access on this device. Its data stays in the vault; pair again to use it.',
        'Приложение теряет доступ на этом устройстве. Его данные остаются в vault; для работы нужно сопряжение заново.',
      ),
      action: tr('Disconnect', 'Отключить'),
      destructive: true,
    );
    if (ok) await _run(() => _core.api.revokeApp(pb.IdRequest(id: p.id)), tr('Could not disconnect', 'Не удалось отключить'));
  }

  Future<void> _dropScope(pb.AppPairing p, SecretScope sc) async {
    final ok = await confirm(
      context,
      title: sc.dropTitle(p.appName),
      message: sc.dropMessage(),
      action: tr('Take access', 'Забрать доступ'),
      destructive: true,
    );
    if (!ok) return;
    final keep = p.scopes.where((s) => s != sc.id).toList();
    await _run(
      () => _core.api.setAppScopes(pb.SetAppScopesRequest(id: p.id, scopes: keep)),
      tr('Could not change the permissions', 'Не удалось изменить разрешения'),
    );
  }

  Future<void> _deleteData(pb.AppDataInfo d) async {
    final ok = await confirm(
      context,
      title: tr('Delete data of ${d.appId}?', 'Удалить данные ${d.appId}?'),
      message: tr(
        'The app\'s records are deleted on all devices. An app that keeps its settings only here loses them.',
        'Записи приложения удаляются на всех устройствах. Приложение, которое хранит настройки только здесь, их потеряет.',
      ),
      action: tr('Delete', 'Удалить'),
      destructive: true,
    );
    if (ok) await _run(() => _core.api.deleteAppData(pb.IdRequest(id: d.appId)), tr('Could not delete', 'Не удалось удалить'));
  }

  @override
  Widget build(BuildContext context) {
    final hm = Hm.of(context);
    final core = context.watch<CoreCubit>().state;
    if (core.appsRev + core.treeRev != _rev) unawaited(_load());
    final apps = _apps;
    final gap = SizedBox(height: hm.gapSm);
    return HmCard(
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          HmSectionTitle(tr('Connected apps', 'Подключённые приложения')),
          gap,
          Text(
            tr(
              'Apps such as MegaChat can keep their settings here: encrypted, in every storage, the same on all devices. An app asks once per device; you allow it here.',
              'Приложения (например, MegaChat) могут хранить здесь свои настройки: зашифрованными, во всех хранилищах, одинаковыми на всех устройствах. Приложение просит доступ один раз на устройстве; разрешение даётся здесь.',
            ),
            style: HmText.small.copyWith(height: 1.5),
          ),
          if (_error != null) ...[gap, HmBanner(_error!, tone: HmTone.error)],
          if (apps != null) ...[
            for (final q in apps.requests) ...[gap, _RequestTile(request: q, onDone: _load)],
            if (apps.pairings.isEmpty && apps.requests.isEmpty) ...[
              gap,
              Text(tr('No apps are paired on this device.', 'На этом устройстве нет сопряжённых приложений.'), style: HmText.small),
            ],
            for (final p in apps.pairings) ...[
              gap,
              _Row(
                title: p.appName.isEmpty ? p.appId : p.appName,
                lines: [
                  [p.appId, platformName(p.platform), if (p.package.isNotEmpty) p.package].join(' · '),
                  tr('paired ', 'сопряжено ') + _date(p.approvedUnixMs) + _lastUsed(p.lastUsedUnixMs),
                  for (final sc in secretScopes)
                    if (p.scopes.contains(sc.id))
                      sc.grantedLine() +
                          (sc.lastRead(p) == Int64.ZERO
                              ? tr(' · has not read them yet', ' · ещё не читало их')
                              : tr(' · last read ', ' · последнее чтение ') + _date(sc.lastRead(p))),
                ],
                action: Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    for (final sc in secretScopes)
                      if (p.scopes.contains(sc.id))
                        HmButton(
                          sc.dropButton(),
                          small: true,
                          variant: HmVariant.neutral,
                          icon: Icons.key_off,
                          onPressed: () => _dropScope(p, sc),
                        ),
                    HmButton(
                      tr('Disconnect', 'Отключить'),
                      small: true,
                      variant: HmVariant.red,
                      icon: Icons.link_off,
                      onPressed: () => _revoke(p),
                    ),
                  ],
                ),
              ),
            ],
            if (apps.data.isNotEmpty) ...[
              const SizedBox(height: 16),
              HmSectionTitle(tr('Stored by apps', 'Хранят приложения')),
              for (final d in apps.data) ...[
                gap,
                _Row(
                  title: d.appId,
                  lines: [tr('${d.records} records · ', 'записей: ${d.records} · ') + formatSize(d.bytes.toInt())],
                  action: HmButton(
                    tr('Delete data', 'Удалить данные'),
                    small: true,
                    variant: HmVariant.neutral,
                    icon: Icons.delete_outline,
                    onPressed: () => _deleteData(d),
                  ),
                ),
              ],
            ],
          ],
        ],
      ),
    );
  }
}

String _date(Int64 ms) => ms == Int64.ZERO ? '—' : formatTime(Int64(ms.toInt() ~/ 1000));

String _lastUsed(Int64 ms) => ms == Int64.ZERO ? '' : tr(', used ', ', использовалось ') + _date(ms);

class _Row extends StatelessWidget {
  const _Row({required this.title, required this.lines, required this.action});

  final String title;
  final List<String> lines;
  final Widget action;

  @override
  Widget build(BuildContext context) {
    final text = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(title, style: HmText.bodyStrong),
        for (final l in lines) Text(l, style: HmText.small),
      ],
    );
    if (Hm.of(context).mobile) {
      return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [text, const SizedBox(height: 8), action]);
    }
    return Row(
      children: [
        Expanded(child: text),
        const SizedBox(width: 12),
        action,
      ],
    );
  }
}

/// A pairing request: the code to compare and the two answers.
class _RequestTile extends StatefulWidget {
  const _RequestTile({required this.request, required this.onDone});

  final pb.PairingRequestInfo request;
  final Future<void> Function() onDone;

  @override
  State<_RequestTile> createState() => _RequestTileState();
}

class _RequestTileState extends State<_RequestTile> {
  bool _busy = false;

  Future<void> _answer(PairingAnswer answer) async {
    setState(() => _busy = true);
    await decidePairing(context.read<CoreCubit>(), widget.request, answer);
    if (mounted) setState(() => _busy = false);
    await widget.onDone();
  }

  @override
  Widget build(BuildContext context) {
    final hm = Hm.of(context);
    return HmCard(
      recessed: true,
      padding: const EdgeInsets.all(12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          PairingDetails(request: widget.request),
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            alignment: hm.mobile ? WrapAlignment.start : WrapAlignment.end,
            children: [
              for (final a in pairingAnswers(widget.request))
                HmButton(
                  a.label,
                  small: true,
                  variant: a.variant,
                  loading: _busy && a.answer == PairingAnswer.all,
                  onPressed: _busy ? null : () => _answer(a.answer),
                ),
            ],
          ),
        ],
      ),
    );
  }
}

/// What the user needs to decide: who asks and the code the app shows.
class PairingDetails extends StatelessWidget {
  const PairingDetails({super.key, required this.request});

  final pb.PairingRequestInfo request;

  @override
  Widget build(BuildContext context) {
    final q = request;
    final code = q.code.length == 6 ? '${q.code.substring(0, 3)} ${q.code.substring(3)}' : q.code;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          tr('${q.appName} asks to keep its data in CloudSync', '${q.appName} просит хранить свои данные в CloudSync'),
          style: HmText.bodyStrong,
        ),
        for (final sc in secretScopes)
          if (q.scopes.contains(sc.id)) ...[const SizedBox(height: 8), HmBanner(sc.asks(q.appName), tone: HmTone.error)],
        const SizedBox(height: 4),
        Text(
          [q.appId, platformName(q.platform), if (q.package.isNotEmpty) q.package, if (q.description.isNotEmpty) q.description].join(' · '),
          style: HmText.small,
        ),
        const SizedBox(height: 8),
        Row(
          children: [
            Text(tr('Code: ', 'Код: '), style: HmText.small),
            SelectableText(code, style: HmText.statValue),
          ],
        ),
        const SizedBox(height: 4),
        Text(
          asksSecrets(q)
              ? tr(
                  'Allow only if the app shows the same code. "Allow all" also gives it what is listed above (never notes, other keys or files); "Records only" keeps it to its own data. You can take each permission away later.',
                  'Разрешайте, только если приложение показывает тот же код. «Разрешить всё» даёт ему ещё и то, что перечислено выше (но не заметки, другие ключи и файлы); «Только записи» оставляет доступ лишь к его данным. Каждое разрешение можно потом забрать.',
                )
              : tr(
                  'Allow only if the app shows the same code. It gets access to its own records only, not to files or passwords.',
                  'Разрешайте, только если приложение показывает тот же код. Оно получит доступ лишь к своим записям, не к файлам и паролям.',
                ),
          style: HmText.small.copyWith(height: 1.5),
        ),
        if (q.alreadyPaired) ...[
          const SizedBox(height: 8),
          HmBanner(
            tr(
              'This app is already paired here; allowing replaces that pairing if it is the same app.',
              'Это приложение уже сопряжено здесь; разрешение заменит прежнее сопряжение, если это то же приложение.',
            ),
            tone: HmTone.warning,
          ),
        ],
      ],
    );
  }
}

/// A scope that opens part of the password safe to an app
/// (docs/CONNECTOR.md § 5).
class SecretScope {
  const SecretScope(this.id);

  final String id;

  /// The plain words of the pairing dialog.
  String asks(String app) => switch (id) {
    'passwords' => tr('$app wants to read and change all your website passwords', '$app хочет читать и менять все ваши пароли от сайтов'),
    'ssh_keys' => tr('$app wants to use your SSH keys', '$app хочет использовать ваши SSH-ключи'),
    _ => tr('$app wants to read and change your servers', '$app хочет читать и менять ваши серверы'),
  };

  String grantedLine() => switch (id) {
    'passwords' => tr('can read and change all website passwords', 'может читать и менять все пароли от сайтов'),
    'ssh_keys' => tr('can use your SSH keys', 'может использовать ваши SSH-ключи'),
    _ => tr('can read and change your servers', 'может читать и менять ваши серверы'),
  };

  String dropButton() => switch (id) {
    'passwords' => tr('Take password access', 'Забрать доступ к паролям'),
    'ssh_keys' => tr('Take SSH key access', 'Забрать доступ к SSH-ключам'),
    _ => tr('Take server access', 'Забрать доступ к серверам'),
  };

  String dropTitle(String app) => switch (id) {
    'passwords' => tr('Take password access from $app?', 'Забрать у $app доступ к паролям?'),
    'ssh_keys' => tr('Take SSH key access from $app?', 'Забрать у $app доступ к SSH-ключам?'),
    _ => tr('Take server access from $app?', 'Забрать у $app доступ к серверам?'),
  };

  String dropMessage() => switch (id) {
    'passwords' => tr(
      'The app stays connected and keeps its own records, but can no longer read or change your passwords on this device.',
      'Приложение остаётся подключённым и сохраняет свои записи, но больше не может читать и менять ваши пароли на этом устройстве.',
    ),
    'ssh_keys' => tr(
      'The app stays connected and keeps its own records, but can no longer read your SSH keys on this device.',
      'Приложение остаётся подключённым и сохраняет свои записи, но больше не может читать ваши SSH-ключи на этом устройстве.',
    ),
    _ => tr(
      'The app stays connected and keeps its own records, but can no longer read or change your servers on this device.',
      'Приложение остаётся подключённым и сохраняет свои записи, но больше не может читать и менять ваши серверы на этом устройстве.',
    ),
  };

  /// When the app last read the secrets of this scope (0 = never).
  Int64 lastRead(pb.AppPairing p) => switch (id) {
    'passwords' => p.secretsReadUnixMs,
    'ssh_keys' => p.keysReadUnixMs,
    _ => p.serversReadUnixMs,
  };
}

/// The scopes that open the safe, in the order they are shown.
const secretScopes = [SecretScope('passwords'), SecretScope('ssh_keys'), SecretScope('servers')];

/// Whether a request asks for any part of the safe.
bool asksSecrets(pb.PairingRequestInfo q) => secretScopes.any((sc) => q.scopes.contains(sc.id));

/// How the user answers a pairing request.
enum PairingAnswer { refuse, records, all }

/// The buttons for a request, least access first: an app that asks for
/// passwords may still be allowed its own records only.
List<({String label, HmVariant variant, PairingAnswer answer})> pairingAnswers(pb.PairingRequestInfo q) => [
  (label: tr('Refuse', 'Отклонить'), variant: HmVariant.neutral, answer: PairingAnswer.refuse),
  if (asksSecrets(q)) ...[
    (label: tr('Records only', 'Только записи'), variant: HmVariant.neutral, answer: PairingAnswer.records),
    (label: tr('Allow all', 'Разрешить всё'), variant: HmVariant.red, answer: PairingAnswer.all),
  ] else
    (label: tr('Allow', 'Разрешить'), variant: HmVariant.primary, answer: PairingAnswer.all),
];

Future<void> decidePairing(CoreCubit core, pb.PairingRequestInfo q, PairingAnswer answer) async {
  final approve = answer != PairingAnswer.refuse;
  try {
    await core.api.decidePairing(
      pb.DecidePairingRequest(id: q.id, approve: approve, scopes: answer == PairingAnswer.all ? q.scopes : const <String>[]),
    );
    if (approve) Toasts.success(tr('${q.appName} is connected', '${q.appName} подключено'));
  } catch (e) {
    Toasts.error(tr('Could not answer the request', 'Не удалось ответить на запрос'), errorText(e));
  }
}

/// Asks the user about pairing requests as they come, anywhere in the app
/// while the vault is unlocked. Each request is asked once; unanswered ones
/// stay in Vault → Connected apps until they expire.
class PairingPrompt extends StatefulWidget {
  const PairingPrompt({super.key, required this.child});

  final Widget child;

  @override
  State<PairingPrompt> createState() => _PairingPromptState();
}

class _PairingPromptState extends State<PairingPrompt> {
  final _asked = <String>{};
  bool _open = false;

  Future<void> _check(CoreCubit core) async {
    if (_open || !core.state.unlocked || core.state.conn != Conn.ready) return;
    pb.AppsReply r;
    try {
      r = await core.api.listApps(pb.Empty());
    } catch (_) {
      return;
    }
    final q = r.requests.where((q) => !_asked.contains(q.id)).firstOrNull;
    if (q == null || !mounted) return;
    _asked.add(q.id);
    _open = true;
    final answer = await showHmModal<PairingAnswer>(
      context,
      title: tr('Connect an app?', 'Подключить приложение?'),
      size: HmModalSize.sm,
      body: (_) => PairingDetails(request: q),
      actions: (ctx) => [
        for (final a in pairingAnswers(q)) HmButton(a.label, variant: a.variant, onPressed: () => Navigator.pop(ctx, a.answer)),
      ],
    );
    _open = false;
    // Closed without an answer: the request waits in the Vault screen.
    if (answer != null) await decidePairing(core, q, answer);
    if (mounted) await _check(core);
  }

  @override
  Widget build(BuildContext context) => BlocListener<CoreCubit, CoreState>(
    listenWhen: (a, b) => a.appsRev != b.appsRev || a.unlocked != b.unlocked || a.conn != b.conn,
    listener: (context, _) => _check(context.read<CoreCubit>()),
    child: widget.child,
  );
}
