/**
 * "1 agents · 1 people" is the kind of small wrongness that makes a company's
 * own workspace read as a demo of a workspace rather than as the thing.
 */
import { describe, it, expect } from "vitest";
import { count, people } from "./plural";

describe("count", () => {
  it("keeps the singular at one", () => {
    expect(count(1, "agent")).toBe("1 agent");
    expect(count(1, "team")).toBe("1 team");
  });

  it("pluralises everything else, including zero", () => {
    expect(count(0, "agent")).toBe("0 agents");
    expect(count(2, "agent")).toBe("2 agents");
  });

  it("takes an irregular plural when the noun needs one", () => {
    expect(people(1)).toBe("1 person");
    expect(people(0)).toBe("0 people");
    expect(people(3)).toBe("3 people");
  });
});
