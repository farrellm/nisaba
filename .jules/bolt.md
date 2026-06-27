## 2024-05-18 - React.memo and useCallback optimization
**Learning:** Extracting objects/arrays out of functional components like `const remarkPlugins = [remarkGfm]` prevents unnecessary recreations, preserving prop identity.
**Action:** Always check array and object props passed into third party libraries like `react-markdown` and extract them to module level constants if possible.
## 2024-11-20 - Lexicographical sort for ISO 8601 string dates
**Learning:** Instantiating `new Date(dateString)` iteratively during a `.sort()` comparator is an expensive operation and negatively impacts performance, especially for longer lists. ISO 8601 strings naturally sort correctly using lexicographical operators (`<`, `>`).
**Action:** When sorting arrays based on ISO 8601 date fields, avoid `new Date().getTime()` and utilize string comparison (e.g., `a.date < b.date ? -1 : (a.date > b.date ? 1 : 0)`) to optimize list sorting.
## 2026-06-27 - String sorting optimization
**Learning:** String.prototype.localeCompare initializes a new collator on every invocation when options are passed, which scales poorly in array sorting. Using a pre-initialized Intl.Collator is significantly faster.
**Action:** Use a pre-initialized Intl.Collator module constant for repeated string comparisons, especially when sorting lists based on strings.
