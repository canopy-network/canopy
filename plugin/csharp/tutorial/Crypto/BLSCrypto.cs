using System;
using System.Runtime.InteropServices;
using System.Text;
using Google.Protobuf;
using Types;

namespace CanopyPlugin.Tutorial.Crypto
{
    /// <summary>
    /// BLS12-381 cryptographic utilities for transaction signing.
    /// 
    /// This implementation uses P/Invoke to the Supranational blst library
    /// with a custom DST that matches the drand/kyber BDN scheme used by Canopy.
    /// </summary>
    public static class BLSCrypto
    {
        /// <summary>
        /// The Domain Separation Tag used by drand/kyber for BLS signatures.
        /// This must match the DST used by the Canopy server's BLS verification.
        /// </summary>
        private const string DST = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_";

        private const int BLST_SCALAR_BYTES = 32;
        private const int BLST_P1_BYTES = 48;
        private const int BLST_P2_BYTES = 96;

        // Platform-specific library name
        private const string BlstLibName = "blst";

        #region Native P/Invoke Declarations

        // blst scalar (secret key) - 256 bits
        [StructLayout(LayoutKind.Sequential)]
        private struct blst_scalar
        {
            [MarshalAs(UnmanagedType.ByValArray, SizeConst = 32)]
            public byte[] b;
        }

        // blst P1 point (public key) - compressed 48 bytes
        [StructLayout(LayoutKind.Sequential)]
        private struct blst_p1
        {
            [MarshalAs(UnmanagedType.ByValArray, SizeConst = 144)]
            public byte[] data;
        }

        // blst P1 affine point
        [StructLayout(LayoutKind.Sequential)]
        private struct blst_p1_affine
        {
            [MarshalAs(UnmanagedType.ByValArray, SizeConst = 96)]
            public byte[] data;
        }

        // blst P2 point (signature) - compressed 96 bytes
        [StructLayout(LayoutKind.Sequential)]
        private struct blst_p2
        {
            [MarshalAs(UnmanagedType.ByValArray, SizeConst = 288)]
            public byte[] data;
        }

        [DllImport(BlstLibName, CallingConvention = CallingConvention.Cdecl)]
        private static extern void blst_scalar_from_bendian(ref blst_scalar s, byte[] bytes);

        [DllImport(BlstLibName, CallingConvention = CallingConvention.Cdecl)]
        private static extern void blst_bendian_from_scalar(byte[] bytes, ref blst_scalar s);

        [DllImport(BlstLibName, CallingConvention = CallingConvention.Cdecl)]
        private static extern void blst_sk_to_pk_in_g1(ref blst_p1 pk, ref blst_scalar sk);

        [DllImport(BlstLibName, CallingConvention = CallingConvention.Cdecl)]
        private static extern void blst_p1_compress(byte[] bytes, ref blst_p1 point);

        [DllImport(BlstLibName, CallingConvention = CallingConvention.Cdecl)]
        private static extern void blst_hash_to_g2(
            ref blst_p2 point,
            byte[] msg, int msg_len,
            byte[] dst, int dst_len,
            byte[]? aug, int aug_len);

        [DllImport(BlstLibName, CallingConvention = CallingConvention.Cdecl)]
        private static extern void blst_sign_pk_in_g1(ref blst_p2 sig, ref blst_p2 hash, ref blst_scalar sk);

        [DllImport(BlstLibName, CallingConvention = CallingConvention.Cdecl)]
        private static extern void blst_p2_compress(byte[] bytes, ref blst_p2 point);

        #endregion

        /// <summary>
        /// Parse a secret key from hex string.
        /// </summary>
        public static byte[] SecretKeyFromHex(string hexString)
        {
            return HexToBytes(hexString);
        }

        /// <summary>
        /// Sign a message with a BLS secret key using the drand/kyber compatible scheme.
        /// 
        /// The signing process:
        /// 1. Hash the message to a G2 point using hash_to_g2 with the correct DST
        /// 2. Multiply the G2 point by the secret key scalar (this is the signature)
        /// 3. Serialize the resulting G2 point
        /// 
        /// Returns the 96-byte compressed signature.
        /// </summary>
        public static byte[] Sign(byte[] secretKey, byte[] message)
        {
            if (secretKey.Length != 32)
                throw new ArgumentException("Secret key must be 32 bytes");

            var dstBytes = Encoding.ASCII.GetBytes(DST);

            // Parse secret key
            var sk = new blst_scalar { b = new byte[32] };
            blst_scalar_from_bendian(ref sk, secretKey);

            // Hash message to G2 with our DST
            var hashPoint = new blst_p2 { data = new byte[288] };
            blst_hash_to_g2(ref hashPoint, message, message.Length, dstBytes, dstBytes.Length, null, 0);

            // Sign by multiplying the hash point with the secret key
            var sigPoint = new blst_p2 { data = new byte[288] };
            blst_sign_pk_in_g1(ref sigPoint, ref hashPoint, ref sk);

            // Compress and return the signature (96 bytes)
            var signature = new byte[BLST_P2_BYTES];
            blst_p2_compress(signature, ref sigPoint);

            return signature;
        }

        /// <summary>
        /// Get the public key bytes from a secret key.
        /// Returns the 48-byte compressed G1 public key.
        /// </summary>
        public static byte[] GetPublicKey(byte[] secretKey)
        {
            if (secretKey.Length != 32)
                throw new ArgumentException("Secret key must be 32 bytes");

            // Parse secret key
            var sk = new blst_scalar { b = new byte[32] };
            blst_scalar_from_bendian(ref sk, secretKey);

            // Generate public key
            var pk = new blst_p1 { data = new byte[144] };
            blst_sk_to_pk_in_g1(ref pk, ref sk);

            // Compress and return
            var publicKey = new byte[BLST_P1_BYTES];
            blst_p1_compress(publicKey, ref pk);

            return publicKey;
        }

        /// <summary>
        /// Get the sign bytes for a transaction.
        /// This must match the Go implementation exactly for signature verification.
        /// </summary>
        public static byte[] GetSignBytes(
            string msgType,
            Google.Protobuf.WellKnownTypes.Any msg,
            ulong time,
            ulong createdHeight,
            ulong fee,
            string memo,
            ulong networkId,
            ulong chainId)
        {
            // Create a Transaction with all fields EXCEPT signature (null for signing)
            var tx = new Transaction
            {
                MessageType = msgType,
                Msg = msg,
                // Signature is not set for sign bytes
                CreatedHeight = createdHeight,
                Time = time,
                Fee = fee,
                Memo = memo,
                NetworkId = networkId,
                ChainId = chainId
            };

            // Use deterministic marshaling
            return tx.ToByteArray();
        }

        #region Helper Methods

        /// <summary>
        /// Convert hex string to byte array.
        /// </summary>
        public static byte[] HexToBytes(string hex)
        {
            if (string.IsNullOrEmpty(hex))
                return Array.Empty<byte>();

            if (hex.Length % 2 != 0)
                throw new ArgumentException("Hex string must have even length");

            var bytes = new byte[hex.Length / 2];
            for (int i = 0; i < bytes.Length; i++)
            {
                bytes[i] = Convert.ToByte(hex.Substring(i * 2, 2), 16);
            }
            return bytes;
        }

        /// <summary>
        /// Convert byte array to hex string.
        /// </summary>
        public static string BytesToHex(byte[] bytes)
        {
            if (bytes == null || bytes.Length == 0)
                return string.Empty;

            return BitConverter.ToString(bytes).Replace("-", "").ToLowerInvariant();
        }

        #endregion
    }
}
