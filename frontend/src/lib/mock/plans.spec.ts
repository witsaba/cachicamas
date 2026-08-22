/**
 * The plans' contract.
 *
 * Pricing is the one place on a marketing page where a mistake is not a design
 * problem, so these are mostly integrity checks: the ladder only goes up, the
 * comparison table cannot silently lose a column, and exactly one level is
 * marked — a page with two recommendations has recommended nothing.
 */
import { describe, it, expect } from "vitest";
import { ANNUAL_SAVING, COMPARISON, PLANS, priceFor } from "./plans";

describe("the plans", () => {
  it("has one free entry point and no duplicate levels", () => {
    const slugs = PLANS.map((p) => p.slug);
    expect(new Set(slugs).size).toBe(slugs.length);
    expect(PLANS.filter((p) => p.price.monthly === 0)).toHaveLength(1);
    expect(PLANS[0].price.monthly).toBe(0);
  });

  it("only ever goes up as you read across", () => {
    for (let i = 1; i < PLANS.length; i += 1) {
      expect(
        PLANS[i].price.monthly,
        `${PLANS[i].slug} vs ${PLANS[i - 1].slug}`,
      ).toBeGreaterThan(PLANS[i - 1].price.monthly);
    }
  });

  it("makes annual cheaper than monthly on every paid level", () => {
    for (const plan of PLANS) {
      if (plan.price.monthly === 0) {
        expect(plan.price.annual, plan.slug).toBe(0);
        continue;
      }
      expect(plan.price.annual, plan.slug).toBeLessThan(plan.price.monthly);
      expect(priceFor(plan, "annual")).toBe(plan.price.annual);
      expect(priceFor(plan, "monthly")).toBe(plan.price.monthly);
    }
    expect(ANNUAL_SAVING).toBeTruthy();
  });

  it("recommends exactly one level", () => {
    expect(PLANS.filter((p) => p.recommended)).toHaveLength(1);
  });

  it("says who each level is for, and what it staffs", () => {
    for (const plan of PLANS) {
      expect(plan.forWhom.length, plan.slug).toBeGreaterThan(15);
      expect(plan.staffing.length, plan.slug).toBeGreaterThan(5);
      expect(plan.includes.length, plan.slug).toBeGreaterThan(2);
      expect(plan.cta.length, plan.slug).toBeGreaterThan(3);
    }
  });

  it("puts the three Workforce-only capabilities on the top level and nowhere else", () => {
    const top = PLANS[PLANS.length - 1];
    expect(top.slug).toBe("workforce");
    const text = top.includes.join(" ");
    expect(text).toMatch(/ten agents/i);
    expect(text).toMatch(/open desks/i);
    expect(text).toMatch(/paired duo/i);
    for (const plan of PLANS.slice(0, -1)) {
      expect(plan.includes.join(" "), plan.slug).not.toMatch(/open desks/i);
      expect(plan.includes.join(" "), plan.slug).not.toMatch(/paired duo/i);
    }
  });
});

describe("the comparison table", () => {
  it("has a value for every level in every row", () => {
    // A missing key renders an empty cell, which reads as "not included" and
    // is the cheapest way to accidentally misprice a feature.
    for (const row of COMPARISON) {
      for (const plan of PLANS) {
        expect(
          Object.prototype.hasOwnProperty.call(row.values, plan.slug),
          `${row.label} / ${plan.slug}`,
        ).toBe(true);
      }
    }
  });

  it("never gives a lower level something a higher one lacks", () => {
    const rank = new Map(PLANS.map((p, i) => [p.slug, i]));
    for (const row of COMPARISON) {
      const included = PLANS.filter((p) => row.values[p.slug] === true).map(
        (p) => rank.get(p.slug)!,
      );
      if (!included.length) continue;
      const lowest = Math.min(...included);
      for (const plan of PLANS) {
        if (rank.get(plan.slug)! > lowest) {
          expect(row.values[plan.slug], `${row.label} / ${plan.slug}`).not.toBe(
            false,
          );
        }
      }
    }
  });
});
