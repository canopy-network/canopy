export interface EncryptedPayload {
  ciphertext: string; // hex
  iv: string;         // hex
  salt: string;       // hex
}

export type KdfName = "argon2id" | "pbkdf2-sha256" | "canopy-cli";

export interface KeystoreRecord extends EncryptedPayload {
  version: number;
  kdf: KdfName;
  // present when kdf === "argon2id"
  argon2?: { time: number; memKb: number; parallelism: number; keylen: number };
  // present when kdf === "pbkdf2-sha256" (legacy)
  pbkdf2Iterations?: number;
  address: string;
  publicKeyHex: string;
  createdAt: number;
}
