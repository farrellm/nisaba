## 2024-06-29 - [Cross-Site Request Forgery Prevention]
**Vulnerability:** External links created with `target="_blank"` and `rel="noopener"` without `noreferrer`.
**Learning:** `rel="noopener noreferrer"` should be used whenever `target="_blank"` is applied to external links to prevent the target page from accessing the `window.opener` object and leaking the Referer header to unintended locations. While modern browsers patch the opener vulnerability using just `noopener`, using `noreferrer` is a best-practice layered defense mechanism.
**Prevention:** Always pair `target="_blank"` with `rel="noopener noreferrer"` for external links to enhance privacy and security.
