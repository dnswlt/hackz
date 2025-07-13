import http from 'k6/http';
import { check, sleep } from 'k6';

// Read hostname from environment variable or default to 'localhost'
const HOSTNAME = __ENV.HOSTNAME || 'localhost';
const VU_COUNT = parseInt(__ENV.VU_COUNT) || 100;
const BASE_URL = `https://${HOSTNAME}:8443`;

// Number of items to preload and query.
const ITEMS_COUNT = 1024;

export const options = {
    stages: [
        { duration: '5s', target: VU_COUNT },  // Ramp up
        { duration: '50s', target: VU_COUNT },  // Full load
        { duration: '5s', target: 0 },   // Ramp down
    ],
    // InsecureSkipTLSVerify: true is needed for self-signed certificates on localhost.
    // This allows k6 to skip certificate validation, essential for self-signed certs.
    // DO NOT USE IN PRODUCTION WITHOUT UNDERSTANDING THE IMPLICATIONS!
    insecureSkipTLSVerify: true,

    // Thresholds: Define pass/fail criteria for your test
    thresholds: {
        'http_req_duration': ['p(95)<500'], // 95% of requests must complete within 500ms
        'http_req_failed': ['rate<0.01'],    // Less than 1% of requests should fail
    },
};

export function setup() {
    // Make sure the server knows about the items we're GET'ting.
    const itemIDs = [];

    for (let i = 0; i < ITEMS_COUNT; i++) {
        const id = `item${i}`;
        const payload = JSON.stringify({ id: id, name: `Preloaded ${id}` });

        const res = http.post(`${BASE_URL}/rpz/items`, payload, {
            headers: { 'Content-Type': 'application/json' },
        });

        check(res, {
            [`POST ${id} succeeded`]: (r) => r.status >= 200 && r.status <= 299,
        });

        itemIDs.push(id);
    }

    return { itemIDs };
}

export default function () {
    const id = `item${Math.floor(Math.random() * ITEMS_COUNT)}`;
    const res = http.get(`${BASE_URL}/rpz/items/${id}`);

    check(res, {
        'GET 200': (r) => r.status === 200,
    });
}
