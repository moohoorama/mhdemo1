//go:build !release

package core

// lenientRestore: a development build fails a restore on any mismatch, so it is seen at once.
const lenientRestore = false
