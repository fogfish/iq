# AGENTS Instructions

## Philosophy

1. **New Feature Branch** - Always create a new branch for implementation of each issue, feature, fix, etc.
2. **Understand before modifying** — read code, tests, trace calls
3. **Verify proportionally** — sensitive code = thorough testing
4. **Ask when uncertain** — ambiguity warrants a question
5. **Flag sensitive changes** — tell the user when touching elevated care zones
6. **Keep diffs small** — easier to review and revert
7. **Never leak secrets**


## Coding Convention and Requirments

* Prefer pure functional style
* Make small and readable functions
* Avoid comments in the code, purpose of code blocks is self explanatory
* Use modernized Golang syntax
* Use `github.com/fogfig/it/v2` to write assertion for unit tests
* Prefer table driven tests when possible
* Keep high up to 90% test coverage for newly added features, do not write test for shake of coverage
* Always add license header to newly created files

## Testing

Always do testing before making pull request:
* `go test -v ./...` unit testing
* `bash ./it.sh` integration testing uses examples to validates app, you can skip for small changes


## Releasing 

* Create pull request to https://github.com/fogfish/iq when feature, fix or changes are completed. 
* Push changes with `git push github ...` 
