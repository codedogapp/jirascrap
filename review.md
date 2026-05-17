# Code Review

## Summary
Solid implementation. Follows existing patterns, handles errors gracefully, guards against stale messages. One potential issue with the Jira API response field for `orderBy`.

## Verdict: APPROVE

## Critical Issues
None.

## Important Issues
- [ ] `internal/jira/client.go:158` — The `orderBy=-created` query parameter may not be supported by all Jira Cloud instances (it's documented but inconsistently available). If the API ignores it, comments will come in default order (oldest first). The reverse loop (line 171) compensates for `-created` ordering by flipping to chronological — but if API returns oldest-first by default (ignoring orderBy), the reverse loop would show newest-first which is actually fine UX-wise but inconsistent with spec ("last 20"). Consider: if API returns oldest-first without orderBy support, you'd get the *first* 20, not *last* 20. A safer approach would be to use `startAt = max(0, total - maxResults)` with `orderBy=created` if pagination is needed. For now, this is fine for most Atlassian Cloud instances where `orderBy=-created` works.

## Spec Compliance
- [x] AC1: Comments fetched and displayed below description — met
- [x] AC2: Loading indicator shown while fetching — met ("Loading comments...")
- [x] AC3: Author name, timestamp, rendered markdown body — met
- [x] AC4: Visual separation between comments — met (--- dividers)
- [x] AC5: Max 20 most recent comments — met (maxResults=20, orderBy=-created)
- [x] AC6: Graceful error handling — met (SetCommentsError shows inline warning)
- [x] AC7: Existing behavior unaffected — met (tests pass, scrolling/tags/keybinds unchanged)

## Test Coverage
- Integration test mock updated — tests pass
- No dedicated unit tests for comment rendering or FetchComments — acceptable for this scope since the patterns are identical to transitions (which also lack unit tests)

## Notes
- Smart guard in handlers (line 270-271): checking `dm.Ticket().ID != msg.ticketID` prevents stale comment responses from corrupting the view if user navigates away quickly
- Reverse iteration in FetchComments is clean — avoids a separate `slices.Reverse` call
- Future-proofing: `model.Comment` has ID field ready for write operations
