## 2026-06-30 - Focus-visible styling on read-only TextFields
**Learning:** MUI readOnly TextFields do not natively apply the .Mui-focused class when receiving keyboard focus, making them invisible to tab navigation even with tabIndex=0. Relying on pseudo-class :focus-within directly targets the inner elements accurately.
**Action:** Use &:focus-within .MuiOutlinedInput-notchedOutline for custom focus styling on any read-only/clickable MUI text fields.
