/**
 * Recursively extract all leaf field paths from a JSON object using dot notation.
 * e.g. {"data": {"user": {"name": "x", "tags": [1,2]}}}
 * => ["data.user.name", "data.user.tags.0", "data.user.tags.1"]
 *
 * For arrays, extracts paths from the first element as a representative sample.
 */
export function extractPaths(obj: unknown, prefix = ''): string[] {
  const paths: string[] = [];

  if (obj === null || obj === undefined || typeof obj !== 'object') {
    // Leaf value — if we have a prefix, it's a valid path
    if (prefix) paths.push(prefix);
    return paths;
  }

  if (Array.isArray(obj)) {
    if (obj.length === 0) {
      if (prefix) paths.push(prefix);
      return paths;
    }
    // Use first element as sample for nested structure
    const first = obj[0];
    if (first !== null && typeof first === 'object') {
      paths.push(...extractPaths(first, prefix ? `${prefix}.0` : '0'));
    } else {
      if (prefix) paths.push(prefix);
    }
    return paths;
  }

  // Plain object
  for (const [key, value] of Object.entries(obj)) {
    const path = prefix ? `${prefix}.${key}` : key;
    if (value !== null && typeof value === 'object') {
      paths.push(...extractPaths(value, path));
    } else {
      paths.push(path);
    }
  }

  return paths;
}

/**
 * Compute similarity between two field names for auto-matching.
 * Returns a score from 0 to 1.
 */
function similarity(a: string, b: string): number {
  // Get the last segment (the actual field name)
  const nameA = a.split('.').pop()!.toLowerCase();
  const nameB = b.split('.').pop()!.toLowerCase();

  // Exact match on field name
  if (nameA === nameB) return 1;

  // Normalize: remove underscores, hyphens, convert to lowercase
  const normA = nameA.replace(/[-_]/g, '');
  const normB = nameB.replace(/[-_]/g, '');
  if (normA === normB) return 0.9;

  // One contains the other
  if (normA.includes(normB) || normB.includes(normA)) return 0.7;

  // Full path exact match
  const fullA = a.toLowerCase().replace(/[-_]/g, '');
  const fullB = b.toLowerCase().replace(/[-_]/g, '');
  if (fullA === fullB) return 0.85;

  return 0;
}

/**
 * Auto-match source fields to target fields based on name similarity.
 * Returns an array of {source, target} pairs for matches above threshold.
 * Each field is used at most once.
 */
export function autoMatch(
  sourcePaths: string[],
  targetPaths: string[],
  threshold = 0.6,
): { source: string; target: string; score: number }[] {
  // Build all pairs with scores
  const pairs: { source: string; target: string; score: number }[] = [];
  for (const s of sourcePaths) {
    for (const t of targetPaths) {
      const score = similarity(s, t);
      if (score >= threshold) {
        pairs.push({ source: s, target: t, score });
      }
    }
  }

  // Greedy match: highest score first, each field used once
  pairs.sort((a, b) => b.score - a.score);
  const usedSource = new Set<string>();
  const usedTarget = new Set<string>();
  const result: typeof pairs = [];

  for (const p of pairs) {
    if (usedSource.has(p.source) || usedTarget.has(p.target)) continue;
    usedSource.add(p.source);
    usedTarget.add(p.target);
    result.push(p);
  }

  return result;
}
