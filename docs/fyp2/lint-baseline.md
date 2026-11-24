# CI lint policy and inherited backlog

The original long workflow invoked golangci-lint v1 against a v2 configuration,
so lint never reached the source analysis. The workflow now pins v2.2.1, matching
the version already specified by the repository's older build workflow.

Once the tool actually ran, the whole source reported more than 500 findings
(the default output caps each linter at 50). They include inherited naming,
line-length, test-package, error-handling and complexity findings. This is not
a lint-clean baseline.

The delivery gate keeps all configured linters enabled and rejects findings
introduced since the supplied revision:
`692f63aff4932a54ef782d2c8feaaa2457ba4247`.

All 37 findings initially identified in the new changes were fixed. Running the
same configured linters with `--new-from-rev` against that revision then reported
zero new issues locally under Go 1.24.2. Checkout uses full history so the
comparison is available on CI. This policy does not suppress or resolve the
inherited backlog. A clean gate must be described as **no new lint findings**, not
as **the whole repository is lint-clean**.

To inspect the full backlog, run `golangci-lint run` without the baseline flag.
To reproduce the delivery gate, run:

```sh
golangci-lint run \
  --new-from-rev=692f63aff4932a54ef782d2c8feaaa2457ba4247 \
  --max-issues-per-linter=0 --max-same-issues=0
```

The default test job still runs the full `go test ./...` set. It now uses Xvfb,
as the Ebitengine packages initialize a display even when tests do not open a
visible window. No test filters were added to make the default test job pass.

The long integration jobs also preserve their test process exit codes. Results
are uploaded even after failure and the capability report still runs, but a
successful artifact upload can no longer hide a failed parent test suite.
