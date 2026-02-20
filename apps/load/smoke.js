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
  // Prepare scenario events with timestamps: now, now+1s, now+2s
  const now = new Date();
  const events = [
    { ts: now.toISOString(), method: "GET", path: "/debug/echo" },
    { ts: new Date(now.getTime() + 1000).toISOString(), method: "GET", path: "/debug/echo" },
    { ts: new Date(now.getTime() + 2000).toISOString(), method: "GET", path: "/debug/echo" },
  ];

  // Upload scenario with 3 events
  const scenarioId = "smoke-upload";
  const jsonl = events.map(e => JSON.stringify({ ts: e.ts, type: "http", method: e.method, path: e.path })).join("\n");
  const uploadPayload = JSON.stringify({ scenarioId, jsonl });

  const uploadRes = http.post(`${BASE_URL}/scenarios/upload`, uploadPayload, { headers: { "Content-Type": "application/json" } });
  check(uploadRes, { "/scenarios/upload 200": (r) => r.status === 200 });
  if (uploadRes.status !== 200) {
    return;
  }

  // Start replay with mode=timestamp, speed=100, maxDelayMs=0 to avoid waiting
  const startPayload = JSON.stringify({
    scenarioId,
    targetBaseUrl: BASE_URL,
    rps: 1000, // high RPS to avoid waiting, or could be 5 anyway
    durationSec: 10,
    mode: "timestamp",
    speed: 100,
    maxDelayMs: 0,
    startFromTs: events[0].ts,
    endAtTs: events[events.length - 1].ts,
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
  let requests = 0;
  while (true) {
    const statusRes = http.get(`${BASE_URL}/replay/status?runId=${runId}`);
    if (!check(statusRes, { "/replay/status 200": (r) => r.status === 200 })) {
      break;
    }

    let state = "";
    try {
      const statusData = statusRes.json();
      state = statusData.state || "";
      requests = (statusData.stats && statusData.stats.requests) || 0;
    } catch (e) {
      break;
    }

    if (state !== "running") {
      // Assert requests > 0 and state != running
      check({ requests, state }, {
        "requests > 0": (obj) => obj.requests > 0,
        "state != running": (obj) => obj.state !== "running",
      });
      break;
    }

    sleep(1);
  }

  // Stop replay
  const stopPayload = JSON.stringify({ runId });
  const stopRes = http.post(`${BASE_URL}/replay/stop`, stopPayload, { headers: { "Content-Type": "application/json" } });
  check(stopRes, { "/replay/stop 200": (r) => r.status === 200 });

  // After stopping, GET /metrics and assert response body contains replay metrics
  const metrics = http.get(`${BASE_URL}/metrics`);
  const checkMetrics = check(metrics, {
    "/metrics 200": (r) => r.status === 200,
    "metrics include replay_requests_total": (r) => r.body.includes("replay_requests_total"),
    "metrics include replay_runs_active": (r) => r.body.includes("replay_runs_active"),
  });

  if (!checkMetrics) {
    throw new Error("Metrics missing expected replay metrics");
  }

  // New part: GET /replay/report?runId=... and check response
  if (runId) {
    const reportRes = http.get(`${BASE_URL}/replay/report?runId=${runId}`);
    check(reportRes, {
      "/replay/report 200": (r) => r.status === 200,
      "report body contains runId": (r) => r.body && r.body.includes(runId),
    });

    // GET /replay/runs and assert list not empty
    const runsRes = http.get(`${BASE_URL}/replay/runs?limit=20`);
    check(runsRes, {
      "/replay/runs 200": (r) => r.status === 200,
      "runs list not empty": (r) => {
        try {
          const runsList = r.json();
          return Array.isArray(runsList) && runsList.length > 0;
        } catch (_) {
          return false;
        }
      },
    });
  }

  sleep(1);
}
