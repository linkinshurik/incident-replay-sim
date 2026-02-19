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

  const replayStatus = http.get(`${BASE_URL}/replay/status`);
  check(replayStatus, {
    "/replay/status without runId returns 400": (r) => r.status === 400,
  });

  sleep(1);
}
