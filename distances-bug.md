## Background
We updated the behavior of the query function in cyborgdb-core and cyborgdb-service so that if a user calls the Query() function without an include argument, it returns only ids by default. The previous behavior was that if the user didn't provide an include argument, Query would return ids, distances, and metadata. Currently, the tests in go don't seem to check which properties are actually returned from query.

We already made a fix for this on the cyborgdb-js and cyborgdb-py repos and added an additional test on the contract tests to specifically check for this.

## Goal
- Update tests that call Query to make sure query returns the correct information
- Add the same contract test that we added to cyborgdb-js
- Ensure that the test suites match the test suites in cyborgdb-js and cyborgdb-py

## Related repos in my repos folder
- cyborgdb-core
- cyborgdb-service
- cyborgdb-js
- cyborgdb-py

## Related PRs
https://github.com/cyborginc/cyborgdb-py/pull/67
https://github.com/cyborginc/cyborgdb-js/pull/67