# Navivox Voice Morph Surface Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a reusable, accessible Flutter voice-mode morphing graphic for Navivox.

**Architecture:** Add a focused `features/voice/widgets/voice_morph_surface.dart` file containing the state enum, style resolver, widget, and painter. Tests verify public widget/painter behavior without golden image coupling.

**Tech Stack:** Flutter, `CustomPainter`, `AnimationController`, `RepaintBoundary`, Flutter widget tests.

---

### Task 1: Voice Morph Widget

**Files:**
- Create: `flutter-navivox/app/lib/features/voice/widgets/voice_morph_surface.dart`
- Test: `flutter-navivox/app/test/features/voice/widgets/voice_morph_surface_test.dart`

- [ ] **Step 1: Write the failing tests**

Test semantics labels for all states, style differences between states, intensity clamping, and reduced-motion phase freezing.

- [ ] **Step 2: Run the focused test**

Run: `cd flutter-navivox/app && flutter test test/features/voice/widgets/voice_morph_surface_test.dart`

Expected: fail because `VoiceMorphSurface` does not exist.

- [ ] **Step 3: Implement the widget and painter**

Create `VoiceMorphState`, `VoiceMorphStyle`, `VoiceMorphSurface`, and `VoiceMorphPainter`. The widget clamps intensity, freezes phase when `reducedMotion` is true, wraps `CustomPaint` in `RepaintBoundary`, and sets a live semantic label.

- [ ] **Step 4: Verify focused and app tests**

Run:

```sh
cd flutter-navivox/app
flutter test test/features/voice/widgets/voice_morph_surface_test.dart
flutter test
flutter analyze
```

Expected: all pass with no analyzer issues.
