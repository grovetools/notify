## v0.6.0 (2026-02-02)

This release focuses on configuration migration and documentation structure improvements. The configuration format has been migrated from YAML to TOML (3e8d2dd), accompanied by the addition of the MIT License (4ec5384) and updates to the configuration documentation (88e309a).

Documentation workflows have been streamlined by moving docgen configuration and templates to the notebook environment (1a71334, 2472e91) and reorganizing context rules (6c206ef). Additionally, the overview and concept lookup instructions have been updated (a060f54, fbc0290).

On the build and maintenance side, the release workflow has been restored (2d701a5), module dependencies have been updated to the grovetools namespace (0856551), and version injection during builds has been fixed (8f1a4bf).

### Features
* feat: add configuration/readme update (88e309a)
* feat: update readme/overview (a060f54)

### Bug Fixes
* fix: update VERSION_PKG to grovetools/core path (8f1a4bf)

### Documentation
* docs: add concept lookup instructions to CLAUDE.md (fbc0290)

### Chores and Refactoring
* ci: restore release workflow (2d701a5)
* Add MIT License (4ec5384)
* chore: update docs.json (95469df)
* chore: migrate grove.yml to grove.toml (3e8d2dd)
* chore: move README template to notebook (2472e91)
* chore: remove docgen files from repo (1a71334)
* chore: move docs.rules to .cx/ directory (6c206ef)
* refactor: update docgen title to match package name (d65f535)
* chore: update go.mod for grovetools migration (0856551)

### File Changes
```
 .cx/docs.rules                | 13 ++++++++++
 .github/workflows/release.yml | 58 +++++++------------------------------------
 CLAUDE.md                     | 15 ++++++++++-
 LICENSE                       | 21 ++++++++++++++++
 Makefile                      |  2 +-
 README.md                     | 54 +++++++++++++++++++++-------------------
 docs/01-overview.md           | 52 ++++++++++++++++++++------------------
 docs/04-configuration.md      | 35 ++++++++++++++++++++++++++
 docs/README.md.tpl            |  6 -----
 docs/docgen.config.yml        | 27 --------------------
 docs/docs.rules               |  1 -
 go.mod                        | 11 ++++++--
 go.sum                        | 49 +++++++++++++++++++++++++++++++++---
 grove.toml                    | 10 ++++++++
 grove.yml                     |  9 -------
 notifications.schema.json     |  2 +-
 pkg/docs/docs.json            | 33 +++++++++++++++++-------
 17 files changed, 239 insertions(+), 159 deletions(-)
```

## v0.1.1-nightly.4d6373c (2025-10-03)

## v0.1.0 (2025-10-01)

The documentation system has been significantly enhanced with the introduction of automatic Table of Contents (TOC) generation and updates to the `docgen` configuration (f380cf1, a827ed7). The configuration has been standardized for better maintainability (3d6d871), and the generated documentation has been made more succinct (3d7c6e2).

The CI/CD process has been refined. The release workflow now extracts release notes directly from `CHANGELOG.md` to ensure consistency between the repository and GitHub releases (323d008). Redundant tests have been removed from the release workflow to streamline the process (6767b86), and a syntax issue in the CI trigger has been corrected to use `branches: [ none ]` for clearer intent (0e46882).

### Features

- make docs succinct, edit docs.rules, add stripines (3d7c6e2)
- add TOC generation and docgen configuration updates (f380cf1)
- update release workflow to use CHANGELOG.md (323d008)

### Bug Fixes

- update CI workflow to use none branches instead of commenting (0e46882)

### Documentation

- update docgen configuration and README templates (a827ed7)

### Code Refactoring

- standardize docgen.config.yml key order and settings (3d6d871)

### Chores

- temporarily disable CI workflow (7b70b44)
- update .gitignore rules (6f8f4a0)

### Continuous Integration

- remove redundant tests from release workflow (6767b86)

### File Changes

```
 .github/workflows/ci.yml         |  4 +--
 .github/workflows/release.yml    | 27 ++++++++-----------
 .gitignore                       |  3 +++
 CLAUDE.md                        | 30 +++++++++++++++++++++
 README.md                        | 49 +++++++++++++++++++++++++++++++++
 docs/01-overview.md              | 37 +++++++++++++++++++++++++
 docs/02-configuration.md         | 58 ++++++++++++++++++++++++++++++++++++++++
 docs/README.md.tpl               |  6 +++++
 docs/docgen.config.yml           | 30 +++++++++++++++++++++
 docs/docs.rules                  |  1 +
 docs/prompts/01-overview.md      | 31 +++++++++++++++++++++
 docs/prompts/02-configuration.md | 23 ++++++++++++++++
 pkg/docs/docs.json               | 27 +++++++++++++++++++
 13 files changed, 308 insertions(+), 18 deletions(-)
```

## v0.0.11 (2025-09-17)

### Chores

* bump dependencies

## v0.0.10 (2025-09-13)

### Chores

* update Grove dependencies to latest versions

## v0.0.9 (2025-08-28)

### Chores

* **deps:** sync Grove dependencies to latest versions
* add Grove ecosystem files

## v0.0.8 (2025-08-27)

### Bug Fixes

* add version cmd

## v0.0.7 (2025-08-25)

### Continuous Integration

* add Git LFS disable to release workflow
* disable Git LFS and linting in workflow

## v0.0.6 (2025-08-25)

### Chores

* **deps:** bump dependencies
* bump dependencies

## v0.0.5 (2025-08-15)

### Chores

* **deps:** bump dependencies
* bump deps

### Continuous Integration

* switch to Linux runners to reduce costs
* consolidate to single test job on macOS
* reduce test matrix to macOS with Go 1.24.4 only

### Bug Fixes

* remove fmt
* disable ci for now

### Code Refactoring

* standardize E2E binary naming and use grove.yml for binary discovery

## v0.0.4 (2025-08-12)

### Bug Fixes

* disable ci for now

## v0.0.3 (2025-08-12)

### Bug Fixes

* install libnotify in ci environment

## v0.0.2 (2025-08-12)

### Bug Fixes

* makefile

