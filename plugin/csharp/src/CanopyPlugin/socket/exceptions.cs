using System;
using CanopyPlugin.Core;

namespace CanopyPlugin.Socket
{
    /// <summary>
    /// Socket communication exceptions for the Canopy blockchain plugin.
    /// Provides socket-specific error handling for protocol communication and marshaling operations.
    /// </summary>
    
    public class MarshalError : PluginException
    {
        public MarshalError(object originalError) 
            : base($"marshal() failed with err: {originalError}", PluginErrorCode.Marshal, "plugin")
        {
        }
    }

    public class UnmarshalError : PluginException
    {
        public UnmarshalError(object originalError) 
            : base($"unmarshal() failed with err: {originalError}", PluginErrorCode.Unmarshal, "plugin")
        {
        }
    }

    public class SocketTimeoutError : PluginException
    {
        public SocketTimeoutError(string requestType = "request", double? timeout = null) 
            : base(BuildMessage(requestType, timeout), PluginErrorCode.PluginTimeout, "socket")
        {
        }

        private static string BuildMessage(string requestType, double? timeout)
        {
            return timeout.HasValue 
                ? $"{requestType} timed out after {timeout}s" 
                : $"{requestType} timed out";
        }
    }

    public class SocketTimeoutException : PluginException
    {
        public SocketTimeoutException(string requestType, TimeSpan timeout) 
            : base($"{requestType} timed out after {timeout.TotalSeconds}s", PluginErrorCode.PluginTimeout, "socket")
        {
        }
    }

    public class SocketConnectionException : PluginException
    {
        public SocketConnectionException(string message) 
            : base(message, 1, "socket")
        {
        }
    }

    public class InvalidSocketResponseException : PluginException
    {
        public InvalidSocketResponseException(string expected) 
            : base($"No {expected} response received", 1, "socket")
        {
        }
    }

    public class SocketConnectionError : PluginException
    {
        public SocketConnectionError(string message = "Socket connection failed") 
            : base(message, 1, "socket")
        {
        }
    }

    public class InvalidSocketResponseError : PluginException
    {
        public InvalidSocketResponseError(string expected, string? received = null) 
            : base(BuildMessage(expected, received), 1, "socket")
        {
        }

        private static string BuildMessage(string expected, string? received)
        {
            return !string.IsNullOrEmpty(received) 
                ? $"Expected {expected} response, but received {received}" 
                : $"No {expected} response received";
        }
    }
}
