import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

const metricsCounter = new Counter('metrics_requests');
const metricsErrors = new Rate('metrics_errors');
const metricsDuration = new Trend('metrics_duration');

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
    executor: 'constant-vus',
    vus: 10,
    duration: '1m',
    thresholds: {
        http_req_duration: ['p(95)<200'],
        metrics_errors: ['rate<0.001'],
    },
};

export default function () {
    const start = Date.now();
    const res = http.get(`${BASE_URL}/metrics`);
    const duration = Date.now() - start;

    metricsDuration.add(duration);

    const success = check(res, {
        'status is 200': (r) => r.status === 200,
        'body contains metrics': (r) => r.body.includes('fintech_'),
        'response time < 500ms': (r) => r.timings.duration < 500,
    });

    if (!success) {
        metricsErrors.add(1);
    } else {
        metricsCounter.add(1);
    }

    sleep(0.1);
}
