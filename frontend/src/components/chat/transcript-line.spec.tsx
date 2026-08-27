/**
 * TranscriptLine — one line of a conversation.
 *
 * The two assertions that carry real weight here are the sanitizer and the
 * suspension. Model output reaches the DOM through `dangerouslySetInnerHTML`,
 * so the allowlist in `lib/markdown.ts` is the only thing between a model and
 * a script tag; and a pending permission must show the exact call, because an
 * approval nobody can read is not a decision anyone made.
 */
import { $ } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { TranscriptLine } from "./transcript-line";
import type { TranscriptEntry } from "~/lib/mock/chat";
import { agentBySlug } from "~/lib/mock/staff";

/** Every line belongs to a colleague; the screen renders their real name. */
const ASSISTANT = agentBySlug("assistant")!;
const WHO = {
  agent: ASSISTANT,
  youName: "Ana Rivas",
  youInitials: "AR",
} as const;

const HOLD: TranscriptEntry = {
  kind: "hold",
  id: "h1",
  tool: "Billing",
  intent: "Send a payment reminder",
  args: [
    ["to", "accounts@northgate.example"],
    ["about", "Invoice 7712 · £7,900"],
  ],
  risk: "This email leaves the building and cannot be recalled.",
  decision: "pending",
};

describe("components/chat/transcript-line", () => {
  it("labels who is speaking, in a fixed gutter", async () => {
    const you = await createDOM();
    await you.render(
      <TranscriptLine {...WHO}
        entry={{
          kind: "said",
          id: "a",
          who: "you",
          text: "hi",
          state: "final",
        }}
      />,
    );
    const line = you.screen.querySelector('[data-testid="line-said-a"]');
    expect(line?.getAttribute("data-who")).toBe("you");
    expect(line?.textContent).toContain("Ana Rivas");

    const model = await createDOM();
    await model.render(
      <TranscriptLine {...WHO}
        entry={{
          kind: "said",
          id: "b",
          who: "chat",
          text: "hi",
          state: "final",
        }}
      />,
    );
    expect(
      model.screen.querySelector('[data-testid="line-said-b"]')?.textContent,
    ).toContain("Assistant");
  });

  it("renders model markdown, and strips what the allowlist forbids", async () => {
    const { screen, render } = await createDOM();
    await render(
      <TranscriptLine {...WHO}
        entry={{
          kind: "said",
          id: "m",
          who: "chat",
          text: '**bold** <script>alert(1)</script><img src=x onerror="alert(2)">',
          state: "final",
        }}
      />,
    );
    const line = screen.querySelector('[data-testid="line-said-m"]');
    const html = line?.innerHTML ?? "";
    expect(html).toContain("<strong>bold</strong>");
    expect(html).not.toContain("<script");
    expect(html).not.toContain("onerror");
  });

  it("never renders a person's own words as markup", async () => {
    // What a person typed is text. Rendering it as markdown would let a
    // paste turn into markup in their own transcript.
    const { screen, render } = await createDOM();
    await render(
      <TranscriptLine {...WHO}
        entry={{
          kind: "said",
          id: "u",
          who: "you",
          text: "<b>not bold</b>",
          state: "final",
        }}
      />,
    );
    const line = screen.querySelector('[data-testid="line-said-u"]');
    expect(line?.textContent).toContain("<b>not bold</b>");
    expect(line?.querySelector("b")).toBeFalsy();
  });

  it("shows a caret while a line is still arriving, and drops it when it is done", async () => {
    const streaming = await createDOM();
    await streaming.render(
      <TranscriptLine {...WHO}
        entry={{
          kind: "said",
          id: "s",
          who: "chat",
          text: "par",
          state: "streaming",
        }}
      />,
    );
    expect(
      streaming.screen.querySelector('[data-testid="stream-caret"]'),
    ).toBeTruthy();

    const done = await createDOM();
    await done.render(
      <TranscriptLine {...WHO}
        entry={{
          kind: "said",
          id: "d",
          who: "chat",
          text: "part",
          state: "final",
        }}
      />,
    );
    expect(
      done.screen.querySelector('[data-testid="stream-caret"]'),
    ).toBeFalsy();
  });

  it("prints a note as a rule with its label, hiding the rule itself", async () => {
    const { screen, render } = await createDOM();
    await render(
      <TranscriptLine {...WHO}
        entry={{ kind: "note", id: "n", label: "TURN OPENED", detail: "run 1" }}
      />,
    );
    const note = screen.querySelector('[data-testid="line-note-n"]');
    expect(note?.textContent).toContain("TURN OPENED");
    expect(note?.textContent).toContain("run 1");
    expect(note?.querySelector('[aria-hidden="true"]')).toBeTruthy();
  });

  it("shows a running tool as running, with a word not just a colour", async () => {
    const { screen, render } = await createDOM();
    await render(
      <TranscriptLine {...WHO}
        entry={{
          kind: "tool",
          id: "t",
          tool: "dba.query",
          intent: "Read one row",
          args: [["mode", "read-only"]],
          state: "running",
        }}
      />,
    );
    const line = screen.querySelector('[data-testid="line-tool-t"]');
    expect(line?.textContent).toContain("Running");
    expect(line?.textContent).toContain("dba.query");
    expect(line?.textContent).toContain("read-only");
  });

  it("says plainly that a refused call did not run", async () => {
    const { screen, render } = await createDOM();
    await render(
      <TranscriptLine {...WHO}
        entry={{
          kind: "tool",
          id: "t",
          tool: "dba.execute",
          intent: "Drop it",
          args: [],
          state: "denied",
        }}
      />,
    );
    expect(
      screen.querySelector('[data-testid="line-tool-t"]')?.textContent,
    ).toContain("Nothing ran");
  });

  it("shows a pending permission with the exact call and both answers", async () => {
    const { screen, render } = await createDOM();
    await render(<TranscriptLine {...WHO} entry={HOLD} onDecide$={$(() => {})} />);
    const hold = screen.querySelector('[data-testid="line-hold-h1"]');
    const text = hold?.textContent ?? "";
    expect(text).toContain("Waiting for you");
    expect(text).toContain("Send a payment reminder");
    // The exact thing it wants to do, before anyone can answer.
    expect(text).toContain("accounts@northgate.example");
    expect(text).toContain("cannot be recalled");
    expect(
      screen.querySelector('[data-testid="permission-allow"]'),
    ).toBeTruthy();
    expect(
      screen.querySelector('[data-testid="permission-refuse"]'),
    ).toBeTruthy();
  });

  it("withdraws the controls once the decision is made, and keeps the record", async () => {
    const { screen, render } = await createDOM();
    await render(<TranscriptLine {...WHO} entry={{ ...HOLD, decision: "denied" }} />);
    expect(
      screen.querySelector('[data-testid="permission-allow"]'),
    ).toBeFalsy();
    expect(
      screen.querySelector('[data-testid="permission-refuse"]'),
    ).toBeFalsy();
    const text =
      screen.querySelector('[data-testid="line-hold-h1"]')?.textContent ?? "";
    expect(text).toContain("You refused");
    // The thing it wanted to do stays on the record after the refusal.
    expect(text).toContain("accounts@northgate.example");
  });

  it("renders a failure as a typed envelope with a recovery, never as a spinner", async () => {
    const { screen, render } = await createDOM();
    await render(
      <TranscriptLine {...WHO}
        entry={{
          kind: "fault",
          id: "f",
          code: "No access",
          message: "I have not been given access to the calendar account.",
          recovery: "Someone with admin rights can grant it from Organisation.",
        }}
      />,
    );
    const text =
      screen.querySelector('[data-testid="line-fault-f"]')?.textContent ?? "";
    expect(text).toContain("No access");
    expect(text).toContain("I have not been given access");
    expect(text).toContain("Someone with admin rights");
    // Nothing retries quietly; the screen says so where a person reads it.
    expect(text).toContain("No automatic retry");
  });
});
