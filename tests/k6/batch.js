import http from 'k6/http';
import { check } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const PAGE_LIMIT = Number(__ENV.PAGE_LIMIT || 20);
const TOTAL_USERS = Number(__ENV.USER_COUNT || 200);
const TOTAL_PAGES = Math.max(1, Math.ceil(TOTAL_USERS / PAGE_LIMIT));

export const options = {
  discardResponseBodies: false,
  scenarios: {
    batch_recommendation_stress: {
      executor: 'constant-arrival-rate',
      rate: Number(__ENV.RATE || 12),
      timeUnit: '1s',
      duration: __ENV.DURATION || '1m',
      preAllocatedVUs: Number(__ENV.PRE_VUS || 20),
      maxVUs: Number(__ENV.MAX_VUS || 80),
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.02'],
    http_req_duration: ['p(95)<1500', 'p(99)<2500'],
    checks: ['rate>0.98'],
  },
};

export default function () {
  const page = 1 + Math.floor(Math.random() * TOTAL_PAGES);
  const url = `${BASE_URL}/recommendations/batch?page=${page}&limit=${PAGE_LIMIT}`;

  const res = http.get(url, {
    tags: { endpoint: 'batch_recommendation' },
  });

  const body = safeJson(res);

  check(res, {
    'status is 200': () => res.status === 200,
    'response has results': () => Array.isArray(body?.results),
    'response has summary': () => !!body?.summary,
    'response page matches': () => body?.page === page,
  });
}

function safeJson(res) {
  try {
    return res.json();
  } catch (_) {
    return null;
  }
}