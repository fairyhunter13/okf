# Defect

* [A trailing slash on the bundle root invented missing
  resources](a-trailing-slash-invented-missing-resources.md) - anyExists reads filepath.Dir(Root)
  as the repo root. So a root spelled knowledge/ resolved every repo-relative resource against the
  bundle itself. That gave 12 conformance errors on one real bundle with the slash, and 0 without.
