-- 0004_permission_decisions.sql — CH-10.3 sibling table for permission
-- decisions (R-CPM-006, R-CCS-018, NFR-CCS-006 forward-only).
--
-- CH-10 widens Exchange with PermissionDecisions []PermissionDecisionRecord
-- (one record per ask/made pair the gate produced during the turn).
-- Rather than ALTER chat_exchanges (which would violate NFR-CCS-006
-- forward-only), the new field lives in a sibling table keyed by
-- (chat_exchanges.participant_id, chat_exchanges.position) — the
-- same composite key the existing exchanges table uses.
--
-- Forward-only (NFR-CCS-006): no DROP, no ALTER, no TRUNCATE, no
-- destructive ops. The runner refuses any migration whose every
-- line does not match the CREATE TABLE / CREATE INDEX / INSERT /
-- ALTER TABLE ADD COLUMN allowlist. This migration passes the
-- check trivially.
--
-- outcome is CHECK-constrained to the chat wire's CLOSED 2-value
-- vocabulary (D-12 collapse: Layer 2's 4-value PermissionOutcome
-- reduces to 2 at the chat wire — AllowOnce + AllowAlways →
-- "allow_once"; Deny + ModifyInput → "deny"). The chat projector
-- translates BEFORE persistence; the chat store only sees the
-- collapsed form.

CREATE TABLE chat_permission_decisions (
    participant_id text NOT NULL,
    exchange_position integer NOT NULL,
    position integer NOT NULL,
    wire_call_id text NOT NULL,
    tool text NOT NULL,
    outcome text NOT NULL CHECK (outcome IN ('allow_once','deny')),
    PRIMARY KEY (participant_id, exchange_position, position),
    FOREIGN KEY (participant_id, exchange_position)
        REFERENCES chat_exchanges (participant_id, position) ON DELETE CASCADE
);
CREATE INDEX chat_permission_decisions_lookup_idx
    ON chat_permission_decisions (participant_id, exchange_position);