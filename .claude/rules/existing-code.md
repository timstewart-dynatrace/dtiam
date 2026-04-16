# Understanding Existing Code

---

## 1. Before Making Changes [MUST]

1. **Read the tests first**
   - Tests show intended behavior
   - Tests reveal edge cases the author anticipated
   - Tests document how the code is used

2. **Trace the data flow**
   - Follow the primary use case end-to-end
   - Understand how data moves: CLI args -> command -> handler -> API -> printer
   - Map dependencies between packages

3. **Identify existing patterns**
   - How does this codebase handle the same type of task?
   - What conventions exist (naming, error handling, output)?
   - What is considered an anti-pattern here?

**MUST NOT** make changes based on assumptions from file names or partial reads. Read the full file before modifying it.

## 2. Code Review Checklist [SHOULD]

When reviewing existing code before changes:

- [ ] Can I understand what this does after reading it once?
- [ ] Is the naming clear and consistent with the rest of the codebase?
- [ ] Are edge cases handled?
- [ ] Is there test coverage for the behavior I'm changing?
- [ ] Could this be simpler without losing correctness?
- [ ] Does it follow project patterns (BaseHandler, Printer, etc.)?

## 3. Making Changes Safely [MUST]

1. **Write a test first** that captures current behavior
2. **Make the smallest change** that achieves the goal
3. **Run all tests** to ensure nothing broke: `make test`
4. **Run the linter** to catch regressions: `make lint`
5. **Document the change** in code comments and CHANGELOG

Large rewrites require explicit user approval before starting. If a "small fix" grows into a larger refactor mid-implementation, stop and ask.

## 4. Refactoring Safely [MUST]

1. **Separate refactoring from features** — different commits, different branches
2. **Preserve behavior** — refactoring must not change what code does
3. **Test thoroughly** — all existing tests must pass without modification
4. **Communicate intent** — commit message explains *why* the refactoring

**MUST NOT** refactor and add features in the same change. Mixed changes are impossible to review or revert cleanly.

## 5. When Code is Confusing [SHOULD]

- Check git blame for context: `git blame <file>`
- Look at related issues/PRs for discussion
- Do NOT delete "mysterious" code without understanding why it exists
- Code that looks unused may be called via reflection, dynamic dispatch, or external tooling — verify before removing

## 6. Understanding Before Deleting [MUST]

**MUST NOT** delete code without understanding why it exists.

If you want to delete code:
1. Confirm it is unreachable (grep for references, check for dynamic calls)
2. If unsure, comment it out with a note explaining why before removing
3. Verify tests still pass after removal
