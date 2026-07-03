/**
 * Node-server adapter config para Qwik City.
 *
 * Genera un bundle Node que sirve SSR (Qwik City router + render).
 * El entry point es `src/entry.express.tsx`, que además implementa el
 * reverse proxy /api/* hacia el Go bin y sirve los assets estáticos.
 *
 * Fuente: https://qwik.dev/docs/deployments/node/
 *
 * Se invoca vía `pnpm build.server` (definido en package.json).
 */
import { nodeServerAdapter } from "@builder.io/qwik-city/adapters/node-server/vite";
import { extendConfig } from "@builder.io/qwik-city/vite";
import baseConfig from "../../vite.config";

// `vite.config.ts` raíz importa `defineConfig` de `vitest/config` para tener
// los tipos del bloque `test`. Eso crea una incompatibilidad estructural
// (vitest UserConfig vs vite UserConfig) con `extendConfig` de Qwik, que
// espera la versión de vite. El cast resuelve sin afectar runtime.
//
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const baseAsViteConfig = baseConfig as any;

export default extendConfig(baseAsViteConfig, () => {
  return {
    build: {
      ssr: true,
      rollupOptions: {
        input: ["src/entry.express.tsx", "@qwik-city-plan"],
      },
    },
    plugins: [
      nodeServerAdapter({
        name: "node-server",
        // `ssg: null` deshabilita la generación de HTML estático en build
        // time. El server SSR-renderiza cada request, así que no
        // necesitamos HTML pregenerado (que viene con `q:container="paused"`
        // y rompe el runtime del browser).
        ssg: null,
        // El `origin` lo lee el adapter de process.env.ORIGIN o process.env.URL.
        // Lo dejamos en undefined acá — el server bundle usa el ORIGIN que
        // pasemos al runtime (via env var ORIGIN en el Dockerfile).
      }),
    ],
  };
});
