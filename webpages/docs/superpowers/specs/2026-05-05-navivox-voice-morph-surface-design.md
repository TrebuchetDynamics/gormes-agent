# Navivox Voice Morph Surface Design

## Goal

Add a soft morphing voice-mode graphic to the Flutter Navivox app that feels alive
without requiring live audio plumbing in this slice.

## Contract

`VoiceMorphSurface` exposes `state`, `intensity`, and `reducedMotion`.
`state` controls semantic label, palette, tempo, and posture. `intensity` is a
clamped `0.0..1.0` signal that can be fake today and later fed by mic or TTS
amplitude. `reducedMotion` keeps state-specific visuals but freezes phase-driven
movement.

## States

- `idle`: calm breathing.
- `listening`: wider, receptive pulse.
- `thinking`: slower cooler swirl.
- `speaking`: warm intensity-driven pulse.
- `disabled`: muted static surface.

## Implementation

Create a Flutter `CustomPainter`-backed widget under
`flutter-navivox/app/lib/features/voice/widgets/`. The widget uses
`AnimationController` only for phase, wraps the paint area in `RepaintBoundary`,
and exposes a live semantic label for assistive technology. It does not add
visible instructional text.

## Tests

Widget tests cover state semantics, state-to-style differences, intensity
clamping, and reduced-motion phase freezing. Tests inspect the painter contract
instead of relying on fragile golden screenshots.
