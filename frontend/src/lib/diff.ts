/**
 * Line-mode diff computation using LCS (Longest Common Subsequence).
 *
 * This is a simpler, more predictable approach than diff-match-patch's
 * character-level diffing with semantic cleanup, which can merge changes
 * across lines in unexpected ways.
 *
 * Algorithm (Myers-style LCS diff):
 *   1. Split both texts into lines.
 *   2. Compute LCS array of the two line arrays.
 *   3. Walk from the LCS to emit delete/insert/equal lines.
 *
 * Output: a flat array of DiffLine objects, ordered as they appear in
 * the diff. Consecutive same-type lines are NOT merged.
 */

export type DiffType = "delete" | "insert" | "equal";

export interface DiffLine {
  type: DiffType;
  text: string;
}

export interface DiffResult {
  lines: DiffLine[];
}

/**
 * Compute LCS (Longest Common Subsequence) of two arrays.
 * Returns a 2D array where lcs[i][j] = length of LCS of arr1[0..i-1] and arr2[0..j-1].
 */
function computeLCS<T>(arr1: T[], arr2: T[]): number[][] {
  const m = arr1.length;
  const n = arr2.length;
  const lcs: number[][] = Array.from({ length: m + 1 }, () =>
    new Array(n + 1).fill(0),
  );

  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      if (arr1[i - 1] === arr2[j - 1]) {
        lcs[i][j] = lcs[i - 1][j - 1] + 1;
      } else {
        lcs[i][j] = Math.max(lcs[i - 1][j], lcs[i][j - 1]);
      }
    }
  }

  return lcs;
}

/**
 * Trace back through the LCS matrix to produce diff lines.
 * This walks from (m, n) back to (0, 0), emitting diff operations.
 */
function traceBack(
  oldLines: string[],
  newLines: string[],
  lcs: number[][],
): DiffLine[] {
  let i = oldLines.length;
  let j = newLines.length;

  const ops: DiffLine[] = [];

  while (i > 0 || j > 0) {
    if (i > 0 && j > 0 && oldLines[i - 1] === newLines[j - 1]) {
      // Equal line
      ops.unshift({ type: "equal", text: oldLines[i - 1] });
      i--;
      j--;
    } else if (j > 0 && (i === 0 || lcs[i][j - 1] >= lcs[i - 1][j])) {
      // Insert
      ops.unshift({ type: "insert", text: newLines[j - 1] });
      j--;
    } else {
      // Delete
      ops.unshift({ type: "delete", text: oldLines[i - 1] });
      i--;
    }
  }

  return ops;
}

/**
 * Compute a line-level diff between two text strings.
 *
 * Uses LCS-based diffing on line-split arrays, which is more predictable
 * than character-level diffing for multi-line text.
 *
 * @param oldText - The original text
 * @param newText - The modified text
 * @returns DiffResult with an ordered array of DiffLine objects
 */
/**
 * Strip trailing empty strings produced by split("\n").
 * "hello\n".split("\n") = ["hello", ""]
 * "".split("\n") = [""]
 * This function removes those trailing empties so we don't create phantom
 * equal lines from empty-string content.
 */
function splitLines(text: string): string[] {
  const lines = text.split("\n");
  while (lines.length > 0 && lines[lines.length - 1] === "") {
    lines.pop();
  }
  return lines;
}

export function computeLineDiff(oldText: string, newText: string): DiffResult {
  const oldLines = splitLines(oldText);
  const newLines = splitLines(newText);

  const lcs = computeLCS(oldLines, newLines);
  const lines = traceBack(oldLines, newLines, lcs);

  return { lines };
}
