# Authentication

Credential storage and resolution.

## Key Rules

- Resolution order: `LINEAR_API_KEY` env → native keyring → file → interactive prompt.
- Never persist credentials without user consent.
- All providers implement the `Provider` interface — inject mocks for testing.

## Contents

- [Credentials](credentials.md) — provider chain, keyring backends, and storage locations.
