## 2024-05-18 - React.memo and useCallback optimization
**Learning:** Extracting objects/arrays out of functional components like `const remarkPlugins = [remarkGfm]` prevents unnecessary recreations, preserving prop identity.
**Action:** Always check array and object props passed into third party libraries like `react-markdown` and extract them to module level constants if possible.
