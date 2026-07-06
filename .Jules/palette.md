## 2026-06-30 - Focus-visible styling on read-only TextFields
**Learning:** MUI readOnly TextFields do not natively apply the .Mui-focused class when receiving keyboard focus, making them invisible to tab navigation even with tabIndex=0. Relying on pseudo-class :focus-within directly targets the inner elements accurately.
**Action:** Use &:focus-within .MuiOutlinedInput-notchedOutline for custom focus styling on any read-only/clickable MUI text fields.

## 2026-06-30 - ARIA labels for nested Switch components
**Learning:** MUI Switch components nested inside MenuItem often lack intrinsic labels for screen readers unless explicitly provided via inputProps, even if a visible ListItemText exists nearby.
**Action:** Use inputProps={{ 'aria-label': 'Action Name' }} or aria-labelledby for Switch components inside MenuItems or other list items to ensure clear accessibility.

## 2026-06-30 - Keyboard visibility for hover-revealed controls
**Learning:** Hover-revealed controls (like actions that appear or become fully opaque on container `:hover`) are invisible or fail contrast requirements for keyboard users navigating via Tab unless `:focus-within` is also applied to the container.
**Action:** Whenever using `&:hover` on a container to reveal or highlight child controls, always pair it with `&:focus-within` on the container or `&:focus-visible` on the element itself to ensure full keyboard accessibility.
## 2024-07-04 - Add autoFocus and aria-label to inline editing TextFields
**Learning:** React Material-UI `TextField` components used for inline editing of block responses lack accessibility context when toggled via a button, as screen readers only announce "edit text". Furthermore, the UX is clunky because the user must click again to focus the newly revealed input.
**Action:** Always include `autoFocus` and `inputProps={{ 'aria-label': 'Descriptive label' }}` on inline edit input fields to eliminate the extra click and provide proper screen reader context.
## 2026-07-05 - ARIA labels for readOnly clickable TextFields
**Learning:** MUI `TextField` components used as clickable read-only elements to reveal full content lack accessible names, making their interactive nature unclear to screen reader users. The label prop only serves as visual/semantic context, but does not convey action.
**Action:** Always include an `aria-label` inside `inputProps` for any `TextField` or similar component that functions as a button or toggle, clearly describing the resulting action (e.g., "Expand [key]").
## 2024-07-06 - ARIA labels for nested standard HTML input equivalents in MUI
**Learning:** MUI components that wrap standard inputs, such as `Switch` and `Radio`, or decorative wrappers like `Chip`, often fail to expose their action context to screen readers despite their clear visual function. The label is only visual, not functional metadata for accessibility.
**Action:** When using MUI `Switch`, `Radio`, or interactive `Chip`s, always explicitly provide `inputProps={{ 'aria-label': 'Action description' }}` (or `aria-label` directly on the `Chip`) to define their purpose to screen readers.
