/**
 * SkillListItem classes — pure function table tests.
 *
 * Anti-drift note:
 *   These tests assert behavior (selected vs unselected state) and
 *   essential token presence (truncation, hover affordance). They do
 *   NOT lock exact Tailwind class strings (anti-fragile — small
 *   utility-class reorders should not break tests).
 */
import { describe, expect, it } from "vitest";
import { listItemClasses } from "./classes";

describe("listItemClasses", () => {
  it("unselected — applies hover background and base styling", () => {
    const classes = listItemClasses(false);
    expect(classes.container).toContain("hover:bg-slate-50");
    expect(classes.container).not.toContain("bg-slate-100");
    expect(classes.container).toContain("cursor-pointer");
    expect(classes.container).toContain("w-full");
    expect(classes.container).toContain("rounded");
    expect(classes.container).toContain("text-left");
  });

  it("selected — applies selected background, NO hover variant", () => {
    const classes = listItemClasses(true);
    expect(classes.container).toContain("bg-slate-100");
    expect(classes.container).not.toContain("hover:bg-slate-50");
    expect(classes.container).toContain("cursor-pointer");
    expect(classes.container).toContain("w-full");
  });

  it("name span — medium weight, slate-900 text, truncation", () => {
    const unselected = listItemClasses(false);
    const selected = listItemClasses(true);
    expect(unselected.name).toContain("text-sm");
    expect(unselected.name).toContain("font-medium");
    expect(unselected.name).toContain("text-slate-900");
    expect(unselected.name).toContain("truncate");
    expect(selected.name).toBe(unselected.name);
  });

  it("meta span — muted color, smaller text, truncation", () => {
    const unselected = listItemClasses(false);
    const selected = listItemClasses(true);
    expect(unselected.meta).toContain("text-xs");
    expect(unselected.meta).toContain("text-slate-500");
    expect(unselected.meta).toContain("truncate");
    expect(selected.meta).toBe(unselected.meta);
  });
});
