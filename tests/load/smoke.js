import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 10,
  duration: '30s',
  thresholds: {
    http_req_failed: ['rate<0.01'], // Error rate < 1%
    http_req_duration: ['p(95)<500'], // 95% requests < 500ms
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/api/v1';

// Login once to get token
let token = '';

export function setup() {
  const loginRes = http.post(`${BASE_URL}/auth/login`, JSON.stringify({
    email: 'admin@goledger.io',
    password: 'admin123',
  }), {
    headers: { 'Content-Type': 'application/json' },
  });

  check(loginRes, {
    'login successful': (r) => r.status === 200,
  });

  const body = JSON.parse(loginRes.body);
  return { token: body.token };
}

export default function (data) {
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${data.token}`,
  };

  // Create account
  const accountRes = http.post(`${BASE_URL}/accounts`, JSON.stringify({
    name: `Test Account ${__VU}-${__ITER}`,
    currency: 'USD',
    allow_negative_balance: false,
  }), { headers });

  check(accountRes, {
    'account created': (r) => r.status === 201,
  });

  // List accounts
  const listRes = http.get(`${BASE_URL}/accounts?limit=50`, { headers });
  
  check(listRes, {
    'list accounts success': (r) => r.status === 200,
  });

  sleep(1);
}
