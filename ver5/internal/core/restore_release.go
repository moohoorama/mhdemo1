//go:build release

package core

// lenientRestore: a release build plays on through a rule mismatch in a save, and RestoreWarned records it.
const lenientRestore = true
