Reference bundles copied verbatim from `GoogleCloudPlatform/knowledge-catalog`
(`okf/bundles/`, Apache 2.0), minus the generated `viz.html` files.

Vendored at commit `62432a095456147ee71e70ac6e4dc0d2dea3ac30`, which is where §5
started requiring an explicit UTC offset on every timestamp. Upstream ships no tags, so the sha is
the only baseline a refresh can diff against. Record the new one here whenever these are
re-copied.

They pin conformance to something outside our own reading of the spec. Whatever `okf check`
rejects here is a bug in the checker, not in the bundle.
