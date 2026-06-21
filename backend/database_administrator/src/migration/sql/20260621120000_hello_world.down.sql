-- Hello-world down migration: symmetric no-op so `goose down` works
-- locally for tests without changing schema. Per design §7 (Q5),
-- schema-only changes should always write a reversible down; data
-- moves should leave a TODO comment instead. This is neither — it is
-- a smoke test — but the symmetric body costs nothing.
SELECT 1;
