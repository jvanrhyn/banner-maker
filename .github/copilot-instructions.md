# Copilot Instructions for this repository

- Branch safety:
  - Always start and work in a feature branch (not `main` or `master`).
  - Do not perform merges into `main` without explicit user confirmation.
  - If `main` must be updated, ask for permission first and wait for approval.

- Push policy:
  - Do not push any commits to remote by default.
  - Only run `git push` when explicitly instructed by the user.
  - Confirm the target remote/branch before pushing.

- Review and skills discovery:
  - Always inspect available skills and tooling for Go and Go TUI frameworks (Bubble Tea, Bubbles, Lip Gloss, Charmbracelet, etc.) before implementation.
  - If additional skills are available, mention them and follow best practices for those frameworks.

- Development approach:
  - Use Test Driven Development (TDD) whenever possible.
  - Prioritize writing or updating tests first, then implement the minimal code to pass tests.
  - Keep tests focused, deterministic, and fast.

- Communication:
  - In each update, include:
    - what branch is in use
    - what files were touched
    - whether tests were added/updated
    - what next actions are needed from the user

