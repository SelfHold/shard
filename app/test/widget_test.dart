import 'dart:io';

import 'package:shard/app.dart';
import 'package:shard/data/rpc/endpoint.dart';
import 'package:shard/presentation/core/core_cubit.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

class _NoCore implements CoreLauncher {
  @override
  Future<Endpoint> endpoint({bool start = true}) async => throw const SocketException('no core in tests');
}

void main() {
  for (final size in const [Size(1280, 800), Size(360, 800)]) {
    testWidgets('shows the no-core screen at ${size.width.toInt()} px', (tester) async {
      tester.view.physicalSize = size;
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final core = CoreCubit(_NoCore());
      await tester.pumpWidget(ShardApp(core: core));
      await core.connect();
      await tester.pump();
      expect(find.text('No connection to the core'), findsWidgets);
      expect(find.text('Retry'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }
}
