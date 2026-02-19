import http from "k6/http";
import { check, sleep } from "k6";

const BASE_URL = __ENV.BASE_URL || "http://127.0.0.1:8080";

export const options = {
  vus: 1,
  duration: "30s",
  thresholds: {
    http_req_failed: ["rate<0.35"],
  },
};

export default function () {
  const h = http.get(`${BASE_URL}/healthz`);
  check(h, { "healthz 200": (r) => r.status === 200 });

  const m = http.get(`${BASE_URL}/metrics`);
  check(m, { "metrics 200": (r) => r.status === 200 });

  // Start replay with targetBaseUrl http://127.0.0.1:8080, rps 5, durationSec 5
  const startPayload = JSON.stringify({
    scenarioId: "smoke-test",
    targetBaseUrl: "http://127.0.0.1:8080",
    rps: 5,
    durationSec: 5
  });

  const startRes = http.post(`${BASE_URL}/replay/start`, startPayload, { headers: { "Content-Type": "application/json" } });
  check(startRes, { "/replay/start 200": (r) => r.status === 200 });

  if (startRes.status !== 200) {
    return;
  }

  let runId = "";
  try {
    const startData = startRes.json();
    runId = startData.runId;
  } catch (e) {
    return;
  }

  // Poll /replay/status until state != running
  while (true) {
    const statusRes = http.get(`${BASE_URL}/replay/status?runId=${runId}`);
    if (!check(statusRes, { "/replay/status 200": (r) => r.status === 200 })) {
      break;
    }

    let state = "";
    try {
      const statusData = statusRes.json();
      state = statusData.state || "";
    } catch (e) {
      break;
    }

    if (state !== "running") {
      break;
    }

    sleep(1);
  }

  // Stop replay
  const stopPayload = JSON.stringify({ runId });
  const stopRes = http.post(`${BASE_URL}/replay/stop`, stopPayload, { headers: { "Content-Type": "application/json" } });
  check(stopRes, { "/replay/stop 200": (r) => r.status === 200 });

  sleep(1);
}
