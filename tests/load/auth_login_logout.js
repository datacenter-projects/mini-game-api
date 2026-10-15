// load test: login → logout ของหลังบ้าน (docs/modules/agent_auth.md, agent_auth_phase2.md)
//
//   k6 run tests/load/auth_login_logout.js
//   k6 run -e BASE_URL=http://localhost:8282 -e ACCOUNT_COUNT=200 tests/load/auth_login_logout.js
//
// ต้องเปิด server ด้วย .env.test (LOGIN_IP_LIMIT_PER_MINUTE กว้าง เพราะทุก VU ยิงจาก IP เดียว)
// และ seed บัญชีก่อน: scripts/seed_loadtest -count >= จำนวน VU สูงสุด
//
// threshold ด้านล่างเป็นค่าตั้งต้น — ปรับหลังได้ผลรอบแรกบนเครื่องที่ใช้วัดจริง
// login ช้ากว่าเส้นอื่นโดยตั้งใจ: bcrypt cost 12 (~250ms ต่อครั้งต่อ core)

import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, PASSWORD, JSON_HEADERS, accountFor } from './lib/config.js';

export const options = {
  scenarios: {
    login_logout: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 20 },
        { duration: '1m', target: 50 },
        { duration: '30s', target: 0 },
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    checks: ['rate>0.99'],
    'http_req_duration{name:login}': ['p(95)<1500'],
    'http_req_duration{name:logout}': ['p(95)<200'],
  },
};

export default function () {
  const username = accountFor(__VU);

  const login = http.post(
    `${BASE_URL}/api/v1/bo/pb/auth/login`,
    JSON.stringify({ username, password: PASSWORD }),
    { headers: JSON_HEADERS, tags: { name: 'login' } },
  );
  const loggedIn = check(login, {
    'login: HTTP 200': (r) => r.status === 200,
    'login: code 200': (r) => r.status === 200 && r.json('code') === 200,
  });
  if (!loggedIn) {
    sleep(1);
    return;
  }

  const token = login.json('data.token');
  const logout = http.post(`${BASE_URL}/api/v1/bo/pr/auth/logout`, null, {
    headers: { Authorization: `Bearer ${token}` },
    tags: { name: 'logout' },
  });
  check(logout, {
    'logout: code 200': (r) => r.status === 200 && r.json('code') === 200,
  });

  sleep(1);
}
