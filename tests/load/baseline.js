import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";

const errorRate = new Rate("errors");
const transferDuration = new Trend("transfer_duration", true);

export const options = {
  scenarios: {
    baseline: {
      executor: "constant-arrival-rate",
      rate: 1000,
      timeUnit: "1s",
      duration: "2m",
      preAllocatedVUs: 200,
      maxVUs: 500,
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(99)<200"],
    errors: ["rate<0.01"],
  },
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

let authToken = "";

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
    description: "load test",
    idempotency_key: `k6-${__VU}-${__ITER}`,
  });

  const start = Date.now();
  const res = http.post(`${BASE_URL}/v1/transactions/transfer`, payload, { headers });
  transferDuration.add(Date.now() - start);

  const ok = check(res, {
    "status 200 or 201": (r) => r.status === 200 || r.status === 201,
    "has transaction_id": (r) => r.json("transaction_id") !== undefined,
  });
  errorRate.add(!ok);

  sleep(0.001);
}
