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
impeccable-direction seed b489eace (persuade; standing exit taken)

THESIS: your company's specialists are staff, not features — you meet them,
read what they do, see how long they have been here, and start a conversation.
The direction roll dealt a wayfinding programme; the user took the standing
exit instead, so this build is the category standard played straight, with the
craft bar set at Intercom and Attio. It refuses the machine-facing reading of
the same product: no codes, no terminal, no runtime figures, no architecture on
any surface a customer sees.

OWN-WORLD: white cards on a cool-neutral page, 1px lines, 6-12px radii, one
brand blue for actions and selection, and six department hues that identify a
department and never rank it. Onest sets every word. Elevation is reserved for
things that float. The one form rule that carries meaning: a person is a
circle, an agent is a rounded square — and it always ships a word beside it.

STORY: a visitor understands they can hire specialists their company is
missing, sees exactly which five and what each one does, finds a plan, and
signs in. Inside, an employee lands in their own company, picks a colleague
from the left rail, and talks to them.

FIRST VIEWPORT: signed out, a statement headline over one live conversation
playing as proof, with the primary action beside it and the staff strip below.
Signed in, a persistent left rail — company, Chat, Agents, Teams,
Organization — against a white content column that opens on the front desk.

FORM: the standing exit, the canon at full fidelity; seed b489eace.

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
        {/* One family for the whole product. Onest is a neutral grotesque with
            just enough character to stay ours at 11px and at 38px, and it is
            preconnected so no surface repaints its own type after first byte. */}
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link
          rel="preconnect"
          href="https://fonts.gstatic.com"
          crossOrigin=""
        />
        <link
          rel="stylesheet"
          href="https://fonts.googleapis.com/css2?family=Onest:wght@400;500;600;700&display=swap"
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
