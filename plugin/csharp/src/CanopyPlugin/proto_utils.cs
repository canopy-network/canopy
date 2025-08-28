using System;
using System.Collections.Generic;
using System.Text;
using System.Text.Json;
using Google.Protobuf;
using Types;

namespace CanopyPlugin.Core
{
    public static class ProtoUtils
    {
        public static byte[] Marshal(IMessage message)
        {
            try
            {
                return message.ToByteArray();
            }
            catch (Exception ex)
            {
                throw new ArgumentException($"Marshal failed: {ex.Message}", ex);
            }
        }

        public static byte[] Marshal(object obj)
        {
            try
            {
                if (obj is IMessage message)
                {
                    return message.ToByteArray();
                }
                else
                {
                    var json = JsonSerializer.Serialize(obj);
                    return Encoding.UTF8.GetBytes(json);
                }
            }
            catch (Exception ex)
            {
                throw new ArgumentException($"Marshal failed: {ex.Message}", ex);
            }
        }

        public static T Unmarshal<T>(byte[] data) where T : IMessage<T>, new()
        {
            try
            {
                if (data == null || data.Length == 0)
                {
                    return default(T)!;
                }

                var parser = new MessageParser<T>(() => new T());
                return parser.ParseFrom(data);
            }
            catch (Exception ex)
            {
                throw new ArgumentException($"Unmarshal failed: {ex.Message}", ex);
            }
        }

        public static object Unmarshal(Type messageType, byte[] data)
        {
            try
            {
                if (data == null || data.Length == 0)
                {
                    return null!;
                }

                if (typeof(IMessage).IsAssignableFrom(messageType))
                {
                    var method = typeof(ProtoUtils).GetMethod("Unmarshal", new[] { typeof(byte[]) });
                    var genericMethod = method!.MakeGenericMethod(messageType);
                    return genericMethod.Invoke(null, new object[] { data })!;
                }
                else
                {
                    var json = Encoding.UTF8.GetString(data);
                    return JsonSerializer.Deserialize(json, messageType)!;
                }
            }
            catch (Exception ex)
            {
                throw new ArgumentException($"Unmarshal failed: {ex.Message}", ex);
            }
        }

        public static IMessage FromAny(Dictionary<string, object> anyMessage)
        {
            try
            {
                if (anyMessage == null)
                {
                    throw new ArgumentException("Any message is null or undefined");
                }

                var typeUrl = anyMessage.ContainsKey("typeUrl") ? anyMessage["typeUrl"]?.ToString() :
                             anyMessage.ContainsKey("type_url") ? anyMessage["type_url"]?.ToString() : null;
                
                var value = anyMessage.ContainsKey("value") ? anyMessage["value"] : null;

                if (string.IsNullOrEmpty(typeUrl))
                {
                    throw new ArgumentException("Any message missing type URL");
                }

                if (value == null)
                {
                    throw new ArgumentException("Any message missing value");
                }

                var typeName = typeUrl.Contains("/") ? typeUrl.Substring(typeUrl.LastIndexOf("/") + 1) : typeUrl;

                byte[] valueBytes;
                if (value is byte[] bytes)
                {
                    valueBytes = bytes;
                }
                else if (value is string str)
                {
                    valueBytes = Convert.FromBase64String(str);
                }
                else
                {
                    throw new ArgumentException("Value must be byte array or base64 string");
                }

                switch (typeName)
                {
                    case "MessageSend":
                    case "types.MessageSend":
                        var messageSend = Unmarshal<Types.MessageSend>(valueBytes);
                        if (messageSend == null)
                        {
                            throw new ArgumentException("Failed to unmarshal MessageSend");
                        }
                        return messageSend;

                    case "Transaction":
                    case "types.Transaction":
                        var transaction = Unmarshal<Types.Transaction>(valueBytes);
                        if (transaction == null)
                        {
                            throw new ArgumentException("Failed to unmarshal Transaction");
                        }
                        return transaction;

                    default:
                        throw new ArgumentException($"Unknown message type in Any: {typeName}");
                }
            }
            catch (Exception ex)
            {
                throw new ArgumentException($"FromAny failed: {ex.Message}", ex);
            }
        }

        public static byte[] JoinLenPrefix(params byte[][] items)
        {
            var result = new List<byte>();

            foreach (var item in items)
            {
                if (item == null || item.Length == 0)
                {
                    continue;
                }

                if (item.Length > 255)
                {
                    throw new ArgumentException($"Item too long: {item.Length} bytes (max 255)");
                }

                result.Add((byte)item.Length);
                result.AddRange(item);
            }

            return result.ToArray();
        }

        public static byte[] FormatUInt64(ulong value)
        {
            var bytes = BitConverter.GetBytes(value);
            if (BitConverter.IsLittleEndian)
            {
                Array.Reverse(bytes);
            }
            return bytes;
        }

        public static byte[] FormatUInt64(string value)
        {
            if (!ulong.TryParse(value, out var parsed))
            {
                throw new ArgumentException($"Invalid uint64 value: {value}");
            }
            return FormatUInt64(parsed);
        }

        public static byte[] FormatUInt64(object value)
        {
            return value switch
            {
                ulong ul => FormatUInt64(ul),
                long l when l >= 0 => FormatUInt64((ulong)l),
                int i when i >= 0 => FormatUInt64((ulong)i),
                string s => FormatUInt64(s),
                _ => throw new ArgumentException($"Cannot convert {value?.GetType()} to uint64")
            };
        }
    }
}
