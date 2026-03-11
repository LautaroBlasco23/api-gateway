import http from 'k6/http';
import { check } from 'k6';

export const options = { vus: 1, iterations: 1 };

const GW = 'http://localhost:8080';

export default function () {
  // Proxy pass-through
  check(http.get(`${GW}/echo`), {
    'proxied 200': r => r.status === 200,
  });

  // CORS preflight
  check(
    http.options(`${GW}/echo`, null, {
      headers: {
        'Origin': 'http://example.com',
        'Access-Control-Request-Method': 'GET',
      },
    }),
    { 'CORS 204': r => r.status === 204 }
  );

  // Validation — valid payload
  check(
    http.post(
      `${GW}/echo/users`,
      JSON.stringify({ email: 'test@test.com', username: 'user1', password: 'pass1234' }),
      { headers: { 'Content-Type': 'application/json' } }
    ),
    { 'valid payload 200': r => r.status === 200 }
  );

  // Validation — invalid email
  check(
    http.post(
      `${GW}/echo/users`,
      JSON.stringify({ email: 'not-an-email', username: 'user1', password: 'pass1234' }),
      { headers: { 'Content-Type': 'application/json' } }
    ),
    { 'invalid email 400': r => r.status === 400 }
  );

  // Injection detection
  check(
    http.get(`${GW}/echo?q=%3Cscript%3Ealert(1)%3C%2Fscript%3E`),
    { 'injection blocked 400': r => r.status === 400 }
  );

  // No registered route
  check(http.get(`${GW}/nonexistent`), {
    'no route 400': r => r.status === 400,
  });
}
