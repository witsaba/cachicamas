/**
 * Agents — everyone who works here, and everyone who could.
 *
 * Split into two groups rather than sorted into one list, because "on staff"
 * and "could be on staff" are different questions and a mixed list makes a
 * person read every status word to answer either of them.
 */
import { component$ } from "@builder.io/qwik";
import { AgentCard } from "~/components/workspace/agent-card/agent-card";
import {
  PAGE_WELL,
  PageHeader,
} from "~/components/workspace/page-header/page-header";
import { AGENTS } from "~/lib/mock/staff";

export const AgentDirectory = component$(() => {
  const onStaff = AGENTS.filter((a) => a.status !== "available");
  const available = AGENTS.filter((a) => a.status === "available");

  return (
    <div class={PAGE_WELL}>
      <PageHeader
        title="Agents"
        lede="Specialist colleagues. Each one has a job, a set of things it is allowed to use, and a limit it will not cross without asking you."
      />

      <section aria-labelledby="on-staff">
        <h2
          id="on-staff"
          class="text-2xs text-ink-soft pb-3 font-semibold tracking-wide uppercase"
        >
          On staff · {onStaff.length}
        </h2>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {onStaff.map((agent) => (
            <AgentCard key={agent.slug} agent={agent} />
          ))}
        </div>
      </section>

      <section aria-labelledby="available" class="pt-9">
        <h2
          id="available"
          class="text-2xs text-ink-soft font-semibold tracking-wide uppercase"
        >
          You could also hire · {available.length}
        </h2>
        <p class="text-ink-mid max-w-[62ch] pt-1 pb-3 text-base">
          Included on the Company plan. Nobody starts work until you say so.
        </p>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {available.map((agent) => (
            <AgentCard key={agent.slug} agent={agent} />
          ))}
        </div>
      </section>
    </div>
  );
});
