# whisper.wasm fixture

`whisper.wasm` is vendored from `github.com/agnivade/whisper-wasi` commit
`48c8dc14efd1f74c4b3b8fcc1c045a977c2bf7d7`.

The source repository is a small MIT-licensed whisper.cpp WASI prototype. Its
`binary.wasm` file was copied byte-for-byte to this directory as
`whisper.wasm`.

Fixture metadata:

- size: `3387207` bytes
- sha256: `e575a73bff506574513709c26ced98b65a90b6960810078fc4d928882bc1bd2e`
- upstream file: `binary.wasm`

`jfk.wav` is the small speech fixture from `ggerganov/whisper.cpp` samples,
used only for end-to-end transcription tests. It is not a model artifact.

- size: `352078` bytes
- sha256: `59dfb9a4acb36fe2a2affc14bacbee2920ff435cb13cc314a08c13f66ba7860e`
- upstream file: `samples/jfk.wav`

Rebuild notes:

1. Clone `https://github.com/agnivade/whisper-wasi`.
2. Check out commit `48c8dc14efd1f74c4b3b8fcc1c045a977c2bf7d7`.
3. Rebuild that repository's `binary.wasm` using its upstream build process.
4. Copy `binary.wasm` here as `whisper.wasm`.
5. Update `ArtifactSHA256`, `ArtifactSizeBytes`, and this README if the bytes
   change.
