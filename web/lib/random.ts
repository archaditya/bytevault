// Small deterministic PRNG so mock data is stable across server/client renders (avoids hydration mismatches).
export function createSeededRandom(seed: number) {
  let s = seed % 2147483647;
  if (s <= 0) s += 2147483646;
  return function next(): number {
    s = (s * 16807) % 2147483647;
    return (s - 1) / 2147483646;
  };
}

export function pick<T>(rand: () => number, arr: T[]): T {
  return arr[Math.floor(rand() * arr.length)];
}

export function pickWeighted<T>(rand: () => number, entries: [T, number][]): T {
  const total = entries.reduce((sum, [, w]) => sum + w, 0);
  let r = rand() * total;
  for (const [value, weight] of entries) {
    r -= weight;
    if (r <= 0) return value;
  }
  return entries[entries.length - 1][0];
}

export function randomInt(rand: () => number, min: number, max: number): number {
  return Math.floor(rand() * (max - min + 1)) + min;
}

export function randomFloat(rand: () => number, min: number, max: number): number {
  return rand() * (max - min) + min;
}

export function daysAgo(n: number, jitterHours = 0, rand?: () => number): string {
  const d = new Date();
  d.setDate(d.getDate() - n);
  if (jitterHours && rand) {
    d.setHours(d.getHours() - Math.floor(rand() * jitterHours));
  }
  return d.toISOString();
}

export function minutesAgo(n: number): string {
  const d = new Date();
  d.setMinutes(d.getMinutes() - n);
  return d.toISOString();
}
