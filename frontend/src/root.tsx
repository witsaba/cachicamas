import { component$, isDev } from "@builder.io/qwik";
import { QwikCityProvider, RouterOutlet } from "@builder.io/qwik-city";
import { RouterHead } from "./components/router-head/router-head";

import "./global.css";

/**
 * The direction contract for this build, emitted into the document so it can be
 * audited in the shipped output rather than only in source. Grep the built
 * markup for `impeccable-direction` to find it.
 */
const DIRECTION_CONTRACT = `<!--
impeccable-direction seed 8a09c7ce (operate, form 7)

THESIS: a company's specialist agents are services on a dealing-room board —
each with a code, a state and a number, all readable at once. Refuses the
launcher grid of identical icons, which would imply six working specialists
when five do not exist.

OWN-WORLD: deep ink-navy panels on a void ground, 1px hairlines at or above
3:1 on every ground they sit on, hard corners, no shadow and no glow. STRUCTURE
IS NEUTRAL — panel labels, key legends, gauges and the current dock cell carry
no colour. Colour is reserved for what is happening: amber is the system
speaking about itself now (wordmark, prompt, caret, focus, primary action, the
demonstration marker, the one archetype in build), cyan is navigable, green
running, violet suspended, red failed. State works on two axes, because five of
six specialists are inactive: colour separates what is happening, and a filled
versus hollow mark separates what is not. Spline Sans Mono is the machine's
voice; Spline Sans is language.

STORY: this is a company of specialists with jobs and boundaries; most are not
built yet and the board says so; the one that is being built can be opened and
can be stopped mid-run.

FIRST VIEWPORT: signed out, the rail alone over the headline and the stack's
real counts, with the register whole below it. Signed in, the rail and the
command line over the register — six archetype cells with lamps, plans and
gauges — the runtime beside it, and the function-key dock along the bottom
edge. The board arrives painted; nothing stages.

FORM: the exchange terminal; candidate 7 of the grounded list; seed 8a09c7ce.

FINISH: unreviewed and undocumented is unfinished; this build ends with the
finish review, the verdict, and DESIGN.md.
-->`;

export default component$(() => {
  /**
   * The root of a QwikCity site always starts with the <QwikCityProvider>
   * component, immediately followed by the document's <head> and <body>.
   */
  return (
    <QwikCityProvider>
      <head>
        <meta charset="utf-8" />
        {!isDev && (
          <link
            rel="manifest"
            href={`${import.meta.env.BASE_URL}manifest.json`}
          />
        )}
        {/* The two voices. Spline Sans Mono carries the machine — codes,
            labels, states, tabular figures — and Spline Sans carries language.
            Both are preconnected so the terminal does not repaint its own
            chrome after first byte. */}
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link
          rel="preconnect"
          href="https://fonts.gstatic.com"
          crossOrigin=""
        />
        <link
          rel="stylesheet"
          href="https://fonts.googleapis.com/css2?family=Spline+Sans+Mono:wght@400;500;600&family=Spline+Sans:wght@400;500;600&display=swap"
        />
        <RouterHead />
      </head>
      <body lang="en">
        <div hidden dangerouslySetInnerHTML={DIRECTION_CONTRACT} />
        <RouterOutlet />
      </body>
    </QwikCityProvider>
  );
});
