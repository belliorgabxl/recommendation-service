import http from "k6/http";
import { check } from "k6";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const LIMIT = Number(__ENV.LIMIT || 10);
const USER_COUNT = Number(__ENV.USER_COUNT || 200);

export const options = {
  discardResponseBodies: false,
  scenarios: {
    single_recommendation_load: {
      executor: "constant-arrival-rate",
      rate: Number(__ENV.RATE || 60),
      timeUnit: "1s",
      duration: __ENV.DURATION || "1m",
      preAllocatedVUs: Number(__ENV.PRE_VUS || 30),
      maxVUs: Number(__ENV.MAX_VUS || 120),
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.03"],
    http_req_duration: ["p(95)<500", "p(99)<900"],
    checks: ["rate>0.97"],
  },
};

export default function () {
  const userId = 1 + Math.floor(Math.random() * USER_COUNT);
  const url = `${BASE_URL}/users/${userId}/recommendations?limit=${LIMIT}`;

  const res = http.get(url, {
    tags: { endpoint: "single_recommendation" },
  });

  const okStatus = res.status === 200 || res.status === 503;
  const body = safeJson(res);

  check(res, {
    "status is 200 or 503": () => okStatus,
    "200 response has recommendations array": () =>
      res.status !== 200 || Array.isArray(body?.recommendations),
    "200 response has metadata": () => res.status !== 200 || !!body?.metadata,
  });
}

function safeJson(res) {
  try {
    return res.json();
  } catch (_) {
    return null;
  }
}
