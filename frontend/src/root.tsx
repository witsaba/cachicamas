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

OWN-WORLD: deep ink-navy panels on a void ground, 1px hairlines, hard corners,
no shadow and no glow. Five working colours, each with one job: amber is the
machine speaking, cyan is navigable, green running, violet suspended, red
failed. Spline Sans Mono is the machine's voice; Spline Sans is language.

STORY: this is a company of specialists with jobs and boundaries; most are not
built yet and the board says so; the one that is being built can be opened and
can be stopped mid-run.

FIRST VIEWPORT: status rail, command line, then the register — six archetype
cells with lamps, plans and gauges — with the runtime's real counts beside it
and the function-key dock along the bottom edge.

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
