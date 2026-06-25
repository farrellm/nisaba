## 2024-05-18 - React.memo and useCallback optimization
**Learning:** Extracting objects/arrays out of functional components like `const remarkPlugins = [remarkGfm]` prevents unnecessary recreations, preserving prop identity.
**Action:** Always check array and object props passed into third party libraries like `react-markdown` and extract them to module level constants if possible.
## 2024-06-25 - Avoid Date object allocation in sorts
**Learning:** Instantiating `Date` objects inside array `sort` comparators is very slow because it triggers parsing and allocation on every comparison. Since ISO 8601 strings compare lexicographically perfectly, `updatedAt` strings can be compared directly without conversion.
**Action:** Always prefer direct string comparison for ISO 8601 date strings when sorting, especially in hot paths like React useMemo hooks.
