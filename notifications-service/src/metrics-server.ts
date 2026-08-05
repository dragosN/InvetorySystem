import { config } from "./config";
import { registry } from "./metrics";

export function startMetricsServer(): ReturnType<typeof Bun.serve> {
  const server = Bun.serve({
    port: config.metricsPort,
    async fetch(req) {
      const url = new URL(req.url);
      if (req.method === "GET" && url.pathname === "/metrics") {
        return new Response(await registry.metrics(), {
          headers: { "Content-Type": registry.contentType },
        });
      }
      if (req.method === "GET" && url.pathname === "/healthz") {
        return Response.json({ status: "ok" });
      }
      return new Response("not found", { status: 404 });
    },
  });

  console.log(
    JSON.stringify({
      msg: "notifications metrics listening",
      url: `http://localhost:${server.port}/metrics`,
    }),
  );

  return server;
}
