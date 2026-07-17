/**
 * SkillEditor classes — pure function table tests.
 *
 * Verifies the contract that the editor's form-control class
 * strings live in a single source of truth (this file). Pinning
 * key tokens (not exact strings) keeps tests anti-fragile.
 */
import { describe, expect, it } from "vitest";
import { skillEditorClasses } from "./classes";

describe("skillEditorClasses", () => {
  it("returns a static table (no input args)", () => {
    const a = skillEditorClasses();
    const b = skillEditorClasses();
    // Pure function: same input → same output.
    expect(a).toEqual(b);
  });

  it("description input uses slate border + small text + focus ring", () => {
    const c = skillEditorClasses();
    expect(c.descriptionInput).toContain("border-slate-200");
    expect(c.descriptionInput).toContain("text-sm");
    expect(c.descriptionInput).toContain("rounded");
    expect(c.descriptionInput).toContain("focus:ring-1");
    expect(c.descriptionInput).toContain("focus:outline-none");
  });

  it("body textarea is monospace + non-resizable + slate themed", () => {
    const c = skillEditorClasses();
    expect(c.bodyTextarea).toContain("font-mono");
    expect(c.bodyTextarea).toContain("resize-none");
    expect(c.bodyTextarea).toContain("border-slate-200");
    expect(c.bodyTextarea).toContain("text-sm");
    // Body textarea is taller / has more padding than description input.
    expect(c.bodyTextarea).toContain("px-3");
  });

  it("footer row uses flex + border-t + slate-200 divider", () => {
    const c = skillEditorClasses();
    expect(c.footerRow).toContain("flex");
    expect(c.footerRow).toContain("items-center");
    expect(c.footerRow).toContain("justify-between");
    expect(c.footerRow).toContain("border-t");
    expect(c.footerRow).toContain("border-slate-200");
  });

  it("noChangesHint uses muted color + extra-small text", () => {
    const c = skillEditorClasses();
    expect(c.noChangesHint).toContain("text-xs");
    expect(c.noChangesHint).toContain("text-slate-400");
  });
});
