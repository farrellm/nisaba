
## $(date +%Y-%m-%d) - Fix SSRF via Double URL Encoding Bypass
**Vulnerability:** Path traversal (SSRF) bypass in `redditPostPath` where `net/url` does not unescape `%2f` (slash) during parsing, allowing attackers to use `%2f` or `%252f` to hide `.` or `..` segments from `strings.Split(parsed.Path, "/")` validation.
**Learning:** `url.Parse` preserves `%2f` in `parsed.Path` and `parsed.EscapedPath()`. Attackers can abuse double URL encoding to slip dot-segments past basic `strings.Split` checks before forwarding to a downstream API that decodes them. Using standard `url.PathUnescape` on user input directly can crash if the string contains a malformed or naturally occurring `%` sign.
**Prevention:** Iteratively decode only the specific URL encoding characters relevant to path separation and traversal (`%2f`, `%2e`, `%25`) before splitting and validating path segments to ensure robust defense against double encoding evasion.
