# mysh

## Language Policy

English is the primary language for this project. All user-facing text, documentation, commit messages, PR descriptions, and code comments must be written in English first.

Japanese is supported as a second language via the i18n system (`internal/i18n/i18n.go`). When adding user-facing strings:

1. Add the English string to the `en` map (primary)
2. Add the Japanese translation to the `ja` map (secondary)

Documentation files should be in English. Japanese translations are optional and use the `.ja.md` suffix (e.g., `docs/import-guide.ja.md`).
