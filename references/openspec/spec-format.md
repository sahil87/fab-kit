# OpenSpec Specification Format

## Overview

OpenSpec uses structured markdown for specifications. This document covers:
- Main spec format (source of truth)
- Delta spec format (changes)
- Parsing rules
- Validation requirements

## Main Spec Format

### Required Structure

```markdown
# [Domain] Specification

## Purpose

[High-level description of what this specification covers]

## Requirements

### Requirement: [Name]

[Requirement description using RFC 2119 keywords]

#### Scenario: [Name]
- GIVEN [initial context]
- WHEN [action occurs]
- THEN [expected outcome]
```

### Required Sections

| Section | Required | Purpose |
|---------|----------|---------|
| `# Title` | Yes | Spec identification |
| `## Purpose` | Yes | High-level description |
| `## Requirements` | Yes | Container for requirements |
| `### Requirement:` | Yes (at least 1) | Individual requirement |
| `#### Scenario:` | Yes (at least 1 per req) | Test case |

### RFC 2119 Keywords

| Keyword | Level | Meaning |
|---------|-------|---------|
| `MUST` | Absolute | Required for compliance |
| `SHALL` | Absolute | Same as MUST |
| `MUST NOT` | Absolute | Prohibited |
| `SHALL NOT` | Absolute | Same as MUST NOT |
| `SHOULD` | Recommended | Best practice |
| `SHOULD NOT` | Discouraged | Not recommended |
| `MAY` | Optional | Permitted but not required |

### Complete Example

```markdown
# Authentication Specification

## Purpose

This specification defines the authentication behavior for the platform,
including login, logout, and session management.

## Requirements

### Requirement: User Login

The system SHALL authenticate users with valid email and password combinations.

The system MUST NOT reveal whether an email exists when authentication fails.

The system SHOULD rate-limit failed login attempts.

#### Scenario: Successful Login

- GIVEN a registered user with email "user@example.com"
- AND password "ValidP@ssw0rd"
- WHEN they submit a login request with correct credentials
- THEN the system returns a 200 status
- AND provides a JWT access token
- AND provides a refresh token
- AND sets appropriate cookie headers

#### Scenario: Invalid Credentials

- GIVEN an email "user@example.com"
- WHEN they submit a login request with incorrect password
- THEN the system returns a 401 status
- AND provides a generic error message "Invalid credentials"
- AND does not reveal whether the email exists

#### Scenario: Rate Limiting

- GIVEN 5 failed login attempts for "user@example.com"
- WHEN another login attempt is made
- THEN the system returns a 429 status
- AND indicates the retry-after period

### Requirement: Session Management

The system SHALL invalidate sessions on logout.

The system MUST expire access tokens after 1 hour.

The system MAY extend sessions with refresh tokens.

#### Scenario: Logout

- GIVEN an authenticated user
- WHEN they submit a logout request
- THEN the system invalidates their session
- AND clears authentication cookies
- AND returns a 204 status
```

## Delta Spec Format

Delta specs describe changes to existing specifications.

### Structure

```markdown
# [Domain] Specification Changes

## ADDED Requirements

[New requirements being added]

## MODIFIED Requirements

[Existing requirements being updated]

## REMOVED Requirements

[Requirements being deprecated]
```

### Sections

| Section | Purpose | Archive Behavior |
|---------|---------|------------------|
| `## ADDED Requirements` | New functionality | Appended to main spec |
| `## MODIFIED Requirements` | Changed behavior | Replaces matching requirement |
| `## REMOVED Requirements` | Deprecated functionality | Deleted from main spec |

### Delta Spec Example

```markdown
# Authentication Specification Changes

## ADDED Requirements

### Requirement: Two-Factor Authentication

The system SHALL support TOTP-based two-factor authentication.

The system SHOULD offer SMS as a fallback 2FA method.

#### Scenario: TOTP Enrollment

- GIVEN an authenticated user without 2FA
- WHEN they request 2FA enrollment
- THEN the system generates a TOTP secret
- AND displays a QR code for authenticator apps
- AND provides manual entry code

#### Scenario: TOTP Verification

- GIVEN a user with 2FA enabled
- AND valid email/password credentials
- WHEN they provide a valid TOTP code
- THEN authentication completes successfully

## MODIFIED Requirements

### Requirement: User Login

The system SHALL authenticate users with valid email and password combinations.

The system MUST NOT reveal whether an email exists when authentication fails.

The system MUST require 2FA verification when enabled for the account.

#### Scenario: Login with 2FA

- GIVEN a user with 2FA enabled
- AND valid email/password credentials
- WHEN they submit login without TOTP
- THEN the system returns a 202 status
- AND indicates 2FA is required

## REMOVED Requirements

### Requirement: Remember Me

<!-- This requirement is being removed in favor of refresh token approach -->
```

## Parsing Rules

### Requirement Block Identification

```typescript
// Requirement starts with "### Requirement: "
const REQUIREMENT_PATTERN = /^### Requirement:\s*(.+)$/m;

// Requirement block ends at next "### Requirement:" or section
function extractRequirementBlock(content: string, name: string): string {
  const start = content.indexOf(`### Requirement: ${name}`);
  const nextReq = content.indexOf('### Requirement:', start + 1);
  const nextSection = content.indexOf('## ', start + 1);

  const end = Math.min(
    nextReq > -1 ? nextReq : Infinity,
    nextSection > -1 ? nextSection : Infinity,
    content.length
  );

  return content.substring(start, end).trim();
}
```

### Scenario Parsing

```typescript
// Scenario pattern
const SCENARIO_PATTERN = /^#### Scenario:\s*(.+)$/m;

// Step patterns (case-insensitive)
const STEP_PATTERNS = {
  GIVEN: /^-\s*GIVEN\s+(.+)$/i,
  WHEN: /^-\s*WHEN\s+(.+)$/i,
  THEN: /^-\s*THEN\s+(.+)$/i,
  AND: /^-\s*AND\s+(.+)$/i,
};
```

### Delta Section Parsing

```typescript
// Delta section patterns
const ADDED_SECTION = /^## ADDED Requirements$/m;
const MODIFIED_SECTION = /^## MODIFIED Requirements$/m;
const REMOVED_SECTION = /^## REMOVED Requirements$/m;

interface DeltaSpec {
  added: RequirementBlock[];
  modified: RequirementBlock[];
  removed: RequirementBlock[];
}

function parseDeltaSpec(content: string): DeltaSpec {
  return {
    added: extractSection(content, 'ADDED'),
    modified: extractSection(content, 'MODIFIED'),
    removed: extractSection(content, 'REMOVED'),
  };
}
```

## Archive Merge Algorithm

### Process

```typescript
function mergeSpecs(mainSpec: string, delta: DeltaSpec): string {
  let result = mainSpec;

  // 1. Process REMOVED (delete matching requirements)
  for (const removed of delta.removed) {
    result = removeRequirement(result, removed.name);
  }

  // 2. Process MODIFIED (replace matching requirements)
  for (const modified of delta.modified) {
    result = replaceRequirement(result, modified.name, modified.content);
  }

  // 3. Process ADDED (append to Requirements section)
  for (const added of delta.added) {
    result = appendRequirement(result, added.content);
  }

  return result;
}
```

### Requirement Matching

Requirements are matched by exact name:

```typescript
function findRequirement(spec: string, name: string): number {
  const pattern = new RegExp(`^### Requirement:\\s*${escapeRegex(name)}$`, 'm');
  const match = pattern.exec(spec);
  return match ? match.index : -1;
}
```

### Edge Cases

| Case | Behavior |
|------|----------|
| MODIFIED not found | Warning, treated as ADDED |
| REMOVED not found | Warning, no action |
| Duplicate names | First match used |
| Malformed section | Validation error |

## Validation Rules

### Required Validations

```typescript
interface ValidationResult {
  valid: boolean;
  errors: ValidationError[];
  warnings: ValidationWarning[];
}

function validateSpec(spec: string): ValidationResult {
  const errors: ValidationError[] = [];
  const warnings: ValidationWarning[] = [];

  // Check required sections
  if (!spec.includes('## Purpose')) {
    errors.push({ type: 'MISSING_SECTION', section: 'Purpose' });
  }

  if (!spec.includes('## Requirements')) {
    errors.push({ type: 'MISSING_SECTION', section: 'Requirements' });
  }

  // Check for at least one requirement
  if (!REQUIREMENT_PATTERN.test(spec)) {
    errors.push({ type: 'NO_REQUIREMENTS' });
  }

  // Check each requirement has scenarios
  const requirements = extractRequirements(spec);
  for (const req of requirements) {
    if (!SCENARIO_PATTERN.test(req.content)) {
      warnings.push({
        type: 'NO_SCENARIOS',
        requirement: req.name
      });
    }
  }

  // Check RFC 2119 keyword usage
  const keywords = ['MUST', 'SHALL', 'SHOULD', 'MAY'];
  for (const req of requirements) {
    const hasKeyword = keywords.some(kw => req.content.includes(kw));
    if (!hasKeyword) {
      warnings.push({
        type: 'NO_RFC_KEYWORDS',
        requirement: req.name,
      });
    }
  }

  return {
    valid: errors.length === 0,
    errors,
    warnings,
  };
}
```

### Strict Mode

With `--strict` flag, warnings become errors:

```typescript
function validateStrict(spec: string): ValidationResult {
  const result = validateSpec(spec);

  return {
    valid: result.errors.length === 0 && result.warnings.length === 0,
    errors: [...result.errors, ...result.warnings.map(w => ({
      type: w.type,
      ...w,
    }))],
    warnings: [],
  };
}
```

## Code Examples in Specs

### Inline Code

```markdown
The system SHALL validate `email` format using RFC 5322.
```

### Code Blocks

````markdown
#### Scenario: API Response

- GIVEN a valid request
- WHEN the system responds
- THEN the response matches:
  ```json
  {
    "status": "success",
    "data": {
      "id": "<uuid>",
      "createdAt": "<iso-8601>"
    }
  }
  ```
````

### Schema Definitions

````markdown
### Requirement: User Schema

The system SHALL validate users against:

```typescript
interface User {
  id: string;          // UUID v4
  email: string;       // RFC 5322 format
  name: string;        // 1-100 characters
  createdAt: Date;     // ISO 8601
  role: 'admin' | 'user' | 'guest';
}
```
````

## Organizing Specs

### By Domain

```
openspec/specs/
├── auth/
│   ├── login.md
│   ├── logout.md
│   └── 2fa.md
├── payments/
│   ├── checkout.md
│   └── refunds.md
└── users/
    ├── registration.md
    └── profile.md
```

### By Feature Area

```
openspec/specs/
├── api/
│   └── v1/
│       ├── users.md
│       └── products.md
├── webhooks/
│   └── events.md
└── admin/
    └── dashboard.md
```

### Naming Convention

- Use kebab-case: `user-registration.md`
- Keep names descriptive but concise
- Match domain terminology
