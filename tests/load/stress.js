import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";

const errorRate = new Rate("errors");
const transferDuration = new Trend("transfer_duration", true);

export const options = {
  scenarios: {
    stress: {
      executor: "ramping-arrival-rate",
      startRate: 100,
      timeUnit: "1s",
      preAllocatedVUs: 500,
      maxVUs: 2000,
      stages: [
        { duration: "30s", target: 1000 },
        { duration: "1m", target: 5000 },
        { duration: "2m", target: 10000 },
        { duration: "1m", target: 10000 },
        { duration: "30s", target: 0 },
      ],
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.05"],
    http_req_duration: ["p(95)<500", "p(99)<1000"],
    errors: ["rate<0.05"],
  },
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

export function setup() {
  const res = http.post(
    `${BASE_URL}/v1/auth/login`,
    JSON.stringify({ email: "load@test.com", password: "loadtest123" }),
    { headers: { "Content-Type": "application/json" } }
  );
  if (res.status === 200) {
    return { token: res.json("token") };
  }
  return { token: "" };
}

export default function (data) {
  const headers = {
    "Content-Type": "application/json",
    Authorization: `Bearer ${data.token}`,
  };

  const payload = JSON.stringify({
    source_wallet_id: __ENV.SOURCE_WALLET_ID || "00000000-0000-0000-0000-000000000001",
    destination_wallet_id: __ENV.DEST_WALLET_ID || "00000000-0000-0000-0000-000000000002",
    amount_cents: 100,
    description: "stress test",
    idempotency_key: `k6-stress-${__VU}-${__ITER}`,
  });

  const start = Date.now();
  const res = http.post(`${BASE_URL}/v1/transactions/transfer`, payload, { headers });
  transferDuration.add(Date.now() - start);

  const ok = check(res, {
    "status 2xx": (r) => r.status >= 200 && r.status < 300,
  });
  errorRate.add(!ok);
}

export function handleSummary(data) {
  return {
    "tests/load/results/stress-summary.json": JSON.stringify(data, null, 2),
    stdout: textSummary(data, { indent: "  ", enableColors: true }),
  };
}

function textSummary(data, opts) {
  const lines = [];
  lines.push("=== Stress Test Summary ===");
  lines.push(`Requests: ${data.metrics.http_reqs.values.count}`);
  lines.push(`Error rate: ${(data.metrics.http_req_failed.values.rate * 100).toFixed(2)}%`);
  lines.push(`p50: ${data.metrics.http_req_duration.values["p(50)"].toFixed(2)}ms`);
  lines.push(`p95: ${data.metrics.http_req_duration.values["p(95)"].toFixed(2)}ms`);
  lines.push(`p99: ${data.metrics.http_req_duration.values["p(99)"].toFixed(2)}ms`);
  return lines.join("\n");
}
