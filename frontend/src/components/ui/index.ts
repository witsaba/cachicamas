/**
 * Design-system barrel.
 *
 * Reference: `frontend/src/components/ui/README.md` (line 188 quick-reference
 * example uses `import { Button, MenuItem } from "~/components/ui"`).
 *
 * ADDITIVE barrel — does NOT migrate the 21+ deep-path consumers
 * (Button, MenuItem deep imports stay as-is; out-of-band refactor).
 * The barrel only ENABLES the public-API import for future code
 * (and for the README's existing examples, which currently lie
 * because the file is missing).
 *
 * Re-exports alphabetized by component name.
 */
export { Button } from "./button/button";
export { MenuItem } from "./menu-item/menu-item";
export { SettingCard } from "./setting-card/setting-card";
