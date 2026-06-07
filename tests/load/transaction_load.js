import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

const txCounter = new Counter('transactions_created');
const errorRate = new Rate('errors');
const txDuration = new Trend('transaction_duration');

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const SCENARIO = __ENV.SCENARIO || 'baseline';

const scenarios = {
    baseline: {
        executor: 'constant-vus',
        vus: 100,
        duration: '2m',
    },
    spike: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '30s', target: 100 },
            { duration: '10s', target: 5000 },
            { duration: '1m', target: 5000 },
            { duration: '10s', target: 100 },
            { duration: '30s', target: 100 },
        ],
    },
    stress: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '1m', target: 500 },
            { duration: '2m', target: 1000 },
            { duration: '2m', target: 2000 },
            { duration: '1m', target: 500 },
        ],
    },
    soak: {
        executor: 'constant-vus',
        vus: 200,
        duration: '10m',
    },
    throughput: {
        executor: 'constant-arrival-rate',
        rate: 16700,
        timeUnit: '1s',
        duration: '2m',
        preAllocatedVUs: 500,
        maxVUs: 2000,
    },
};

export const options = {
    scenarios: {
        [SCENARIO]: scenarios[SCENARIO] || scenarios.baseline,
    },
    thresholds: {
        http_req_duration: ['p(95)<200', 'p(99)<500'],
        errors: ['rate<0.01'],
        http_req_failed: ['rate<0.01'],
    },
};

function generateTransaction() {
    const id = Math.random().toString(36).substring(2, 15) + Date.now().toString(36);
    const types = ['DEPOSIT', 'WITHDRAWAL', 'TRANSFER', 'CARD_AUTH'];
    const categories = ['5411', '5812', '5814', '7995', '7832', '7922', ''];
    return {
        idempotency_key: id,
        transaction_type: types[Math.floor(Math.random() * types.length)],
        source_account_id: `acc-${Math.floor(Math.random() * 10000)}`,
        target_account_id: Math.random() > 0.5 ? `acc-${Math.floor(Math.random() * 10000)}` : '',
        amount: Math.round((Math.random() * 100000 + 1) * 100) / 100,
        currency: 'USD',
        fee_amount: Math.round(Math.random() * 50 * 100) / 100,
        merchant_id: Math.random() > 0.7 ? `merch-${Math.floor(Math.random() * 1000)}` : '',
        merchant_category: categories[Math.floor(Math.random() * categories.length)],
    };
}

function generateTransfer() {
    const id = Math.random().toString(36).substring(2, 15) + Date.now().toString(36);
    const src = Math.floor(Math.random() * 10000);
    let tgt = Math.floor(Math.random() * 10000);
    while (tgt === src) {
        tgt = Math.floor(Math.random() * 10000);
    }
    return {
        idempotency_key: id,
        source_account_id: `acc-${src}`,
        target_account_id: `acc-${tgt}`,
        amount: Math.round((Math.random() * 50000 + 1) * 100) / 100,
        currency: 'USD',
    };
}

export default function () {
    const params = {
        headers: { 'Content-Type': 'application/json' },
    };

    // 80% transactions, 20% transfers
    const isTransfer = Math.random() < 0.2;
    const payload = isTransfer ? generateTransfer() : generateTransaction();
    const endpoint = isTransfer ? '/api/v1/transfers' : '/api/v1/transactions';

    const start = Date.now();
    const res = http.post(`${BASE_URL}${endpoint}`, JSON.stringify(payload), params);
    const duration = Date.now() - start;

    txDuration.add(duration);

    const success = check(res, {
        'status is 2xx': (r) => r.status >= 200 && r.status < 300,
        'response time < 500ms': (r) => r.timings.duration < 500,
    });

    if (!success) {
        errorRate.add(1);
    } else {
        txCounter.add(1);
    }

    sleep(Math.random() * 0.01 + 0.001);
}

export function handleSummary(data) {
    const summary = {
        timestamp: new Date().toISOString(),
        scenario: SCENARIO,
        metrics: {
            http_reqs: data.metrics.http_reqs?.values?.count || 0,
            http_req_duration_p95: data.metrics.http_req_duration?.values?.['p(95)'] || 0,
            http_req_duration_p99: data.metrics.http_req_duration?.values?.['p(99)'] || 0,
            http_req_duration_avg: data.metrics.http_req_duration?.values?.avg || 0,
            http_req_failed_rate: data.metrics.http_req_failed?.values?.rate || 0,
            iterations: data.metrics.iterations?.values?.count || 0,
            vus_max: data.metrics.vus_max?.values?.value || 0,
        },
        thresholds: {},
    };

    for (const [name, threshold] of Object.entries(data.thresholds || {})) {
        summary.thresholds[name] = {
            ok: threshold.ok,
        };
    }

    return {
        stdout: JSON.stringify(summary, null, 2),
        [`load-test-${SCENARIO}-results.json`]: JSON.stringify(summary, null, 2),
    };
}
