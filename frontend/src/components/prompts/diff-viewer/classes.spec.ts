import { describe, expect, it } from "vitest";
import { diffLineClasses } from "./classes";

describe("diffLineClasses", () => {
  it("delete — has red background", () => {
    const classes = diffLineClasses("delete");
    expect(classes.container).toContain("bg-red-50");
    expect(classes.container).toContain("text-red-900");
  });

  it("insert — has green background", () => {
    const classes = diffLineClasses("insert");
    expect(classes.container).toContain("bg-emerald-50");
    expect(classes.container).toContain("text-emerald-900");
  });

  it("equal — has gray background", () => {
    const classes = diffLineClasses("equal");
    expect(classes.container).toContain("bg-slate-50");
    expect(classes.container).toContain("text-slate-700");
  });

  it("all types include font-mono and text-sm", () => {
    for (const type of ["delete", "insert", "equal"] as const) {
      const classes = diffLineClasses(type);
      expect(classes.container).toContain("font-mono");
      expect(classes.container).toContain("text-sm");
    }
  });

  it("all types include px and py", () => {
    for (const type of ["delete", "insert", "equal"] as const) {
      const classes = diffLineClasses(type);
      expect(classes.container).toContain("px-3");
      expect(classes.container).toContain("py-1");
    }
  });
});
