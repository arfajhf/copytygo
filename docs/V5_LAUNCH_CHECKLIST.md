# CopyTyGo v5 public launch readiness

CopyTyGo v5 is the **first planned public stable release**. The prior v4.0.6 tag was used internally before public launch. v5 work now focuses on reliability, security, onboarding, official website/documentation and public distribution.

This is a launch gate, not a statement that v5 is released.

## Must pass before the final v5 tag

- [ ] Confirm Windows Native Runtime on the target Windows 11 machine, including the exact App Control/Defender verdict when a build is denied
- [x] Implement a stable project-local Windows native development executable without suppressing policy failures
- [x] Add actionable Windows diagnostic documentation and regression checks
- [x] Provide optional WSL2 full native development without disabling host Smart App Control; Windows-native trust acceptance remains outstanding
- [x] Migrate the Go module and generated templates to the v5 semantic import path `github.com/arfajhf/copytygo/v5`
- [x] Add CI migration test from a real v4.0.6 multi-role TypeScript app with generated resource, and document major-version upgrade commands
- [x] Classify customized v4 upgrade and production-data rollback as non-blocking for the first public launch, because v4.0.6 has not been distributed to external users (revisit if the audience changes)
- [ ] Audit old HTML auth templates and remove or label compatibility-only code; TypeScript must remain the default source for generated auth UI
- [ ] Audit auth, CSRF, static asset serving, file uploads, SQL identifiers, sessions, authorization and production Studio isolation
- [x] Bound local disk operations to Go's os.Root to reject symlink escape during file read/write/delete, plus regression tests
- [x] Reject non-finite numeric inputs; close multipart FormFile reader after metadata extraction
- [x] Add opt-in content-sniffed MIME allowlists and mandatory size limit for SaveUploadedFileAllowed; malformed generated resource JSON is now rejected with HTTP 400
- [ ] Validate upload route authentication and signature handling end-to-end, including reverse-proxy limits and application-specific scanning
- [ ] Confirm malformed JSON, form inputs, and body limits through on-device Fedora acceptance after CI
- [x] Add private-by-default generated resource routes with role enforcement and CSRF; explicit public access and route resync are supported
- [ ] Confirm generated resource protections through end-to-end browser and bearer-client tests on a fresh v5 project
- [ ] Validate all 4 project starters, both database drivers, native builds and production assets
- [ ] Run the complete CI matrix and Windows smoke checks against the release candidate
- [ ] Update all current architecture, authentication, runtime, production and upgrade documentation
- [ ] Publish the official CopyTyGo website with landing page, installation guide, versioned docs, examples, releases and changelog; define a permanent docs URL
- [ ] Add a complete end-to-end example app and newcomer quickstart
- [ ] Prepare final release notes, immutable tag, fresh `go install` verification and launch announcement
- [ ] On website launch, recommend the newest **stable** release as the default, not release candidates or internal v4 tags; keep previous version docs available by direct version URL

**Do not merge or publish a final v5 tag solely because CI is green.** Local acceptance and release-candidate validation remain required.
