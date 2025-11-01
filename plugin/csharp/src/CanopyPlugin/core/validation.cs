using System;
using System.Text;

namespace CanopyPlugin.Core
{
    /// <summary>
    /// Data validation and normalization functions for the Canopy blockchain plugin.
    ///
    /// Provides functions for validating and converting address and amount formats.
    /// </summary>
    public static class Validation
    {
        /// <summary>
        /// Validate that an address is exactly 20 bytes.
        /// Used in transaction validation.
        /// </summary>
        /// <param name="address">Address to validate (byte array or string)</param>
        /// <returns>True if address is valid (exactly 20 bytes)</returns>
        public static bool ValidateAddress(object address)
        {
            try
            {
                byte[] addressBytes;

                if (address is string addressStr)
                {
                    // Handle hex strings
                    if (addressStr.StartsWith("0x", StringComparison.OrdinalIgnoreCase))
                    {
                        addressBytes = Convert.FromHexString(addressStr[2..]);
                    }
                    else
                    {
                        addressBytes = Encoding.UTF8.GetBytes(addressStr);
                    }
                }
                else if (address is byte[] bytes)
                {
                    addressBytes = bytes;
                }
                else
                {
                    return false;
                }

                return addressBytes.Length == 20;
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Exception in ValidateAddress: {ex}");
                return false;
            }
        }

        /// <summary>
        /// Validate that an amount is greater than 0.
        /// Used in transaction validation.
        /// </summary>
        /// <param name="amount">Amount to validate (int or string)</param>
        /// <returns>True if amount is valid (greater than 0)</returns>
        public static bool ValidateAmount(object amount)
        {
            try
            {
                return amount switch
                {
                    string amountStr => ulong.TryParse(amountStr, out var parsed) && parsed > 0,
                    ulong ul => ul > 0,
                    long l => l > 0,
                    int i => i > 0,
                    _ => false
                };
            }
            catch (Exception)
            {
                return false;
            }
        }

        /// <summary>
        /// Convert various address formats to bytes.
        /// </summary>
        /// <param name="address">Address in various formats</param>
        /// <returns>Bytes representation of the address</returns>
        /// <exception cref="ArgumentException">If address cannot be converted or is invalid length</exception>
        public static byte[] NormalizeAddress(object address)
        {
            if (!ValidateAddress(address))
            {
                throw new ArgumentException("Invalid address: must be exactly 20 bytes");
            }

            if (address is string addressStr)
            {
                if (addressStr.StartsWith("0x", StringComparison.OrdinalIgnoreCase))
                {
                    return Convert.FromHexString(addressStr[2..]);
                }
                else
                {
                    return Encoding.UTF8.GetBytes(addressStr);
                }
            }

            return (byte[])address;
        }

        /// <summary>
        /// Convert various amount formats to ulong for arithmetic.
        /// </summary>
        /// <param name="amount">Amount in various formats</param>
        /// <returns>ulong representation of the amount</returns>
        /// <exception cref="ArgumentException">If amount cannot be converted or is invalid</exception>
        public static ulong NormalizeAmount(object amount)
        {
            if (!ValidateAmount(amount))
            {
                throw new ArgumentException("Invalid amount: must be greater than 0");
            }

            if (amount is string amountStr)
            {
                return ulong.Parse(amountStr);
            }

            if (amount is ulong ul)
            {
                return ul;
            }

            if (amount is int i && i > 0)
            {
                return (ulong)i;
            }

            if (amount is long l && l > 0)
            {
                return (ulong)l;
            }

            throw new ArgumentException($"Cannot convert {amount?.GetType()} to ulong");
        }
    }
}
