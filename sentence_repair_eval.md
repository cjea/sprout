# Sentence Repair Evaluation

The sentence-repair pass is optional and should stay off by default.

## When To Enable It

Enable repair only when all of the following are true:

- heuristic segmentation produced at least one suspicious fragment cluster
- a model is available for the request path
- the caller is willing to trade extra latency and cost for better passage quality

In practice, this means enabling repair for cases like:

- `Held ... Pp. 3-10.`
- `Id. , at 484.`
- a trailing statute or case citation split away from the proposition it supports

## When To Leave It Off

Leave repair off when:

- the heuristic output already matches the fixture-backed quality corpus
- there are no suspicious fragment clusters
- deterministic local behavior matters more than marginal cleanup
- the environment does not have a configured model path

## Current Evaluation Shape

The evaluation set mixes:

- real Supreme Court fixture cases derived from `24-777_9ol1.pdf`
- focused legal edge cases for `Id.` and trailing authority cites

The comparison checks:

- heuristic-only output
- heuristic-plus-optional-repair output
- whether repair fixes broken fragment clusters
- whether repair leaves clean sentence boundaries unchanged

Current expectation:

- repair should improve only the known broken clusters
- repair should not touch clean segmentations
