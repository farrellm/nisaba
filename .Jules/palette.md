## 2026-06-30 - Focus-visible styling on read-only TextFields
**Learning:** MUI readOnly TextFields do not natively apply the .Mui-focused class when receiving keyboard focus, making them invisible to tab navigation even with tabIndex=0. Relying on pseudo-class :focus-within directly targets the inner elements accurately.
**Action:** Use &:focus-within .MuiOutlinedInput-notchedOutline for custom focus styling on any read-only/clickable MUI text fields.

## 2026-06-30 - ARIA labels for nested Switch components
**Learning:** MUI Switch components nested inside MenuItem often lack intrinsic labels for screen readers unless explicitly provided via inputProps, even if a visible ListItemText exists nearby.
**Action:** Use inputProps={{ 'aria-label': 'Action Name' }} or aria-labelledby for Switch components inside MenuItems or other list items to ensure clear accessibility.
