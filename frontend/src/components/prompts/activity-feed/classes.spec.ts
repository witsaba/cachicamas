import { describe, expect, it } from "vitest";
import { eventClasses } from "./classes";

describe("eventClasses", () => {
  it("created — has emerald color", () => {
    const classes = eventClasses("created");
    expect(classes.container).toContain("text-emerald-700");
    expect(classes.iconColor).toContain("text-emerald-500");
    expect(classes.text).toContain("text-slate-700");
  });

  it("edited — has blue color", () => {
    const classes = eventClasses("edited");
    expect(classes.container).toContain("text-blue-700");
    expect(classes.iconColor).toContain("text-blue-500");
  });

  it("restored — has amber color", () => {
    const classes = eventClasses("restored");
    expect(classes.container).toContain("text-amber-700");
    expect(classes.iconColor).toContain("text-amber-500");
  });

  it("deleted — has red color", () => {
    const classes = eventClasses("deleted");
    expect(classes.container).toContain("text-red-700");
    expect(classes.iconColor).toContain("text-red-500");
  });

  it("all types include flex and gap styling", () => {
    for (const type of ["created", "edited", "restored", "deleted"] as const) {
      const classes = eventClasses(type);
      expect(classes.container).toContain("flex");
      expect(classes.container).toContain("items-start");
      expect(classes.container).toContain("gap-2");
    }
  });

  it("all types include text-sm", () => {
    for (const type of ["created", "edited", "restored", "deleted"] as const) {
      const classes = eventClasses(type);
      expect(classes.container).toContain("text-sm");
    }
  });
});
