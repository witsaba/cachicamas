// Package archetype — version.go implements the version-aware rebuild
// contract (REQ-CCVP-001/002/003, design AD-3). Generic to any
// archetype kind; the Conversation (or equivalent runtime state in
// future archetypes) consults a versionTracker at every Send boundary
// to decide whether to rebuild the system prompt from the persisted
// row.
//
// Kept in its own file so the tests are hermetic: no Provider / Store /
// ToolSource / PermissionPolicy needed.
package archetype

import "context"

// VersionTracker owns the version-checking state for one in-flight
// runtime slot (one Conversation, one CodingSession, ...). Constructed
// once per slot and consulted on every Send at the Send boundary
// (NEVER mid-turn — REQ-CCVP-003).
type VersionTracker struct {
	loader        Loader
	kind          ArchetypeKind
	participantID string

	// recordedVersion is the version the runtime slot loaded at
	// first-construct (or the last successful Reload). On Reload,
	// the Loader's current version is compared against this value.
	recordedVersion int

	// applyPrompt is invoked when the Loader reports a newer version
	// than recordedVersion. The implementation is responsible for
	// updating the runtime's system prompt slot (e.g. Conversation's
	// harness.System). Decoupled here so the tracker is testable
	// without a Harness.
	applyPrompt func(newPrompt string)
}

// NewVersionTracker loads the current config via loader and seeds the
// tracker. If loader is nil, the tracker is inert (Reload is a no-op).
// If the initial LoadByKindAndOrg returns an error, the tracker is
// constructed with recordedVersion=0 — Reload will retry on the next
// call. This keeps runtime construction from failing because of a
// transient Loader hiccup.
//
// On a successful initial load, applyPrompt IS called with the loaded
// prompt so the runtime slot (e.g. Conversation.harness.System) is
// in sync with the persisted config from construction. The "no-op on
// first load" was the v1 contract; v2 fires applyPrompt on first
// load too so the runtime can be initialised in a single step.
func NewVersionTracker(ctx context.Context, loader Loader, kind ArchetypeKind, participantID string, applyPrompt func(string)) (*VersionTracker, error) {
	vt := &VersionTracker{
		loader:        loader,
		kind:          kind,
		participantID: participantID,
		applyPrompt:   applyPrompt,
	}
	if loader == nil {
		return vt, nil
	}
	cfg, _, err := loader.LoadByKindAndOrg(ctx, kind, participantID)
	if err != nil {
		// Surface the error so the composition root can log it but
		// still construct the runtime slot — a missing config row
		// MUST NOT block first-use. Caller logs and proceeds.
		return vt, err
	}
	vt.recordedVersion = cfg.Version
	if applyPrompt != nil {
		applyPrompt(cfg.SystemPrompt)
	}
	return vt, nil
}

// Reload consults the Loader and, on version mismatch, applies the new
// prompt. On match, it is a no-op (REQ-CCVP-002 Scenario 2). On error
// from the Loader, the error is returned so the caller can decide how
// to surface it (the runtime logs and proceeds with the existing
// prompt — a transient Loader outage must not tear down a turn).
func (vt *VersionTracker) Reload(ctx context.Context) error {
	if vt == nil || vt.loader == nil {
		return nil
	}
	cfg, _, err := vt.loader.LoadByKindAndOrg(ctx, vt.kind, vt.participantID)
	if err != nil {
		return err
	}
	if cfg.Version == vt.recordedVersion {
		return nil
	}
	vt.recordedVersion = cfg.Version
	if vt.applyPrompt != nil {
		vt.applyPrompt(cfg.SystemPrompt)
	}
	return nil
}

// RecordedVersion returns the version the tracker last saw from the
// Loader. Exposed for tests and for the runtime's internal
// observability path.
func (vt *VersionTracker) RecordedVersion() int {
	if vt == nil {
		return 0
	}
	return vt.recordedVersion
}
