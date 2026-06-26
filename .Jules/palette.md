## 2026-06-26 - Added loading spinners to async dialogs
**Learning:** Found that `submitting` states exist on async dialogs (`NewDocumentDialog`, `AddBlockDialog`) but lacked visual indicators besides text changes.
**Action:** Consistently apply `CircularProgress` within submit buttons during async actions for better visual feedback.
