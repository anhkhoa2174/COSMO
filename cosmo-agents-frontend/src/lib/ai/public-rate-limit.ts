/**
 * Rate limiting for the public Ask COSMO endpoint.
 *
 * Every other AI route in this project is behind a session cookie, because a
 * call costs money and an unauthenticated one is a bill anybody on the internet
 * can run up. The landing-page assistant deliberately has no cookie, so the
 * limit is the only thing standing between the product and that bill.
 *
 * Two windows, because they stop different things. The short one stops a person
 * holding down enter; the long one stops a script that paces itself to slip
 * under the short one.
 *
 * Known limitation: the counters live in this process's memory. A deploy resets
 * them, and with more than one instance each keeps its own tally, so the real
 * ceiling is the limit multiplied by the instance count. That is acceptable for
 * a bounded-cost assistant with a small token budget and no tools, and it needs
 * no infrastructure this project does not already run. A shared counter (Redis)
 * is the upgrade if the endpoint ever gets an expensive capability.
 */

export interface RateLimitRule {
  /** Window length in milliseconds. */
  windowMs: number;
  /** Requests allowed inside one window. */
  max: number;
  label: string;
}

export const ASK_COSMO_RULES: RateLimitRule[] = [
  { windowMs: 60_000, max: 6, label: 'minute' },
  { windowMs: 60 * 60_000, max: 40, label: 'hour' },
];

/** Timestamps of recent requests, newest last, keyed by client. */
const hits = new Map<string, number[]>();

/**
 * Cap on distinct keys tracked at once.
 *
 * Without it the map is itself an attack: a request per forged address grows it
 * without bound. When the cap is hit the coldest half is dropped, which loses
 * some history but keeps memory flat — and losing history only ever makes the
 * limiter more permissive to callers who had already gone quiet.
 */
const MAX_TRACKED_KEYS = 10_000;

export interface RateLimitResult {
  ok: boolean;
  /** Seconds the caller should wait, set when `ok` is false. */
  retryAfter?: number;
  rule?: string;
}

function prune(list: number[], now: number, longestWindow: number): number[] {
  const cutoff = now - longestWindow;
  // The list is chronological, so the survivors are a suffix of it.
  let i = 0;
  while (i < list.length && list[i] <= cutoff) i++;
  return i === 0 ? list : list.slice(i);
}

function evictIfCrowded() {
  if (hits.size <= MAX_TRACKED_KEYS) return;
  const keys = [...hits.keys()];
  for (const key of keys.slice(0, Math.floor(keys.length / 2))) {
    hits.delete(key);
  }
}

/**
 * Records a request and reports whether it is allowed.
 *
 * A refused request is *not* recorded. Counting refusals would let a caller who
 * is already over the limit keep their own window rolling forever, which turns
 * a temporary block into a permanent one.
 */
export function checkRateLimit(
  key: string,
  rules: RateLimitRule[] = ASK_COSMO_RULES
): RateLimitResult {
  const now = Date.now();
  const longest = Math.max(...rules.map((r) => r.windowMs));

  const recent = prune(hits.get(key) ?? [], now, longest);

  for (const rule of rules) {
    const since = now - rule.windowMs;
    const inWindow = recent.filter((t) => t > since);
    if (inWindow.length >= rule.max) {
      hits.set(key, recent);
      const oldest = inWindow[0];
      return {
        ok: false,
        retryAfter: Math.max(
          1,
          Math.ceil((oldest + rule.windowMs - now) / 1000)
        ),
        rule: rule.label,
      };
    }
  }

  recent.push(now);
  hits.set(key, recent);
  evictIfCrowded();
  return { ok: true };
}

/**
 * Identifies the caller.
 *
 * `x-forwarded-for` is a client-supplied header and trivially spoofed, so this
 * is a cost brake rather than an access control: it stops accidental and casual
 * abuse, not a determined attacker. The leftmost entry is used because proxies
 * append, so the original client sits at the front. Requests with no usable
 * address share one bucket, which is the conservative choice — they throttle
 * each other rather than each getting a fresh allowance.
 */
export function clientKey(headers: Headers): string {
  const forwarded = headers.get('x-forwarded-for');
  if (forwarded) {
    const first = forwarded.split(',')[0]?.trim();
    if (first) return first;
  }
  return headers.get('x-real-ip')?.trim() || 'unknown';
}

/** Exposed for tests: forget every tracked caller. */
export function resetRateLimits() {
  hits.clear();
}
