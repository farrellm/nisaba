## 2024-05-18 - React.memo and useCallback optimization
**Learning:** Extracting objects/arrays out of functional components like `const remarkPlugins = [remarkGfm]` prevents unnecessary recreations, preserving prop identity.
**Action:** Always check array and object props passed into third party libraries like `react-markdown` and extract them to module level constants if possible.
## 2024-11-20 - Lexicographical sort for ISO 8601 string dates
**Learning:** Instantiating `new Date(dateString)` iteratively during a `.sort()` comparator is an expensive operation and negatively impacts performance, especially for longer lists. ISO 8601 strings naturally sort correctly using lexicographical operators (`<`, `>`).
**Action:** When sorting arrays based on ISO 8601 date fields, avoid `new Date().getTime()` and utilize string comparison (e.g., `a.date < b.date ? -1 : (a.date > b.date ? 1 : 0)`) to optimize list sorting.
## 2026-06-27 - String sorting optimization
**Learning:** String.prototype.localeCompare initializes a new collator on every invocation when options are passed, which scales poorly in array sorting. Using a pre-initialized Intl.Collator is significantly faster.
**Action:** Use a pre-initialized Intl.Collator module constant for repeated string comparisons, especially when sorting lists based on strings.
## 2026-07-01 - Prevent redundant allocations in nested loops
**Learning:** React re-renders with heavy list-processing operations like string transformations (`.toLowerCase()`) or `Intl.Collator` initializations can create hidden memory/CPU spikes, especially if inside `useMemo` hooks mapping arrays against other arrays.
**Action:** Always pre-compute primitive transformations outside of inner maps/filters (like moving `.toLowerCase()` to a Set outside the loop) to change O(n*m) allocations to O(n) + O(m) lookups.
## 2024-11-21 - useMemo dependency tracking with object fallbacks
**Learning:** If a variable is initialized with a fallback object like `const attributes = doc.attributes ?? {}`, using `attributes` in a `useMemo` dependency array (`[attributes]`) will trigger continuous re-renders whenever `doc.attributes` is undefined, because a new empty object reference is created on every pass.
**Action:** When memoizing derived state based on potentially undefined props, use the original prop (e.g., `doc.attributes`) as the `useMemo` dependency instead of an intermediate variable that creates a new reference.

## 2024-07-03 - [Array Allocation in Hot Loops]
**Learning:** React render loops that perform filtering on large lists (like DocumentList.tsx) were allocating new arrays via `.map` for every document processed, generating significant garbage. Furthermore, omitting an early return when no filters are active forced an O(N) pass for every render.
**Action:** When filtering lists in React, check if the filter criteria are empty first to bypass the `.filter()` entirely. Replace `.map().includes()` with `.some()` to avoid generating temporary arrays during the iteration.
