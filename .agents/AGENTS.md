# KubeEdge Custom Rules

## PR Best Practices (per maintainer DoisLONG)

1. **Consolidate Small Coverage-Only PRs:** Always group similar, small test files (like fakers, simple helpers, or pure formatters) into a single, comprehensive PR rather than opening multiple separate PRs. This reduces review overhead for the community.

2. **Negative Test Cases (Error Verification):** For negative test cases where an error is expected (`wantErr == true`), the test must immediately `return` after verifying the error state. Do not perform `reflect.DeepEqual` or assert on the returned data variables, as this makes negative cases brittle by depending on exact zero-value outputs.

3. **Prow Bot Command Awareness:** KubeEdge heavily relies on Prow/Tide. Always proactively remind maintainers to drop the explicit `/lgtm` and `/approve` commands if they provide a verbal approval but forget the tags.
