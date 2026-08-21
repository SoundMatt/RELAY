# Security Policy

## Scope

RELAY is a specification and canonical-type/tooling library, not a network
driver — it does not itself transmit messages to a bus (see §1 of
`spec/relay-spec.md` and `docs/tool-safety-manual.md` §2). The parts of this
repo that process input from outside the local build — `relay conform` and
`relay interop` parsing another binary's CLI output, `relay convert` reading
stdin, envelope conversion (`ToMessage`/`FromMessage`) — are the realistic
attack surface. Report a vulnerability in any of these, or in the canonical
`ValidateFrame` constraints themselves failing to reject malformed input,
through the process below.

## Supported versions

RELAY follows the versioning scheme in `spec/relay-spec.md` §19: the current
stable MAJOR.MINOR series (see `spec/version.json`) receives security fixes.
Older MAJOR versions are not maintained once superseded.

## Reporting a vulnerability

**Do not open a public GitHub issue for a security report.**

Use GitHub's private vulnerability reporting for this repository:
[github.com/SoundMatt/RELAY/security/advisories/new](https://github.com/SoundMatt/RELAY/security/advisories/new).
This opens a private draft security advisory visible only to you and the
repository maintainers, and lets us collaborate on a fix before any public
disclosure.

Include, where you can:

- The affected version/commit.
- A minimal reproduction (a canonical-type value, a wire fragment, or a CLI
  invocation) demonstrating the issue.
- Impact — what a successful exploit would let an attacker do.

## What to expect

This is a small, community-maintained project without a dedicated security
team or a formal SLA. We aim to acknowledge new reports promptly and will
work with you on a fix and coordinated disclosure timeline once the report
is triaged.
