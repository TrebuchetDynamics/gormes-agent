import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:navivox/core/protocol/frame_reader.dart';
import 'package:navivox/core/protocol/navivox_frame.dart';

void main() {
  group('NavivoxFrameReader', () {
    test('emits a single frame from a single byte chunk', () async {
      final controller = StreamController<Uint8List>();
      final reader = NavivoxFrameReader(controller.stream);
      final frames = <NavivoxFrame>[];
      final subscription = reader.frames.listen(frames.add);

      controller.add(_encodeText('chat.submit', 'm-1', 'hi'));
      await controller.close();
      await subscription.asFuture();

      expect(frames.length, 1);
      expect(frames.single.type, 'chat.submit');
      expect(frames.single.messageId, 'm-1');
    });

    test('reassembles a frame split across multiple chunks', () async {
      final wire = _encodeText('chat.submit', 'm-2', 'hello world');
      final controller = StreamController<Uint8List>();
      final reader = NavivoxFrameReader(controller.stream);
      final frames = <NavivoxFrame>[];
      final subscription = reader.frames.listen(frames.add);

      // Drip-feed the bytes one or two at a time.
      for (var i = 0; i < wire.length; i += 3) {
        final end = (i + 3).clamp(0, wire.length);
        controller.add(Uint8List.sublistView(wire, i, end));
      }
      await controller.close();
      await subscription.asFuture();

      expect(frames.length, 1);
      expect(frames.single.messageId, 'm-2');
      expect(utf8.decode(frames.single.payload), '{"text":"hello world"}');
    });

    test('emits multiple frames from a single combined chunk', () async {
      final a = _encodeText('chat.submit', 'a', 'one');
      final b = _encodeText('chat.message', 'b', 'two');
      final c = _encodeText('chat.final', 'c', 'three');
      final combined = Uint8List(a.length + b.length + c.length)
        ..setRange(0, a.length, a)
        ..setRange(a.length, a.length + b.length, b)
        ..setRange(a.length + b.length, a.length + b.length + c.length, c);

      final controller = StreamController<Uint8List>();
      final reader = NavivoxFrameReader(controller.stream);
      final frames = <NavivoxFrame>[];
      final subscription = reader.frames.listen(frames.add);

      controller.add(combined);
      await controller.close();
      await subscription.asFuture();

      expect(frames.map((f) => f.messageId).toList(), ['a', 'b', 'c']);
      expect(frames.map((f) => f.type).toList(), [
        'chat.submit',
        'chat.message',
        'chat.final',
      ]);
    });

    test(
      'exposes decode errors on the errors stream and keeps reading',
      () async {
        final controller = StreamController<Uint8List>();
        final reader = NavivoxFrameReader(controller.stream);
        final frames = <NavivoxFrame>[];
        final errors = <Object>[];
        final framesDone = Completer<void>();
        final framesSub = reader.frames.listen(
          frames.add,
          onDone: framesDone.complete,
        );
        final errorsSub = reader.errors.listen(errors.add);

        // Bad magic: 4 bytes that aren't NVOX.
        controller.add(Uint8List.fromList([0xff, 0xff, 0xff, 0xff]));
        // Then a valid frame on a fresh boundary.
        controller.add(_encodeText('chat.submit', 'after-error', 'ok'));
        await controller.close();
        await framesDone.future;
        await framesSub.cancel();
        await errorsSub.cancel();

        expect(errors.length, greaterThanOrEqualTo(1));
        expect(errors.first, isA<InvalidFrameException>());
        expect(frames.map((f) => f.messageId).toList(), ['after-error']);
      },
    );
  });
}

Uint8List _encodeText(String type, String id, String text) {
  return NavivoxFrameCodec.encode(
    NavivoxFrame(
      type: type,
      messageId: id,
      timestamp: DateTime.utc(2026, 5, 5),
      contentType: 'application/json',
      payload: Uint8List.fromList(utf8.encode(jsonEncode({'text': text}))),
    ),
  );
}
