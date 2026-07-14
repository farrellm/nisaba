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

// removeFromSet returns prev unchanged when item is absent (so React can skip
// the re-render), else a copy with item removed.
export function removeFromSet<T>(prev: Set<T>, item: T): Set<T> {
  if (!prev.has(item)) return prev
  const next = new Set(prev)
  next.delete(item)
  return next
}

// setInSet adds or removes item to match `present`. Idempotent, so it's safe
// to drive from a native <details> `toggle` event (which also fires on
// programmatic open changes) without oscillating.
export function setInSet<T>(prev: Set<T>, item: T, present: boolean): Set<T> {
  return present ? addToSet(prev, item) : removeFromSet(prev, item)
}
