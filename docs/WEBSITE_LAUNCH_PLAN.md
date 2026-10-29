# CopyTyGo official website launch plan

Status: **planned**. The official website and its domain have not been published or chosen. This is a content and release plan, not a claim that a site is live.

## Product position

CopyTyGo combines Go backend development with TypeScript frontend workflows, a `ctg` CLI, authentication, database tools, background jobs and a development Studio. The first release recommended to the public will be **v5.0.0**, after its immutable tag and clean public installation are verified. Internal v4.0.6 is a prelaunch milestone, not the public default.

Do not invent a domain, user numbers, benchmarks, endorsements, or claims about production adoption. Document observed functionality, supported versions and tested deployment scenarios.

## Required website pages

| Public route | Primary purpose |
| --- | --- |
| `/` | Framework positioning, feature highlights, example screenshot, install CTA and quick start |
| `/docs` | Stable documentation alias, pointing to the **latest published stable release**, not an RC |
| `/docs/v5` | Version-pinned v5 docs (and separate paths for future supported majors) |
| `/docs/v5/installation` | Go/Node requirements, installation, platform notes and troubleshooting |
| `/docs/v5/getting-started` | Create an application, run CLI, configure DB, run it locally |
| `/docs/v5/authentication` | Single/multi role, sessions, CSRF and examples |
| `/docs/v5/database` | Models, resource generators, migrations, MySQL and PostgreSQL |
| `/docs/v5/frontend` | TypeScript first, with React and Vue options |
| `/docs/v5/queues` | Queue drivers, scheduler, persistent jobs and retries |
| `/docs/v5/deployment` | Build, environment, production security, storage and infrastructure |
| `/examples` | Small reproducible end-to-end starter project and screenshots |
| `/releases` | Stable releases, changelog and release dates |
| `/community` | GitHub issues/discussions, contributing and support links |

The website navigation should keep **Documentation**, **Getting Started**, **Examples**, **Releases**, and **GitHub** easy to find. A small version selector should show the current stable version and a clearly labeled pre-release section only when needed. Docs search, responsive layout, mobile menu and copyable code examples are launch requirements.

## Stable version selection rules

1. No version should be called stable until an immutable `vX.Y.Z` tag exists, CI passes on that exact SHA, and fresh `go install` has been verified outside the repository.
2. The main install button must display and copy the latest **stable** command, for example `go install github.com/arfajhf/copytygo/v5/cmd/ctg@v5.0.0` **only once v5.0.0 is actually released**.
3. Later stable patch/minor releases should become the recommended version after their own checks pass. Never make alpha, beta or RC tags the default.
4. Keep previously published docs at versioned URLs so links do not break. Redirect `/docs` to the current stable docs, rather than deleting old content.
5. The website release/version manifest should be driven by approved GitHub Release metadata or a reviewed source-controlled version file. A GitHub tag alone is not proof of public readiness.
6. Prelaunch website copy should say “Coming soon / release candidate”, not “Download stable”, until release approval.

## Content and technical requirements

- Use the existing CopyTyGo logo and coherent branding from the repository.
- Show real CLI output or screenshots from tested applications. Avoid unverified performance claims.
- Publish static/pre-rendered, crawlable documentation content, not an empty client-only page.
- Use canonical URLs, descriptive page titles, concise meta descriptions, a sitemap, robots rules, accessible headings and proper internal links.
- Keep code blocks executable, version-specific and tested in CI where possible.
- Separate the documentation **website** from the built-in **local development Studio**; Studio must stay unavailable in production applications.
- Provide copyable Linux, macOS and Windows commands where behavior differs; Windows troubleshooting must not recommend disabling security controls.
- Link to the repository, security guidance, changelog and license. Document a real stable support policy rather than implying unlimited support.
- Decide the official domain before finalizing `version.DocsURL`, README installation CTAs or the welcome-page documentation button.
- Ensure the website's release channel is updated only **after** publishing the approved final tag.

## Suggested launch sequence

1. Finalize security review and release-candidate acceptance; keep `main` and existing v4 tags unchanged until approved.
2. Select domain, hosting, branding and documentation frontend. Keep Go + TypeScript framework docs as the source of truth.
3. Build landing pages, versioned documentation, guide search and an end-to-end example app.
4. Verify site links, mobile accessibility, structured content, sitemap and documented commands.
5. Tag and verify `v5.0.0`, test `go install ...@v5.0.0` from a fresh environment, and publish GitHub Release notes.
6. Switch the website's default recommendation to v5.0.0, update the framework's `version.DocsURL`, and publish the launch announcement.

**No public website or final tag is created by this plan.**
