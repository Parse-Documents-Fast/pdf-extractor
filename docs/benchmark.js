import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: parseInt(__ENV.K6_VUS || '10'),
  duration: __ENV.K6_DURATION || '30s',
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.1'],
  },
};

export default function () {
  const baseUrl = 'http://api:8080';

  // Test /health endpoint
  const healthRes = http.get(`${baseUrl}/health`);
  check(healthRes, {
    'health status 200': (r) => r.status === 200,
    'health has status field': (r) => r.json('status') === 'healthy',
    'health has message': (r) => r.json('message') !== null,
    'health has time': (r) => r.json('time') !== null,
  });

  // Test /metrics endpoint
  const metricsRes = http.get(`${baseUrl}/metrics`);
  check(metricsRes, {
    'metrics status 200': (r) => r.status === 200,
    'metrics has start_time': (r) => r.json('start_time') !== null,
    'metrics has uptime': (r) => r.json('uptime') !== null,
    'metrics has goroutines': (r) => r.json('goroutines') > 0,
    'metrics has memory_mb': (r) => r.json('memory_mb') > 0,
    'metrics has requests_total': (r) => r.json('requests_total') >= 0,
    'metrics has requests_active': (r) => r.json('requests_active') >= 0,
  });

  sleep(1);
}