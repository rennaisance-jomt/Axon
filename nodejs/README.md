# Axon Node.js SDK

A TypeScript-first Node.js client library for Axon browser automation.

## Installation

```bash
npm install @axon/browser
```

## Quick Start

```typescript
import { Axon } from '@axon/browser';

const axon = new Axon('http://localhost:8020/api/v1');

// Create a session
const session = await axon.createSession('mysession');
console.log(`Created session: ${session.session_id}`);

// Navigate to a URL
await axon.navigate('mysession', 'https://github.com');

// Get snapshot
const snapshot = await axon.snapshot('mysession');
console.log(`Page title: ${snapshot.title}`);

// Click an element
const result = await axon.click('mysession', 'e1');
console.log(`Action result: ${result.success}`);
```

## Configuration

The Axon client can be configured via:

1. **Constructor parameter:**
   ```typescript
   const axon = new Axon({ apiUrl: 'http://localhost:8020/api/v1' });
   ```

2. **Environment variable:**
   ```bash
   export AXON_API_URL=http://localhost:8020/api/v1
   ```
   
   ```typescript
   const axon = new Axon();  // Uses AXON_API_URL env var
   ```

## API Reference

### Session Management

```typescript
// Create a session
const session = await axon.createSession('mysession');

// Get session info
const info = await axon.getSession('mysession');

// List all sessions
const sessions = await axon.listSessions();

// Delete a session
await axon.deleteSession('mysession');
```

### Navigation

```typescript
// Navigate to a URL
await axon.navigate('mysession', 'https://github.com');
```

### Snapshots

```typescript
// Get page snapshot
const snapshot = await axon.snapshot('mysession');

// Print page elements
for (const element of snapshot.elements) {
  console.log(`${element.role}: ${element.name}`);
}
```

### Actions

```typescript
// Click
await axon.click('mysession', 'e1');

// Fill input
await axon.fill('mysession', 'e2', 'Hello World');

// Hover
await axon.hover('mysession', 'e3');

// Select option
await axon.select('mysession', 'e4', 'option1');

// Check/Uncheck
await axon.check('mysession', 'e5');
await axon.uncheck('mysession', 'e5');

// Generic action
await axon.act('mysession', 'click', 'e1');
```

### Find and Act

```typescript
// Find element by semantic description and perform action
const result = await axon.findAndAct(
  'mysession', 
  'click', 
  'search button'
);
```

## TypeScript Support

This SDK is written in TypeScript and provides full type definitions out of the box:

```typescript
import { Axon, SnapshotResponse, ActionResponse } from '@axon/browser';

const axon = new AxonScript();

// Type automatically infers return types
const snapshot: SnapshotResponse = await axon.snapshot('mysession');
const result: ActionResponse = await axon.click('mysession', 'e1');
```

## Development

```bash
# Clone the repository
git clone https://github.com/rennissance-jomt/axon
cd axon/nodejs

# Install dependencies
npm install

# Build
npm run build

# Run tests
npm test

# Lint
npm run lint
```

## License

MIT
