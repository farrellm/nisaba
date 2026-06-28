## 2026-06-26 - Added loading spinners to async dialogs
**Learning:** Found that `submitting` states exist on async dialogs (`NewDocumentDialog`, `AddBlockDialog`) but lacked visual indicators besides text changes.
**Action:** Consistently apply `CircularProgress` within submit buttons during async actions for better visual feedback.
## 2023-10-27 - Keyboard Navigation for Custom Interactive Elements
**Learning:** Found custom interactive elements (e.g., `Box` mimicking a button on the RedditPostsPage, `TextField` acting as an expander in BlockCard) that lacked proper keyboard support. While they had `onClick` handlers, they couldn't be triggered via keyboard (Enter/Space) and were missing from the tab order.
**Action:** When using non-native interactive elements (like `Box`) or repurposing inputs (like a read-only `TextField` for a collapsible area), explicitly add `tabIndex={0}`, an `onKeyDown` handler that listens for 'Enter' and ' ', and `role="button"` (if semantically a button). Also ensure focus visibility with CSS styles.
