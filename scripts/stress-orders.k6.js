import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";

/**
 * Stress / load test for POST /orders.
 *
 * Run (local k6):
 *   k6 run scripts/stress-orders.k6.js
 *   k6 run -e VUS=50 -e DURATION=60s scripts/stress-orders.k6.js
 *
 * Run (Docker, no local install):
 *   make stress
 *
 * Watch Grafana while it runs:
 *   http://localhost:3000/d/ecommerce-observability
 */

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const SHARED_CLIENT = __ENV.SHARED_CLIENT === "1";

const orderFailRate = new Rate("order_fail_rate");
const orderCreateMs = new Trend("order_create_duration_ms", true);

export const options = {
  scenarios: {
    stress: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: __ENV.RAMP_UP || "15s", target: Number(__ENV.VUS || 25) },
        { duration: __ENV.DURATION || "45s", target: Number(__ENV.VUS || 25) },
        { duration: __ENV.RAMP_DOWN || "10s", target: 0 },
      ],
      gracefulRampDown: "10s",
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.05"],
    http_req_duration: ["p(95)<100"],
    order_fail_rate: ["rate<0.1"],
  },
};

function clientId(vu, iter) {
  if (SHARED_CLIENT) {
    return "k6-shared";
  }
  return `k6-${vu}-${iter}`;
}

export default function () {
  const payload = JSON.stringify({
    items: [
      {
        sku: `K6-${__VU}-${__ITER}`,
        quantity: 1,
        unit_price: 100,
      },
    ],
  });

  const res = http.post(`${BASE_URL}/orders`, payload, {
    headers: {
      "Content-Type": "application/json",
      "X-Client-Id": clientId(__VU, __ITER),
    },
    tags: { name: "POST /orders" },
  });

  const ok = check(res, {
    "status is 201": (r) => r.status === 201,
    "has order id": (r) => {
      try {
        return Boolean(r.json("id"));
      } catch (_) {
        return false;
      }
    },
  });

  orderFailRate.add(!ok);
  orderCreateMs.add(res.timings.duration);

  // Tiny think time so we don't only measure pure open-loop hammering
  sleep(Number(__ENV.THINK || 0.05));
}

export function handleSummary(data) {
  const reqs = data.metrics.http_reqs;
  const dur = data.metrics.http_req_duration;
  const fails = data.metrics.http_req_failed;
  const values = (dur && dur.values) || {};
  const p50 = values["p(50)"] ?? values.med;
  const p95 = values["p(95)"];
  const max = values.max;

  console.log("\n=== stress summary ===");
  console.log(`requests:     ${reqs ? reqs.values.count : 0}`);
  console.log(
    `fail rate:    ${fails ? (fails.values.rate * 100).toFixed(2) : 0}%`,
  );
  if (p50 != null) console.log(`latency p50:  ${Number(p50).toFixed(1)} ms`);
  if (p95 != null) console.log(`latency p95:  ${Number(p95).toFixed(1)} ms`);
  if (max != null) console.log(`latency max:  ${Number(max).toFixed(1)} ms`);
  console.log("Grafana: http://localhost:3000/d/ecommerce-observability\n");

  return {};
}
