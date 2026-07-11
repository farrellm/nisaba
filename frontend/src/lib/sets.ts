// Immutable Set helpers for React state updaters (setX((prev) => ...)).

// toggleSet returns a copy of prev with item removed if present, added if not.
export function toggleSet<T>(prev: Set<T>, item: T): Set<T> {
  const next = new Set(prev)
  if (next.has(item)) next.delete(item)
  else next.add(item)
  return next
}

// addToSet returns prev unchanged when item is already present (so React can
// skip the re-render), else a copy with item added.
export function addToSet<T>(prev: Set<T>, item: T): Set<T> {
  return prev.has(item) ? prev : new Set(prev).add(item)
}
