/**
 * This is the base config for vite.
 * When building, the adapter config is used which loads this file and extends it.
 */
import { defineConfig, type UserConfig } from "vitest/config";
import { qwikVite } from "@builder.io/qwik/optimizer";
import { qwikCity } from "@builder.io/qwik-city/vite";
import tsconfigPaths from "vite-tsconfig-paths";
import pkg from "./package.json";
import tailwindcss from "@tailwindcss/vite";
type PkgDep = Record<string, string>;
const { dependencies = {}, devDependencies = {} } = pkg as any as {
  dependencies: PkgDep;
  devDependencies: PkgDep;
  [key: string]: unknown;
};
errorOnDuplicatesPkgDeps(devDependencies, dependencies);
/**
 * Note that Vite normally starts from `index.html` but the qwikCity plugin makes start at `src/entry.ssr.tsx` instead.
 */

export default defineConfig(({ command, mode }): UserConfig => {
  return {
        plugins: [
          qwikCity(),
          qwikVite(),
          tsconfigPaths({ root: "." }),
          tailwindcss(),
          // Inline plugin: stub `postgres` (and its sub-paths) for the CLIENT
          // build only. The SSR build (`-c adapters/node-server/vite.config.ts`)
          // sees the real package; the client build sees a no-op stub that
          // throws if accidentally called from the browser.
          //
          // Why this is required:
          //   `frontend/src/lib/db.ts` does
          //     `const { default: postgres } = await import("postgres")`
          //   behind `if (!import.meta.env.SSR)` so the import is unreachable
          //   in the client bundle at runtime. Rollup is conservative about
          //   dynamic imports and walks `postgres`'s module graph anyway,
          //   hitting `import { performance } from "perf_hooks"` and
          //   failing the client build with
          //     "performance" is not exported by "__vite-browser-external".
          //   This plugin intercepts the resolveId before Rollup sees the
          //   package, keeping both graphs (client and SSR) sound.
          {
            name: "cachicamas-stub-server-only-deps",
            enforce: "pre",
            resolveId(id, _importer, options) {
              if (
                options?.ssr === false &&
                (id === "postgres" || id.startsWith("postgres/"))
              ) {
                return {
                  id: "\0cachicamas-postgres-client-stub",
                  moduleSideEffects: false,
                };
              }
              return null;
            },
            load(id) {
              if (id === "\0cachicamas-postgres-client-stub") {
                return [
                  "// Auto-generated stub. `postgres` is server-only;",
                  "// see vite.config.ts and frontend/src/lib/db.ts.",
                  "function notInBrowser() {",
                  "  throw new Error(",
                  '    "[cachicamas] postgres is server-only. The events.signIn callback must not run in the browser context."',
                  "  );",
                  "}",
                  "export default notInBrowser;",
                  "export { notInBrowser as postgres };",
                ].join("\n");
              }
              return null;
            },
          },
        ],
    // Vitest picks up `**/*.spec.{ts,tsx}` by default; the e2e
    // tests live under `frontend/e2e/` and use Playwright's
    // `test()` API, which collides with Vitest's globals.  Keep
    // them out of `pnpm test:ci`.
    test: {
      exclude: [
        "e2e/**",
        "node_modules/**",
        "dist/**",
        ".rollup.cache/**",
      ],
    },
    // This tells Vite which dependencies to pre-build in dev mode.
        optimizeDeps: {
          // Put problematic deps that break bundling here, mostly those with binaries.
          // For example ['better-sqlite3'] if you use that in server functions.
          //
          // cachicamas-github-login events.signIn (PR-followup): `postgres`
          // uses Node built-ins (perf_hooks, stream, crypto) that do not
          // exist in the browser. Skip dev pre-bundling so Vite does not
          // try to resolve its module graph in the browser context. The
          // CLIENT build still walks the graph; the
          // `cachicamas-stub-server-only-deps` plugin (above) handles
          // that by resolving `postgres` to a stub for non-SSR builds.
          exclude: ["postgres"],
          // Pre-bundle the auth stack so dev-mode cold start doesn't choke on
          // @auth/qwik's deep ESM graph. Mirrors the project's pre-build list
          // discipline. (cachicamas-github-login PR-2)
          include: ["@auth/qwik", "@auth/core", "@panva/hkdf"],
        },
    /**
     * This is an advanced setting. It improves the bundling of your server code. To use it, make sure you understand when your consumed packages are dependencies or dev dependencies. (otherwise things will break in production)
     */
    // ssr:
    //   command === "build" && mode === "production"
    //     ? {
    //         // All dev dependencies should be bundled in the server build
    //         noExternal: Object.keys(devDependencies),
    //         // Anything marked as a dependency will not be bundled
    //         // These should only be production binary deps (including deps of deps), CLI deps, and their module graph
    //         // If a dep-of-dep needs to be external, add it here
    //         // For example, if something uses `bcrypt` but you don't have it as a dep, you can write
    //         // external: [...Object.keys(dependencies), 'bcrypt']
    //         external: Object.keys(dependencies),
    //       }
    //     : undefined,
    server: {
      headers: {
        // Don't cache the server response in dev mode
        "Cache-Control": "public, max-age=0",
      },
    },
    preview: {
      headers: {
        // Do cache the server response in preview (non-adapter production build)
        "Cache-Control": "public, max-age=600",
      },
    },
  };
});
// *** utils ***
/**
 * Function to identify duplicate dependencies and throw an error
 * @param {Object} devDependencies - List of development dependencies
 * @param {Object} dependencies - List of production dependencies
 */
function errorOnDuplicatesPkgDeps(
  devDependencies: PkgDep,
  dependencies: PkgDep,
) {
  let msg = "";
  // Create an array 'duplicateDeps' by filtering devDependencies.
  // If a dependency also exists in dependencies, it is considered a duplicate.
  const duplicateDeps = Object.keys(devDependencies).filter(
    (dep) => dependencies[dep],
  );
  // include any known qwik framework packages. `@auth/qwik` is NOT a
  // framework package (it's an auth library that runs at runtime in the
  // Node SSR), so we explicitly exclude it from this check.
  const qwikPkg = Object.keys(dependencies).filter(
    (value) =>
      /^@builder\.io\/qwik/.test(value) ||
      // Catch the framework-only qwik names that don't start with @builder.io.
      value === "qwik" ||
      value === "qwik-city",
  );
  // any errors for missing "qwik-city-plan"
  // [PLUGIN_ERROR]: Invalid module "@qwik-city-plan" is not a valid package
  msg = `Move qwik packages ${qwikPkg.join(", ")} to devDependencies`;
  if (qwikPkg.length > 0) {
    throw new Error(msg);
  }
  // Format the error message with the duplicates list.
  // The `join` function is used to represent the elements of the 'duplicateDeps' array as a comma-separated string.
  msg = `
    Warning: The dependency "${duplicateDeps.join(", ")}" is listed in both "devDependencies" and "dependencies".
    Please move the duplicated dependencies to "devDependencies" only and remove it from "dependencies"
  `;
  // Throw an error with the constructed message.
  if (duplicateDeps.length > 0) {
    throw new Error(msg);
  }
}
