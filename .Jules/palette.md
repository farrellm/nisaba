## 2024-06-24 - Tooltips for icon-only action menus
**Learning:** In MUI based apps, `IconButton` components with `aria-label`s are accessible to screen readers, but sighted mouse users still need `Tooltip` wrappers to understand generic icons like `MoreVertIcon` (kebab menu).
**Action:** Consistently wrap standalone `IconButton`s in `<Tooltip title="...">` to provide visible text hints.
