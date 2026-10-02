## v0.20.0 (Fri, 02 Oct 2026 19:16:28 UTC)
- doc: add logo.
- build(deps): bump ctx42/testing to v0.56.0 and testkit to v0.15.0.
- fix(goldkit): stop panicking on nil bodies and reader-less sources.
- fix(goldkit): remove multipart temp files after body assertion.
- refactor(goldkit): add interface assertions and correct comments.
- test(goldkit): align tests with project test conventions.
- fix(goldkit)!: compare every multipart file and reject empty bodies.
- fix(goldkit)!: assert listed request headers and fail on bad requests.
- fix(goldkit)!: convert time values to the timezone in MetaGetTimeIn.
- fix(goldkit)!: add context to golden file errors.
- style(goldkit): fit lines to the 80-column limit.

## v0.19.0 (Sat, 04 Jul 2026 21:38:39 UTC)
- Documentation and base implementation of `goldkit` module.
- test(goldkit): fix flaky connection-refused assertion on IPv6 hosts.
- docs: remove Go Report Card badge from README.

