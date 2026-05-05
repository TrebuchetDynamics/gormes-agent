import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/app.dart';
import 'package:navivox/features/voice/widgets/voice_morph_surface.dart';

void main() {
  testWidgets('router starts at setup when no fake server exists', (
    tester,
  ) async {
    await tester.pumpWidget(const NavivoxApp());
    await tester.pumpAndSettle();

    expect(find.text('Set up Navivox'), findsOneWidget);
    expect(find.text('Use fake local server'), findsOneWidget);
  });

  testWidgets('can enter fake server mode and show chat', (tester) async {
    await tester.pumpWidget(const NavivoxApp());
    await tester.pumpAndSettle();

    await tester.tap(find.text('Use fake local server'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 250));

    expect(find.text('Fake Local Gormes'), findsOneWidget);
    expect(find.text('Server online'), findsOneWidget);
    expect(find.text('Tool call'), findsOneWidget);
    expect(find.text('Voice message'), findsOneWidget);
    expect(find.byType(VoiceMorphSurface), findsOneWidget);
  });
}
