// Package picker owns deterministic setup/tool picker rows built from the
// upstream parity manifest plus inert plugin metadata.
//
// It exposes picker option reports. It may report shared platformconfig issue
// types, but must not mutate platform selections or perform config persistence.
package picker
