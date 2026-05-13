// Re-export types from the generated protobuf module
// This file bridges the CommonJS proto module to ESM TypeScript

// @ts-expect-error - importing CommonJS module
import protoRoot from './index.cjs';

// Export the types namespace directly
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const types: any = protoRoot.types;
