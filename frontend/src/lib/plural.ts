/**
 * "1 agents · 1 people" is the kind of small wrongness that makes a company's
 * own workspace feel like a demo of a workspace. One helper, used wherever a
 * count meets a noun.
 */
export function count(n: number, one: string, many = `${one}s`): string {
  return `${n} ${n === 1 ? one : many}`;
}

/** The same, for nouns whose plural is not the singular plus an s. */
export const people = (n: number): string => count(n, "person", "people");
