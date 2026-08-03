import { config } from "./config";

const deliveries: Array<{
  received_at: string;
  headers: Record<string, string>;
  body: unknown;
}> = [];

const server = Bun.serve({
  port: config.mockReceiverPort,
  async fetch(req) {
    const url = new URL(req.url);

    if (req.method === "GET" && url.pathname === "/healthz") {
      return Response.json({ status: "ok" });
    }

    if (req.method === "GET" && url.pathname === "/deliveries") {
      return Response.json({ count: deliveries.length, deliveries });
    }

    if (req.method === "DELETE" && url.pathname === "/deliveries") {
      deliveries.length = 0;
      return Response.json({ cleared: true });
    }

    if (req.method === "POST" && url.pathname === "/webhook") {
      const headers: Record<string, string> = {};
      req.headers.forEach((value, key) => {
        headers[key] = value;
      });

      let body: unknown = null;
      try {
        body = await req.json();
      } catch {
        body = await req.text();
      }

      const entry = {
        received_at: new Date().toISOString(),
        headers,
        body,
      };
      deliveries.push(entry);

      console.log(
        JSON.stringify({
          msg: "mock webhook received",
          event_id: headers["x-event-id"],
          body,
        }),
      );

      return Response.json({ received: true }, { status: 200 });
    }

    return new Response("not found", { status: 404 });
  },
});

console.log(
  JSON.stringify({
    msg: "mock webhook receiver listening",
    url: `http://localhost:${server.port}/webhook`,
    deliveries: `http://localhost:${server.port}/deliveries`,
  }),
);
