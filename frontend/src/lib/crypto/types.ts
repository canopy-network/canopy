export interface EncryptedPayload {
  ciphertext: string; // hex
  iv: string;         // hex
  salt: string;       // hex
}

export interface KeystoreRecord extends EncryptedPayload {
  version: number;
  kdf: string;
  pbkdf2Iterations: number;
  address: string;
  publicKeyHex: string;
  createdAt: number;
}
