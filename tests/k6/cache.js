import http from "k6/http";
import { check } from "k6";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const USER_ID = Number(__ENV.USER_ID || 1);
const LIMIT = Number(__ENV.LIMIT || 10);

export const options = {
  discardResponseBodies: false,
  scenarios: {
    cache_effectiveness: {
      executor: "constant-arrival-rate",
      rate: Number(__ENV.RATE || 40),
      timeUnit: "1s",
      duration: __ENV.DURATION || "45s",
      preAllocatedVUs: Number(__ENV.PRE_VUS || 20),
      maxVUs: Number(__ENV.MAX_VUS || 60),
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<150", "p(99)<300"],
    checks: ["rate>0.99"],
  },
};

export function setup() {
  const warmupUrl = `${BASE_URL}/users/${USER_ID}/recommendations?limit=${LIMIT}`;
  const res = http.get(warmupUrl, {
    tags: { endpoint: "single_recommendation_cache_warmup" },
  });

  return {
    warmed: res.status === 200 || res.status === 503,
  };
}

export default function (data) {
  const url = `${BASE_URL}/users/${USER_ID}/recommendations?limit=${LIMIT}`;
  const res = http.get(url, {
    tags: { endpoint: "single_recommendation_cache" },
  });

  const body = safeJson(res);

  check(res, {
    "warmup completed": () => data?.warmed === true,
    "status is 200": () => res.status === 200,
    "cache_hit is true": () => body?.metadata?.cache_hit === true,
    "recommendations array exists": () => Array.isArray(body?.recommendations),
  });
}

function safeJson(res) {
  try {
    return res.json();
  } catch (_) {
    return null;
  }
}
