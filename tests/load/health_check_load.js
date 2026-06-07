import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

const healthCounter = new Counter('health_checks');
const healthErrors = new Rate('health_errors');
const healthDuration = new Trend('health_duration');

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
    executor: 'constant-vus',
    vus: 50,
    duration: '2m',
    thresholds: {
        http_req_duration: ['p(95)<50'],
        health_errors: ['rate<0.001'],
    },
};

export default function () {
    const start = Date.now();
    const res = http.get(`${BASE_URL}/api/v1/health`);
    const duration = Date.now() - start;

    healthDuration.add(duration);

    const success = check(res, {
        'status is 200': (r) => r.status === 200,
        'body has status': (r) => {
            const body = JSON.parse(r.body);
            return body.status === 'healthy';
        },
        'response time < 100ms': (r) => r.timings.duration < 100,
    });

    if (!success) {
        healthErrors.add(1);
    } else {
        healthCounter.add(1);
    }

    sleep(0.02);
}
