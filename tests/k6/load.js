import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    ramp_up: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '30s', target: 20 },
        { duration: '1m',  target: 20 },
        { duration: '15s', target: 0  },
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500'],  // 95% of requests under 500ms
    http_req_failed:   ['rate<0.05'],  // less than 5% errors (excludes expected 429s)
  },
};

const GW = 'http://localhost:8080';

export default function () {
  const r = http.get(`${GW}/echo`);
  check(r, {
    'status is 200 or 429': res => res.status === 200 || res.status === 429,
  });
  sleep(0.1);
}
