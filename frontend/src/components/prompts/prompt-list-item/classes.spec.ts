import { describe, expect, it } from "vitest";
import { listItemClasses } from "./classes";

describe("listItemClasses", () => {
  it("unselected — applies hover background", () => {
    const classes = listItemClasses(false);
    expect(classes.container).toContain("hover:bg-slate-50");
    expect(classes.container).not.toContain("bg-slate-100");
    expect(classes.container).toContain("cursor-pointer");
    expect(classes.container).toContain("w-full");
    expect(classes.slug).toContain("text-sm");
    expect(classes.meta).toContain("text-xs");
  });

  it("selected — applies selected background", () => {
    const classes = listItemClasses(true);
    expect(classes.container).toContain("bg-slate-100");
    expect(classes.container).not.toContain("hover:bg-slate-50");
    expect(classes.slug).toContain("text-sm");
    expect(classes.meta).toContain("text-xs");
  });

  it("slug always includes truncation", () => {
    const unselected = listItemClasses(false);
    const selected = listItemClasses(true);
    expect(unselected.slug).toContain("truncate");
    expect(selected.slug).toContain("truncate");
  });

  it("meta always includes truncation and muted color", () => {
    const unselected = listItemClasses(false);
    const selected = listItemClasses(true);
    expect(unselected.meta).toContain("truncate");
    expect(selected.meta).toContain("truncate");
    expect(unselected.meta).toContain("text-slate-500");
    expect(selected.meta).toContain("text-slate-500");
  });
});
