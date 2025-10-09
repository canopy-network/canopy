using System;
using System.Text;

namespace CanopyPlugin.Core
{
    public static class Keys
    {
        // State key prefixes
        private static readonly byte[] ACCOUNT_PREFIX = { 0x01 };
        private static readonly byte[] POOL_PREFIX = { 0x02 };
        private static readonly byte[] PARAMS_PREFIX = { 0x07 };

        public static byte[] KeyForAccount(object address)
        {
            byte[] addressBytes = address switch
            {
                byte[] bytes => bytes,
                string str => Encoding.UTF8.GetBytes(str),
                _ => throw new ArgumentException("Address must be bytes or string", nameof(address))
            };

            return ProtoUtils.JoinLenPrefix(ACCOUNT_PREFIX, addressBytes);
        }

        public static byte[] KeyForFeeParams()
        {
            byte[] suffix = Encoding.UTF8.GetBytes("/f/");
            return ProtoUtils.JoinLenPrefix(PARAMS_PREFIX, suffix);
        }

        public static byte[] KeyForFeePool(object chainId)
        {
            byte[] chainIdBytes = ProtoUtils.FormatUInt64(chainId);
            return ProtoUtils.JoinLenPrefix(POOL_PREFIX, chainIdBytes);
        }
    }
}
