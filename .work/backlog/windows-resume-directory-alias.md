---
id: windows-resume-directory-alias
kind: story
tags: [infra, tests]
parent: null
depends_on: []
release_binding: null
created: 2026-11-10
updated: 2026-11-10
---

# Resume through a directory symlink on Windows

The directory alias used by TestResumeDirectoryAliasIncludesEachReplyOnce
can find the previous job, but creating the resumed job through the alias
fails at MkdirAll with "Cannot create a file when that file already exists."
The test reproduced twice on base revision 9174f06 under Go 1.27.0 on Windows.

Investigate the alias when resolving the job store and cover creation of the
resumed job along with one-time delivery of the previous inbox reply.
