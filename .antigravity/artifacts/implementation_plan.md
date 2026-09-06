# Implementation Plan
1. Target: `src/middleware/auth.ts`
2. Assumption: Client passes Bearer token in headers.
3. Note: Avoided importing `ioredis` to keep container footprint light.
