/* SCRATCH — screenshot harness. Deleted before commit. */
import { $, component$, type QRL } from "@builder.io/qwik";
import { useLocation } from "@builder.io/qwik-city";
import { Workspace } from "~/components/workspace/workspace";
import { FrontDesk } from "~/components/workspace/screens/front-desk";
import { AgentDirectory } from "~/components/workspace/screens/agent-directory";
import { AgentProfile } from "~/components/workspace/screens/agent-profile";
import { TeamsBoard } from "~/components/workspace/screens/teams-board";
import { OrganizationPanel } from "~/components/workspace/screens/organization-panel";
import { SettingsPanel } from "~/components/workspace/screens/settings-panel";
import { ChatApp } from "~/components/chat/chat-app";
import { agentBySlug } from "~/lib/mock/staff";

const SESSION = {
  user: { name: "Ana Rivas", email: "ana@witsaba.com", image: null },
};
const signOut = {
  submit: $((_fd: FormData) => Promise.resolve()) as QRL<
    (formData: FormData) => unknown
  >,
  actionPath: "/auth/signout",
};

export default component$(() => {
  const loc = useLocation();
  const which = loc.url.searchParams.get("s") ?? "home";
  const section =
    which === "chat"
      ? "chat"
      : which === "agents" || which === "profile"
        ? "agents"
        : which === "teams"
          ? "teams"
          : which === "org"
            ? "organization"
            : which === "settings"
              ? "settings"
              : "home";
  return (
    <Workspace
      section={section as never}
      session={SESSION as never}
      signOut={signOut as never}
      fills={which === "chat"}
    >
      {which === "agents" ? (
        <AgentDirectory />
      ) : which === "profile" ? (
        <AgentProfile agent={agentBySlug("finance")!} />
      ) : which === "teams" ? (
        <TeamsBoard />
      ) : which === "org" ? (
        <OrganizationPanel name="Ana Rivas" email="ana@witsaba.com" />
      ) : which === "settings" ? (
        <SettingsPanel user={SESSION.user} />
      ) : which === "chat" ? (
        <ChatApp youName="Ana Rivas" youEmail="ana@witsaba.com" />
      ) : (
        <FrontDesk name="Ana Rivas" />
      )}
    </Workspace>
  );
});
