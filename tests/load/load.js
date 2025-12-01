import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 50 },   // Ramp up to 50 VUs
    { duration: '1m', target: 100 },   // Stay at 100 VUs for 1 min
    { duration: '30s', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_reqs: ['rate>100'], // > 100 req/s
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/api/v1';

export function setup() {
  // Create test accounts for transfers
  const loginRes = http.post(`${BASE_URL}/auth/login`, JSON.stringify({
    email: 'operator@goledger.io',
    password: 'operator123',
  }), {
    headers: { 'Content-Type': 'application/json' },
  });

  const body = JSON.parse(loginRes.body);
  const token = body.token;

  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
  };

  // Create two accounts for transfers
  // From account: allow negative (can send even with $0)
  // To account: allow positive (can receive)
  const acc1Res = http.post(`${BASE_URL}/accounts`, JSON.stringify({
    name: 'Load Test Account 1',
    currency: 'USD',
    allow_negative_balance: true,
    allow_positive_balance: true,
  }), { headers });

  const acc2Res = http.post(`${BASE_URL}/accounts`, JSON.stringify({
    name: 'Load Test Account 2',
    currency: 'USD',
    allow_negative_balance: true,
    allow_positive_balance: true,
  }), { headers });

  const acc1 = JSON.parse(acc1Res.body);
  const acc2 = JSON.parse(acc2Res.body);

  return {
    token,
    account1: acc1.id,
    account2: acc2.id,
  };
}

export default function (data) {
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${data.token}`,
    'X-Idempotency-Key': `load-test-${__VU}-${__ITER}`,
  };

  // Mix of operations
  const operations = [
    // Create transfer
    () => {
      const transferRes = http.post(`${BASE_URL}/transfers`, JSON.stringify({
        from_account_id: data.account1,
        to_account_id: data.account2,
        amount: '10.00',
        metadata: {
          test: true,
          vu: __VU,
          iter: __ITER,
        },
      }), { headers });

      check(transferRes, {
        'transfer created': (r) => r.status === 201,
      });
    },
    // List transfers
    () => {
      const listRes = http.get(
        `${BASE_URL}/accounts/${data.account1}/transfers?limit=50`,
        { headers }
      );

      check(listRes, {
        'list transfers': (r) => r.status === 200,
      });
    },
    // Get account
    () => {
      const accRes = http.get(`${BASE_URL}/accounts/${data.account1}`, { headers });
      
      check(accRes, {
        'get account': (r) => r.status === 200,
      });
    },
  ];

  // Randomly pick an operation
  const op = operations[Math.floor(Math.random() * operations.length)];
  op();

  sleep(0.5);
}
