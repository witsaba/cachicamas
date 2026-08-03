/**
 * WHAT IS THIS FILE?
 *
 * Server entry point for the Qwik City node-server adapter.
 *
 * Architecture:
 *   - Node http server (no Express, no nginx).
 *   - `/api/*` → reverse proxy to the Go binary (same compose network).
 *   - Static assets (Qwik client chunks) served from `dist/`.
 *   - All other routes → Qwik City SSR (handles prerendered + dynamic).
 *
 * Why Node SSR (not Static SSG):
 *   - The app has dynamic routes (`/organizations/{id}/`) where the IDs
 *     are created at runtime by the form submit. Static SSG cannot
 *     handle this — it can only prerender a fixed list of IDs.
 *   - Node SSR runs the route loader on every request, so newly created
 *     orgs are immediately readable.
 *
 * Footprint: ~80MB final image (node:22-alpine + Qwik server bundle,
 * no nginx). Still "light" — the original Node-alpine + nginx estimate
 * was ~150MB.
 *
 * The URL `/api/...` is exposed as the browser's only "API surface"
 * (relative to the Qwik server's origin). The Go binary is NEVER
 * reachable from the browser in the normal flow.
 */
import { createQwikCity } from "@builder.io/qwik-city/middleware/node";
import qwikCityPlan from "@qwik-city-plan";
import { createServer, request as httpRequest } from "node:http";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import { existsSync, statSync, createReadStream } from "node:fs";
import render from "./entry.ssr";
import { setSecurityHeaders, getSecurityHeaders } from "./lib/security-headers";
import { logInternalTarget } from "./lib/log-config";

const PORT = parseInt(process.env.PORT ?? "3000", 10);
const HOST = process.env.HOST ?? "0.0.0.0";
// Target for /api/* reverse proxy. In compose, the Go service is reachable
// at http://database_administrator:8080. Override via env for local dev.
const API_TARGET =
  process.env.API_TARGET ?? "http://database_administrator:8080";
// Where Qwik's static assets (dist/) are at runtime. In the Docker image
// this is /app/dist (set by Dockerfile).
const STATIC_ROOT =
  process.env.STATIC_ROOT ??
  join(dirname(fileURLToPath(import.meta.url)), "..", "dist");

const { router, notFound, staticFile } = createQwikCity({
  render,
  qwikCityPlan,
  // The origin used by Qwik for canonical/OG tags and CSRF validation.
  // Read from ORIGIN env (set by docker-compose) or default to the
  // public-facing URL.
  getOrigin(req) {
    const fromEnv = process.env.ORIGIN;
    if (fromEnv) return fromEnv;
    // Fallback: derive from Host header (works in dev / single-host deploys).
    const host = req.headers["x-forwarded-host"] ?? req.headers.host;
    const proto =
      req.headers["x-forwarded-proto"] ??
      (host?.toString().startsWith("localhost") ? "http" : "https");
    return host ? `${proto}://${host}` : "http://localhost:3000";
  },
});

/**
 * Reverse-proxy a request to the Go binary. The /api prefix is stripped
 * for legacy routes (e.g., /api/organizations → /organizations), but
 * PRESERVED for routes that include /api in their backend path
 * (e.g., /api/v1/identity/signin-callback → /api/v1/identity/signin-callback).
 *
 * Why the exception: the identity callback slice
 * (cachicamas-identity-signin-callback, see ADR 0003) exposes its
 * endpoint under /api/v1/* on the Go side. The Qwik Node SSR can also
 * reach the backend directly via SERVER_API_BASE_URL (compose DNS),
 * but in `pnpm dev` mode the fallback uses the proxy. The proxy
 * therefore forwards /api/v1/* as-is and strips /api only for the
 * legacy routes that the Go binary registers at the root.
 *
 * Uses Node's http.request (no extra deps) to forward the request body
 * and pipe the response back.
 */
function proxyToApi(
  req: import("node:http").IncomingMessage,
  res: import("node:http").ServerResponse,
): void {
  // Path-shape detection: legacy routes (no /api in the backend path)
  // have /api/* stripped; identity-callback-style routes (backend has
  // /api/v1/* paths) keep the /api prefix intact.
  const keepApiPrefix = req.url?.startsWith("/api/v1/") ?? false;
  const newPath = keepApiPrefix
    ? (req.url ?? "/")
    : (req.url?.replace(/^\/api/, "") ?? "/");
  const target = new URL(newPath, API_TARGET);

  const headers = { ...req.headers, host: target.host };
  // Remove the original host header to avoid mismatches.
  delete (headers as Record<string, unknown>)["x-forwarded-host"];

  const proxyReq = httpRequest(
    {
      hostname: target.hostname,
      port: target.port || "80",
      path: target.pathname + target.search,
      method: req.method,
      headers,
    },
    (proxyRes) => {
      // Merge our security headers on top of the upstream response so
      // a permissive backend cannot re-introduce a relaxed header (e.g.
      // X-Frame-Options: SAMEORIGIN). Node's writeHead merges with
      // previously-set headers, but the merge favors the values
      // passed here, so we re-apply ours explicitly.
      const outHeaders = {
        ...proxyRes.headers,
        ...getSecurityHeaders(req),
      };
      res.writeHead(proxyRes.statusCode ?? 502, outHeaders);
      proxyRes.pipe(res);
    },
  );
  proxyReq.on("error", (err) => {
    console.error("API proxy error:", err.message);
    if (!res.headersSent) {
      res.writeHead(502, { "content-type": "text/plain" });
    }
    res.end(`Bad gateway: ${err.message}`);
  });
  req.pipe(proxyReq);
}

/**
 * Serve static assets (JS chunks, CSS, favicon, manifest, q-manifest).
 * We do NOT serve the SSG-prerendered HTML files (dist/<route>/index.html)
 * because those have `q:container="paused"` and would prevent Qwik's
 * runtime from registering the QRL handlers. The Node server RE-RENDERS
 * every request through the Qwik router, so the HTML is always fresh
 * (with `q:render="ssr-dev"` or similar). The staticFile middleware
 * from createQwikCity handles the assets with proper cache headers.
 *
 * CRITICAL: dynamic `import()` in the browser requires the file to be
 * served with a JavaScript MIME type. We set Content-Type based on the
 * extension. Without this, Qwik's runtime chunk loading fails with
 * "Failed to fetch dynamically imported module".
 */
const MIME_TYPES: Record<string, string> = {
  ".js": "application/javascript; charset=utf-8",
  ".mjs": "application/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".json": "application/json; charset=utf-8",
  ".svg": "image/svg+xml; charset=utf-8",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg",
  ".gif": "image/gif",
  ".webp": "image/webp",
  ".ico": "image/x-icon",
  ".woff": "font/woff",
  ".woff2": "font/woff2",
  ".txt": "text/plain; charset=utf-8",
};

function mimeTypeFor(filePath: string): string {
  const ext = filePath.slice(filePath.lastIndexOf(".")).toLowerCase();
  return MIME_TYPES[ext] ?? "application/octet-stream";
}

function serveStaticAssets(
  req: import("node:http").IncomingMessage,
  res: import("node:http").ServerResponse,
  next: () => void,
): void {
  if (!req.url) return next();
  const url = new URL(req.url, `http://${req.headers.host ?? "localhost"}`);
  if (url.pathname.includes("..")) return next(); // path traversal guard
  const filePath = join(STATIC_ROOT, url.pathname);
  if (!existsSync(filePath)) return next();
  const stat = statSync(filePath);
  if (!stat.isFile()) return next();

  // Skip the SSG-prerendered HTML files. They have `q:container="paused"`
  // and break Qwik's runtime when served as-is.
  if (filePath.endsWith(".html")) return next();

  // Cache headers for chunks (Qwik content-hashes filenames).
  if (filePath.includes("/build/")) {
    res.setHeader("Cache-Control", "public, max-age=31536000, immutable");
  }

  // CRITICAL: Set Content-Type so dynamic import() works in the browser.
  res.setHeader("Content-Type", mimeTypeFor(filePath));
  res.setHeader("Content-Length", stat.size);
  createReadStream(filePath).pipe(res);
}

const server = createServer((req, res) => {
  // 0. Security headers (CSP/HSTS/X-Content-Type-Options/etc.) on
  //    every response — runs first so the proxy, the static assets,
  //    the SSR router, and the 404 fallback all carry the headers.
  //    For the proxy, the headers are ALSO re-merged into the upstream
  //    response (see proxyToApi) to override any permissive backend.
  setSecurityHeaders(req, res, () => {
    // 1. /api/* → Go binary reverse proxy.
    if (req.url?.startsWith("/api/")) {
      return proxyToApi(req, res);
    }

    // 2. Static assets (cache headers, fast path).
    serveStaticAssets(req, res, () => {
      // 3. Qwik SSR (handles prerendered + dynamic routes).
      staticFile(req, res, () => {
        router(req, res, () => {
          notFound(req, res, () => {
            // Last resort: 404 plain text.
            if (!res.headersSent) {
              res.writeHead(404, { "content-type": "text/plain" });
            }
            res.end("Not found");
          });
        });
      });
    });
  });
});

server.listen(PORT, HOST, () => {
  console.log(`[qwik-server] listening on http://${HOST}:${PORT}`);
  // Internal topology (API_TARGET = database_administrator:8080) MUST
  // not appear in production logs. Gated behind DEBUG=1 — see
  // lib/log-config.ts and security-response-headers spec REQ-03.
  if (logInternalTarget()) {
    console.log(`[qwik-server] API target: ${API_TARGET}`);
  }
  console.log(`[qwik-server] static root: ${STATIC_ROOT}`);
});
