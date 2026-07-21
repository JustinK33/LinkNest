import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  vus: Number(__ENV.VUS || 50),
  duration: __ENV.DURATION || "30s",
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<100"],
  },
};

export default function () {
  const linkID = __ENV.LINK_ID || "1";
  const key = `${__VU}-${__ITER}`;
  const res = http.post(`${__ENV.BASE_URL || "http://localhost:8081"}/links/${linkID}/track_click`, null, {
    headers: {
      "Idempotency-Key": key,
      "User-Agent": "k6-linknest-load-test",
      "X-Forwarded-For": `10.0.${__VU % 255}.${__ITER % 255}`,
    },
  });
  check(res, {
    "accepted": (r) => r.status === 202 || r.status === 404,
  });
  sleep(0.1);
}
