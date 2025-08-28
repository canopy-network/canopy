using System;
using Types;

namespace CanopyPlugin.Core
{
    public class PluginException : Exception
    {
        private const string DefaultModule = "plugin";

        public int Code { get; }
        public string Module { get; }
        public string Msg => Message;

        public PluginException(string message, int code = 1, string module = DefaultModule) 
            : base(message)
        {
            Code = code;
            Module = module;
        }

        public PluginError ToProtobuf()
        {
            return new PluginError
            {
                Code = (ulong)Code,
                Module = Module,
                Msg = Msg
            };
        }
    }

    public class ValidationError : PluginException
    {
        public ValidationError(string message, int code = 1, string module = "plugin") 
            : base(message, code, module)
        {
        }
    }

    public class InvalidAddressError : ValidationError
    {
        public InvalidAddressError(byte[]? address = null) 
            : base(CreateMessage(address), PluginErrorCode.InvalidAddress, "contract")
        {
        }

        private static string CreateMessage(byte[]? address)
        {
            var addressStr = address != null ? Convert.ToHexString(address).ToLower() : "unknown";
            return $"Invalid address format: {addressStr}";
        }
    }

    public class InvalidAmountError : ValidationError
    {
        public InvalidAmountError(object? amount = null) 
            : base(CreateMessage(amount), PluginErrorCode.InvalidAmount, "contract")
        {
        }

        private static string CreateMessage(object? amount)
        {
            return amount != null ? $"Invalid amount: {amount}" : "Invalid amount";
        }
    }

    public class InsufficientFundsError : PluginException
    {
        public InsufficientFundsError(int? required = null, int? available = null) 
            : base(CreateMessage(required, available), PluginErrorCode.InsufficientFunds, "contract")
        {
        }

        private static string CreateMessage(int? required, int? available)
        {
            if (required.HasValue && available.HasValue)
            {
                return $"Insufficient funds: required {required}, available {available}";
            }
            return "Insufficient funds";
        }
    }

    public class FeeBelowLimitError : ValidationError
    {
        public FeeBelowLimitError(int? fee = null, int? minimum = null) 
            : base(CreateMessage(fee, minimum), PluginErrorCode.TxFeeBelowStateLimit, "contract")
        {
        }

        private static string CreateMessage(int? fee, int? minimum)
        {
            if (fee.HasValue && minimum.HasValue)
            {
                return $"Transaction fee {fee} is below state minimum {minimum}";
            }
            return "Transaction fee is below state minimum";
        }
    }

    public class UnsupportedMessageTypeError : PluginException
    {
        public UnsupportedMessageTypeError(string messageType) 
            : base($"Unsupported message type: {messageType}", 1, "contract")
        {
        }
    }

    public class PluginNotInitializedError : PluginException
    {
        public PluginNotInitializedError() 
            : base("Plugin or config not initialized", 1, "contract")
        {
        }
    }

    public class ParameterError : PluginException
    {
        public ParameterError(string message) 
            : base(message, 1, "contract")
        {
        }
    }

    public class ParameterException : PluginException
    {
        public ParameterException(string message) 
            : base(message, 1, "contract")
        {
        }
    }

    public class InvalidAddressException : ValidationError
    {
        public InvalidAddressException(object address) 
            : base($"Invalid address: {address}", PluginErrorCode.InvalidAddress, "contract")
        {
        }
    }

    public class InvalidAmountException : ValidationError
    {
        public InvalidAmountException(object amount) 
            : base($"Invalid amount: {amount}", PluginErrorCode.InvalidAmount, "contract")
        {
        }
    }

    public class InsufficientFundsException : PluginException
    {
        public InsufficientFundsException(ulong required, ulong available) 
            : base($"Insufficient funds: required {required}, available {available}", PluginErrorCode.InsufficientFunds, "contract")
        {
        }
    }

    public class FeeBelowLimitException : ValidationError
    {
        public FeeBelowLimitException(ulong fee, ulong minimum) 
            : base($"Transaction fee {fee} is below state minimum {minimum}", PluginErrorCode.TxFeeBelowStateLimit, "contract")
        {
        }
    }

    public class UnsupportedMessageTypeException : PluginException
    {
        public UnsupportedMessageTypeException(string messageType) 
            : base($"Unsupported message type: {messageType}", 1, "contract")
        {
        }
    }

    public class PluginNotInitializedException : PluginException
    {
        public PluginNotInitializedException() 
            : base("Plugin or config not initialized", 1, "contract")
        {
        }
    }

    public static class PluginErrorCode
    {
        public const int PluginTimeout = 1;
        public const int Marshal = 2;
        public const int Unmarshal = 3;
        public const int FailedPluginRead = 4;
        public const int FailedPluginWrite = 5;
        public const int InvalidPluginRespId = 6;
        public const int UnexpectedFsmToPlugin = 7;
        public const int InvalidFsmToPluginMessage = 8;
        public const int InsufficientFunds = 9;
        public const int FromAny = 10;
        public const int InvalidMessageCast = 11;
        public const int InvalidAddress = 12;
        public const int InvalidAmount = 13;
        public const int TxFeeBelowStateLimit = 14;
    }

    public static class ResponseHelper
    {
        public static T CreateErrorResponseFromException<T>(PluginException exception) where T : new()
        {
            var response = new T();
            var errorProperty = typeof(T).GetProperty("Error");
            if (errorProperty != null)
            {
                var error = exception.ToProtobuf();
                errorProperty.SetValue(response, error);
            }
            return response;
        }

        public static PluginCheckResponse CreateCheckErrorResponseFromException(PluginException exception)
        {
            return CreateErrorResponseFromException<PluginCheckResponse>(exception);
        }

        public static PluginDeliverResponse CreateDeliverErrorResponseFromException(PluginException exception)
        {
            return CreateErrorResponseFromException<PluginDeliverResponse>(exception);
        }
    }
}
