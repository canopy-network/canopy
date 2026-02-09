package com.canopy.tutorial.crypto

import com.google.protobuf.Any
import supranational.blst.P2
import supranational.blst.SecretKey
import types.Tx

/**
 * BLS12-381 cryptographic utilities for transaction signing.
 * 
 * This implementation uses the supranational blst library with a custom DST
 * that matches the drand/kyber BDN scheme used by Canopy.
 * 
 * The DST (Domain Separation Tag) is critical - it must match exactly what
 * the server uses for signature verification.
 */
object BLSCrypto {
    
    /**
     * The Domain Separation Tag used by drand/kyber for BLS signatures.
     * This must match the DST used by the Canopy server's BLS verification.
     */
    private const val DST = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_"
    
    /**
     * Create a BLS secret key from hex-encoded private key bytes.
     * The private key is expected in big-endian format (same as drand/kyber).
     */
    fun secretKeyFromHex(hexString: String): SecretKey {
        val bytes = hexString.hexToBytes()
        val sk = SecretKey()
        sk.from_bendian(bytes)
        return sk
    }
    
    /**
     * Sign a message with a BLS secret key using the drand/kyber compatible scheme.
     * 
     * The signing process:
     * 1. Hash the message to a G2 point using hash_to_g2 with the correct DST
     * 2. Multiply the G2 point by the secret key scalar (this is the signature)
     * 3. Serialize the resulting G2 point
     * 
     * Returns the 96-byte compressed signature.
     */
    fun sign(secretKey: SecretKey, message: ByteArray): ByteArray {
        // Create a new P2 point by hashing the message to G2 with our DST
        val p2 = P2()
        p2.hash_to(message, DST)
        
        // Sign by multiplying the hash point with the secret key
        // This modifies p2 in place
        p2.sign_with(secretKey)
        
        // Compress and return the signature (96 bytes)
        return p2.compress()
    }
    
    /**
     * Get the public key bytes from a secret key.
     * Returns the 48-byte compressed G1 public key.
     */
    fun getPublicKey(secretKey: SecretKey): ByteArray {
        // Generate the public key point on G1
        val p1 = supranational.blst.P1.generator()
        p1.mult(secretKeyToScalar(secretKey))
        return p1.compress()
    }
    
    /**
     * Convert SecretKey to Scalar for point multiplication.
     */
    private fun secretKeyToScalar(sk: SecretKey): supranational.blst.Scalar {
        val scalar = supranational.blst.Scalar()
        scalar.from_bendian(sk.to_bendian())
        return scalar
    }
    
    /**
     * Get the sign bytes for a transaction.
     * This must match the Go implementation exactly for signature verification.
     */
    fun getSignBytes(
        msgType: String,
        msg: Any,
        time: Long,
        createdHeight: Long,
        fee: Long,
        memo: String,
        networkId: Long,
        chainId: Long
    ): ByteArray {
        // Create a Transaction with all fields EXCEPT signature (null for signing)
        val tx = Tx.Transaction.newBuilder()
            .setMessageType(msgType)
            .setMsg(msg)
            // signature is not set for sign bytes
            .setCreatedHeight(createdHeight)
            .setTime(time)
            .setFee(fee)
            .setMemo(memo)
            .setNetworkId(networkId)
            .setChainId(chainId)
            .build()
        
        // Use deterministic marshaling (protobuf default is deterministic in Java)
        return tx.toByteArray()
    }
}

/**
 * Extension function to convert ByteArray to hex string.
 */
@OptIn(ExperimentalStdlibApi::class)
fun ByteArray.toHexString(): String = this.toHexString(HexFormat.Default)

/**
 * Extension function to convert hex string to ByteArray.
 */
@OptIn(ExperimentalStdlibApi::class)
fun String.hexToBytes(): ByteArray = this.hexToByteArray(HexFormat.Default)
