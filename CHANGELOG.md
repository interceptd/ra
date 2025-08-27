# Changelog

## [Unreleased] - Security Hardening

### Added
- Initial CHANGELOG for security audit and fixes

### Changed
- Directory and file creation sanitizes user input and prevents directory traversal
- Shell command execution hardened against injection and user input interpolation
- Markdown rendering restricted to avoid XSS, only trusted markdown is rendered with `unsafe_allow_html=true`
- Sensitive data (PAT tokens) are never logged or shown in the UI
- General code paths reviewed for similar security gaps

### Fixed
- Potential directory traversal in use case creation
- Possible shell injection from commands.md or user input
- XSS risk when rendering markdown reports
- Exposure of Azure DevOps Personal Access Token (PAT) in logs or UI

---

This release prepares the repo for a more robust, secure handling of user input and file/command operations.