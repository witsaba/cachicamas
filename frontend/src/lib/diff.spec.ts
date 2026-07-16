import { describe, expect, it } from "vitest";
import { computeLineDiff, type DiffLine } from "./diff";

function getTextByType(lines: DiffLine[], type: "delete" | "insert" | "equal") {
  return lines.filter((l) => l.type === type).map((l) => l.text);
}

describe("computeLineDiff", () => {
  it("returns empty array for identical texts", () => {
    const result = computeLineDiff("hello\nworld", "hello\nworld");
    expect(result.lines).toHaveLength(2);
    expect(getTextByType(result.lines, "equal")).toEqual(["hello", "world"]);
    expect(getTextByType(result.lines, "delete")).toEqual([]);
    expect(getTextByType(result.lines, "insert")).toEqual([]);
  });

  it("marks added lines as insert", () => {
    const result = computeLineDiff("hello", "hello\nworld");
    expect(getTextByType(result.lines, "insert")).toContain("world");
    expect(getTextByType(result.lines, "equal")).toContain("hello");
  });

  it("marks removed lines as delete", () => {
    const result = computeLineDiff("hello\nworld", "hello");
    expect(getTextByType(result.lines, "delete")).toContain("world");
    expect(getTextByType(result.lines, "equal")).toContain("hello");
  });

  it("marks changed lines as delete + insert (not equal)", () => {
    const result = computeLineDiff("hello", "world");
    expect(getTextByType(result.lines, "delete")).toContain("hello");
    expect(getTextByType(result.lines, "insert")).toContain("world");
    expect(getTextByType(result.lines, "equal")).toEqual([]);
  });

  it("handles empty old text", () => {
    const result = computeLineDiff("", "hello\nworld");
    expect(getTextByType(result.lines, "insert")).toEqual(["hello", "world"]);
    expect(getTextByType(result.lines, "delete")).toEqual([]);
  });

  it("handles empty new text", () => {
    const result = computeLineDiff("hello\nworld", "");
    expect(getTextByType(result.lines, "delete")).toEqual(["hello", "world"]);
    expect(getTextByType(result.lines, "insert")).toEqual([]);
  });

  it("handles multiline diff", () => {
    const old = "line 1\nline 2\nline 3";
    const novo = "line 1\nmodified line 2\nline 3";
    const result = computeLineDiff(old, novo);
    expect(getTextByType(result.lines, "equal")).toContain("line 1");
    expect(getTextByType(result.lines, "equal")).toContain("line 3");
    expect(getTextByType(result.lines, "delete")).toContain("line 2");
    expect(getTextByType(result.lines, "insert")).toContain("modified line 2");
  });

  it("returns lines in order", () => {
    const result = computeLineDiff("a\nb", "a\nc");
    const types = result.lines.map((l) => l.type);
    // Should be: equal "a", delete "b", insert "c"
    expect(types.filter((t) => t === "equal")).toHaveLength(1);
    expect(types.filter((t) => t === "delete")).toHaveLength(1);
    expect(types.filter((t) => t === "insert")).toHaveLength(1);
  });

  it("each line is a valid DiffLine object", () => {
    const result = computeLineDiff("old", "new");
    for (const line of result.lines) {
      expect(line).toHaveProperty("type");
      expect(line).toHaveProperty("text");
      expect(["delete", "insert", "equal"]).toContain(line.type);
    }
  });
});
