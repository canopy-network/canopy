/* This file contains the base contract implementation that overrides the basic 'transfer' functionality */

import Long from 'long';

import { types } from '../proto/types.js';

import {
    IPluginError,
    ErrInsufficientFunds,
    ErrInvalidAddress,
    ErrInvalidAmount,
    ErrInvalidMessageCast,
    ErrTxFeeBelowStateLimit,
} from './error.js';

import type { Plugin, Config } from './plugin.js';
import { JoinLenPrefix, FromAny, Unmarshal } from './plugin.js';

// ContractConfig: the configuration of the contract
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const ContractConfig: any = {
    name: "send",
    id: 1,
    version: 1,
    supportedTransactions: ["send"],
    transactionTypeUrls: ["type.googleapis.com/types.MessageSend"],
    eventTypeUrls: [],
    fileDescriptorProtos: [
        Buffer.from("Cg1hY2NvdW50LnByb3RvEgV0eXBlcyI7CgdBY2NvdW50EhgKB2FkZHJlc3MYASABKAxSB2FkZHJlc3MSFgoGYW1vdW50GAIgASgEUgZhbW91bnQiLgoEUG9vbBIOCgJpZBgBIAEoBFICaWQSFgoGYW1vdW50GAIgASgEUgZhbW91bnRCLlosZ2l0aHViLmNvbS9jYW5vcHktbmV0d29yay9nby1wbHVnaW4vY29udHJhY3RiBnByb3RvMw==", "base64"),
        Buffer.from("CgtldmVudC5wcm90bxIFdHlwZXMaGWdvb2dsZS9wcm90b2J1Zi9hbnkucHJvdG8ihwIKBUV2ZW50Eh0KCmV2ZW50X3R5cGUYASABKAlSCWV2ZW50VHlwZRIsCgZjdXN0b20YCyABKAsyEi50eXBlcy5FdmVudEN1c3RvbUgAUgZjdXN0b20SFgoGaGVpZ2h0GFsgASgEUgZoZWlnaHQSHAoJcmVmZXJlbmNlGFwgASgJUglyZWZlcmVuY2USGAoHY2hhaW5JZBhdIAEoBFIHY2hhaW5JZBIhCgxibG9ja19oZWlnaHQYXiABKARSC2Jsb2NrSGVpZ2h0Eh0KCmJsb2NrX2hhc2gYXyABKAxSCWJsb2NrSGFzaBIYCgdhZGRyZXNzGGAgASgMUgdhZGRyZXNzQgUKA21zZyI1CgtFdmVudEN1c3RvbRImCgNtc2cYASABKAsyFC5nb29nbGUucHJvdG9idWYuQW55UgNtc2dCLlosZ2l0aHViLmNvbS9jYW5vcHktbmV0d29yay9nby1wbHVnaW4vY29udHJhY3RiBnByb3RvMw==", "base64"),
        Buffer.from("Cgh0eC5wcm90bxIFdHlwZXMaGWdvb2dsZS9wcm90b2J1Zi9hbnkucHJvdG8iagoLVHJhbnNhY3Rpb24SIQoMbWVzc2FnZV90eXBlGAEgASgJUgttZXNzYWdlVHlwZRImCgNtc2cYAiABKAsyFC5nb29nbGUucHJvdG9idWYuQW55UgNtc2cSEAoDZmVlGAYgASgEUgNmZWUiZwoLTWVzc2FnZVNlbmQSIQoMZnJvbV9hZGRyZXNzGAEgASgMUgtmcm9tQWRkcmVzcxIdCgp0b19hZGRyZXNzGAIgASgMUgl0b0FkZHJlc3MSFgoGYW1vdW50GAMgASgEUgZhbW91bnQiJgoJRmVlUGFyYW1zEhkKCHNlbmRfZmVlGAEgASgEUgdzZW5kRmVlIkgKCVNpZ25hdHVyZRIdCgpwdWJsaWNfa2V5GAEgASgMUglwdWJsaWNLZXkSHAoJc2lnbmF0dXJlGAIgASgMUglzaWduYXR1cmVCLlosZ2l0aHViLmNvbS9jYW5vcHktbmV0d29yay9nby1wbHVnaW4vY29udHJhY3RiBnByb3RvMw==", "base64"),
        Buffer.from("CgxwbHVnaW4ucHJvdG8SBXR5cGVzGgtldmVudC5wcm90bxoIdHgucHJvdG8ikAQKC0ZTTVRvUGx1Z2luEg4KAmlkGAEgASgEUgJpZBIwCgZjb25maWcYAiABKAsyFi50eXBlcy5QbHVnaW5GU01Db25maWdIAFIGY29uZmlnEjcKB2dlbmVzaXMYAyABKAsyGy50eXBlcy5QbHVnaW5HZW5lc2lzUmVxdWVzdEgAUgdnZW5lc2lzEjEKBWJlZ2luGAQgASgLMhkudHlwZXMuUGx1Z2luQmVnaW5SZXF1ZXN0SABSBWJlZ2luEjEKBWNoZWNrGAUgASgLMhkudHlwZXMuUGx1Z2luQ2hlY2tSZXF1ZXN0SABSBWNoZWNrEjcKB2RlbGl2ZXIYBiABKAsyGy50eXBlcy5QbHVnaW5EZWxpdmVyUmVxdWVzdEgAUgdkZWxpdmVyEisKA2VuZBgHIAEoCzIXLnR5cGVzLlBsdWdpbkVuZFJlcXVlc3RIAFIDZW5kEj8KCnN0YXRlX3JlYWQYCCABKAsyHi50eXBlcy5QbHVnaW5TdGF0ZVJlYWRSZXNwb25zZUgAUglzdGF0ZVJlYWQSQgoLc3RhdGVfd3JpdGUYCSABKAsyHy50eXBlcy5QbHVnaW5TdGF0ZVdyaXRlUmVzcG9uc2VIAFIKc3RhdGVXcml0ZRIqCgVlcnJvchhjIAEoCzISLnR5cGVzLlBsdWdpbkVycm9ySABSBWVycm9yQgkKB3BheWxvYWQi5AMKC1BsdWdpblRvRlNNEg4KAmlkGAEgASgEUgJpZBItCgZjb25maWcYAiABKAsyEy50eXBlcy5QbHVnaW5Db25maWdIAFIGY29uZmlnEjgKB2dlbmVzaXMYAyABKAsyHC50eXBlcy5QbHVnaW5HZW5lc2lzUmVzcG9uc2VIAFIHZ2VuZXNpcxIyCgViZWdpbhgEIAEoCzIaLnR5cGVzLlBsdWdpbkJlZ2luUmVzcG9uc2VIAFIFYmVnaW4SMgoFY2hlY2sYBSABKAsyGi50eXBlcy5QbHVnaW5DaGVja1Jlc3BvbnNlSABSBWNoZWNrEjgKB2RlbGl2ZXIYBiABKAsyHC50eXBlcy5QbHVnaW5EZWxpdmVyUmVzcG9uc2VIAFIHZGVsaXZlchIsCgNlbmQYByABKAsyGC50eXBlcy5QbHVnaW5FbmRSZXNwb25zZUgAUgNlbmQSPgoKc3RhdGVfcmVhZBgIIAEoCzIdLnR5cGVzLlBsdWdpblN0YXRlUmVhZFJlcXVlc3RIAFIJc3RhdGVSZWFkEkEKC3N0YXRlX3dyaXRlGAkgASgLMh4udHlwZXMuUGx1Z2luU3RhdGVXcml0ZVJlcXVlc3RIAFIKc3RhdGVXcml0ZUIJCgdwYXlsb2FkIpUCCgxQbHVnaW5Db25maWcSEgoEbmFtZRgBIAEoCVIEbmFtZRIOCgJpZBgCIAEoBFICaWQSGAoHdmVyc2lvbhgDIAEoBFIHdmVyc2lvbhI1ChZzdXBwb3J0ZWRfdHJhbnNhY3Rpb25zGAQgAygJUhVzdXBwb3J0ZWRUcmFuc2FjdGlvbnMSNAoWZmlsZV9kZXNjcmlwdG9yX3Byb3RvcxgFIAMoDFIUZmlsZURlc2NyaXB0b3JQcm90b3MSMgoVdHJhbnNhY3Rpb25fdHlwZV91cmxzGAYgAygJUhN0cmFuc2FjdGlvblR5cGVVcmxzEiYKD2V2ZW50X3R5cGVfdXJscxgHIAMoCVINZXZlbnRUeXBlVXJscyI+Cg9QbHVnaW5GU01Db25maWcSKwoGY29uZmlnGAEgASgLMhMudHlwZXMuUGx1Z2luQ29uZmlnUgZjb25maWciOQoUUGx1Z2luR2VuZXNpc1JlcXVlc3QSIQoMZ2VuZXNpc19qc29uGAEgASgMUgtnZW5lc2lzSnNvbiJBChVQbHVnaW5HZW5lc2lzUmVzcG9uc2USKAoFZXJyb3IYYyABKAsyEi50eXBlcy5QbHVnaW5FcnJvclIFZXJyb3IiLAoSUGx1Z2luQmVnaW5SZXF1ZXN0EhYKBmhlaWdodBgBIAEoBFIGaGVpZ2h0ImUKE1BsdWdpbkJlZ2luUmVzcG9uc2USJAoGZXZlbnRzGAEgAygLMgwudHlwZXMuRXZlbnRSBmV2ZW50cxIoCgVlcnJvchhjIAEoCzISLnR5cGVzLlBsdWdpbkVycm9yUgVlcnJvciI4ChJQbHVnaW5DaGVja1JlcXVlc3QSIgoCdHgYASABKAsyEi50eXBlcy5UcmFuc2FjdGlvblICdHgijAEKE1BsdWdpbkNoZWNrUmVzcG9uc2USLQoSYXV0aG9yaXplZF9zaWduZXJzGAEgAygMUhFhdXRob3JpemVkU2lnbmVycxIcCglyZWNpcGllbnQYAiABKAxSCXJlY2lwaWVudBIoCgVlcnJvchhjIAEoCzISLnR5cGVzLlBsdWdpbkVycm9yUgVlcnJvciI6ChRQbHVnaW5EZWxpdmVyUmVxdWVzdBIiCgJ0eBgBIAEoCzISLnR5cGVzLlRyYW5zYWN0aW9uUgJ0eCJnChVQbHVnaW5EZWxpdmVyUmVzcG9uc2USJAoGZXZlbnRzGAEgAygLMgwudHlwZXMuRXZlbnRSBmV2ZW50cxIoCgVlcnJvchhjIAEoCzISLnR5cGVzLlBsdWdpbkVycm9yUgVlcnJvciJVChBQbHVnaW5FbmRSZXF1ZXN0EhYKBmhlaWdodBgBIAEoBFIGaGVpZ2h0EikKEHByb3Bvc2VyX2FkZHJlc3MYAiABKAxSD3Byb3Bvc2VyQWRkcmVzcyJjChFQbHVnaW5FbmRSZXNwb25zZRIkCgZldmVudHMYASADKAsyDC50eXBlcy5FdmVudFIGZXZlbnRzEigKBWVycm9yGGMgASgLMhIudHlwZXMuUGx1Z2luRXJyb3JSBWVycm9yIksKC1BsdWdpbkVycm9yEhIKBGNvZGUYASABKARSBGNvZGUSFgoGbW9kdWxlGAIgASgJUgZtb2R1bGUSEAoDbXNnGAMgASgJUgNtc2cicgoWUGx1Z2luU3RhdGVSZWFkUmVxdWVzdBIoCgRrZXlzGAEgAygLMhQudHlwZXMuUGx1Z2luS2V5UmVhZFIEa2V5cxIuCgZyYW5nZXMYAiADKAsyFi50eXBlcy5QbHVnaW5SYW5nZVJlYWRSBnJhbmdlcyI8Cg1QbHVnaW5LZXlSZWFkEhkKCHF1ZXJ5X2lkGAEgASgEUgdxdWVyeUlkEhAKA2tleRgCIAEoDFIDa2V5InQKD1BsdWdpblJhbmdlUmVhZBIZCghxdWVyeV9pZBgBIAEoBFIHcXVlcnlJZBIWCgZwcmVmaXgYAiABKAxSBnByZWZpeBIUCgVsaW1pdBgDIAEoBFIFbGltaXQSGAoHcmV2ZXJzZRgEIAEoCFIHcmV2ZXJzZSJ2ChdQbHVnaW5TdGF0ZVJlYWRSZXNwb25zZRIxCgdyZXN1bHRzGAEgAygLMhcudHlwZXMuUGx1Z2luUmVhZFJlc3VsdFIHcmVzdWx0cxIoCgVlcnJvchhjIAEoCzISLnR5cGVzLlBsdWdpbkVycm9yUgVlcnJvciJgChBQbHVnaW5SZWFkUmVzdWx0EhkKCHF1ZXJ5X2lkGAEgASgEUgdxdWVyeUlkEjEKB2VudHJpZXMYAiADKAsyFy50eXBlcy5QbHVnaW5TdGF0ZUVudHJ5UgdlbnRyaWVzInIKF1BsdWdpblN0YXRlV3JpdGVSZXF1ZXN0EiYKBHNldHMYASADKAsyEi50eXBlcy5QbHVnaW5TZXRPcFIEc2V0cxIvCgdkZWxldGVzGAIgAygLMhUudHlwZXMuUGx1Z2luRGVsZXRlT3BSB2RlbGV0ZXMiRAoYUGx1Z2luU3RhdGVXcml0ZVJlc3BvbnNlEigKBWVycm9yGGMgASgLMhIudHlwZXMuUGx1Z2luRXJyb3JSBWVycm9yIjUKC1BsdWdpblNldE9wEhAKA2tleRgBIAEoDFIDa2V5EhQKBXZhbHVlGAIgASgMUgV2YWx1ZSIiCg5QbHVnaW5EZWxldGVPcBIQCgNrZXkYASABKAxSA2tleSI6ChBQbHVnaW5TdGF0ZUVudHJ5EhAKA2tleRgBIAEoDFIDa2V5EhQKBXZhbHVlGAIgASgMUgV2YWx1ZUImWiRnaXRodWIuY29tL2Nhbm9weS1uZXR3b3JrL2Nhbm9weS9saWJiBnByb3RvMw==", "base64"),
    ],
};

// Contract() defines the smart contract that implements the extended logic of the nested chain
export class Contract {
    Config: Config;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    FSMConfig: any;
    plugin: Plugin;
    fsmId: Long;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    constructor(config: Config, fsmConfig: any, plugin: Plugin, fsmId: Long) {
        this.Config = config;
        this.FSMConfig = fsmConfig;
        this.plugin = plugin;
        this.fsmId = fsmId;
    }

    // Genesis() implements logic to import a json file to create the state at height 0 and export the state at any height
    // eslint-disable-next-line @typescript-eslint/no-explicit-any, @typescript-eslint/no-unused-vars
    Genesis(_request: any): any {
        return {}; // TODO map out original token holders
    }

    // BeginBlock() is code that is executed at the start of `applying` the block
    // eslint-disable-next-line @typescript-eslint/no-explicit-any, @typescript-eslint/no-unused-vars
    BeginBlock(_request: any): any {
        return {};
    }

    // EndBlock() is code that is executed at the end of 'applying' a block
    // eslint-disable-next-line @typescript-eslint/no-explicit-any, @typescript-eslint/no-unused-vars
    EndBlock(_request: any): any {
        return {};
    }

    // CheckMessageSend() statelessly validates a 'send' message
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    CheckMessageSend(msg: any): any {
        // check sender address
        if (!msg.fromAddress || msg.fromAddress.length !== 20) {
            return { error: ErrInvalidAddress() };
        }
        // check recipient address
        if (!msg.toAddress || msg.toAddress.length !== 20) {
            return { error: ErrInvalidAddress() };
        }
        // check amount
        const amount = msg.amount as Long | number | undefined;
        if (!amount || (Long.isLong(amount) ? amount.isZero() : amount === 0)) {
            return { error: ErrInvalidAmount() };
        }
        // return the authorized signers
        return {
            recipient: msg.toAddress,
            authorizedSigners: [msg.fromAddress],
        };
    }
}

// Async versions of contract methods for proper state handling
export class ContractAsync {
    // CheckTx() is code that is executed to statelessly validate a transaction
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    static async CheckTx(contract: Contract, request: any): Promise<any> {
        // validate fee
        const [resp, err] = await contract.plugin.StateRead(contract, {
            keys: [
                { queryId: Long.fromNumber(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER)), key: KeyForFeeParams() },
            ],
        });

        if (err) {
            return { error: err };
        }
        if (resp?.error) {
            return { error: resp.error };
        }

        // convert bytes into fee parameters
        const feeParamsBytes = resp?.results?.[0]?.entries?.[0]?.value;
        if (feeParamsBytes && feeParamsBytes.length > 0) {
            const [minFees, unmarshalErr] = Unmarshal(feeParamsBytes, types.FeeParams);
            if (unmarshalErr) {
                return { error: unmarshalErr };
            }
            // check for the minimum fee
            const txFee = request.tx?.fee as Long | number | undefined;
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const sendFee = (minFees as any)?.sendFee as Long | number | undefined;
            if (txFee !== undefined && sendFee !== undefined) {
                const txFeeNum = Long.isLong(txFee) ? txFee.toNumber() : txFee;
                const sendFeeNum = Long.isLong(sendFee) ? sendFee.toNumber() : sendFee;
                if (txFeeNum < sendFeeNum) {
                    return { error: ErrTxFeeBelowStateLimit() };
                }
            }
        }

        // get the message
        const [msg, msgErr] = FromAny(request.tx?.msg);
        if (msgErr) {
            return { error: msgErr };
        }
        // handle the message
        if (msg) {
            return contract.CheckMessageSend(msg);
        }
        return { error: ErrInvalidMessageCast() };
    }

    // DeliverTx() is code that is executed to apply a transaction
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    static async DeliverTx(contract: Contract, request: any): Promise<any> {
        // get the message
        const [msg, err] = FromAny(request.tx?.msg);
        if (err) {
            return { error: err };
        }
        // handle the message
        if (msg) {
            return ContractAsync.DeliverMessageSend(contract, msg, request.tx?.fee as Long);
        }
        return { error: ErrInvalidMessageCast() };
    }

    // DeliverMessageSend() handles a 'send' message
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    static async DeliverMessageSend(contract: Contract, msg: any, fee: Long | number | undefined): Promise<any> {
        const fromQueryId = Long.fromNumber(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));
        const toQueryId = Long.fromNumber(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));
        const feeQueryId = Long.fromNumber(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));

        // calculate the from key and to key
        const fromKey = KeyForAccount(msg.fromAddress!);
        const toKey = KeyForAccount(msg.toAddress!);
        const feePoolKey = KeyForFeePool(Long.fromNumber(contract.Config.ChainId));

        // get the from and to account
        const [response, readErr] = await contract.plugin.StateRead(contract, {
            keys: [
                { queryId: feeQueryId, key: feePoolKey },
                { queryId: fromQueryId, key: fromKey },
                { queryId: toQueryId, key: toKey },
            ],
        });

        // check for internal error
        if (readErr) {
            return { error: readErr };
        }
        // ensure no error fsm error
        if (response?.error) {
            return { error: response.error };
        }

        // get the from bytes and to bytes
        let fromBytes: Uint8Array | null = null;
        let toBytes: Uint8Array | null = null;
        let feePoolBytes: Uint8Array | null = null;

        for (const resp of response?.results || []) {
            const qid = resp.queryId as Long;
            if (qid.equals(fromQueryId)) {
                fromBytes = resp.entries?.[0]?.value || null;
            } else if (qid.equals(toQueryId)) {
                toBytes = resp.entries?.[0]?.value || null;
            } else if (qid.equals(feeQueryId)) {
                feePoolBytes = resp.entries?.[0]?.value || null;
            }
        }

        // convert the bytes to account structures
        const [fromRaw, fromErr] = Unmarshal(fromBytes || new Uint8Array(), types.Account);
        if (fromErr) {
            return { error: fromErr };
        }
        const [toRaw, toErr] = Unmarshal(toBytes || new Uint8Array(), types.Account);
        if (toErr) {
            return { error: toErr };
        }
        const [feePoolRaw, feePoolErr] = Unmarshal(feePoolBytes || new Uint8Array(), types.Pool);
        if (feePoolErr) {
            return { error: feePoolErr };
        }

        // Cast to any for property access
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const from = fromRaw as any;
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const to = toRaw as any;
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const feePool = feePoolRaw as any;

        // add fee to 'amount to deduct'
        const msgAmount = Long.isLong(msg.amount) ? msg.amount : Long.fromNumber(msg.amount as number || 0);
        const feeAmount = Long.isLong(fee) ? fee : Long.fromNumber(fee as number || 0);
        const amountToDeduct = msgAmount.add(feeAmount);

        // get from amount
        const fromAmount = Long.isLong(from?.amount) ? from.amount : Long.fromNumber(from?.amount as number || 0);

        // if the account amount is less than the amount to subtract; return insufficient funds
        if (fromAmount.lessThan(amountToDeduct)) {
            return { error: ErrInsufficientFunds() };
        }

        // for self-transfer, use same account data
        const isSelfTransfer = Buffer.from(fromKey).equals(Buffer.from(toKey));
        const toAccount = isSelfTransfer ? from : to;

        // get amounts as Long
        const newFromAmount = fromAmount.subtract(amountToDeduct);
        const toAmount = Long.isLong(toAccount?.amount) ? toAccount.amount : Long.fromNumber(toAccount?.amount as number || 0);
        const newToAmount = toAmount.add(msgAmount);
        const poolAmount = Long.isLong(feePool?.amount) ? feePool.amount : Long.fromNumber(feePool?.amount as number || 0);
        const newPoolAmount = poolAmount.add(feeAmount);

        // Update the accounts
        const updatedFrom = types.Account.create({ address: from?.address, amount: newFromAmount });
        const updatedTo = types.Account.create({ address: toAccount?.address, amount: newToAmount });
        const updatedPool = types.Pool.create({ id: feePool?.id, amount: newPoolAmount });

        // convert the accounts to bytes
        const newFromBytes = types.Account.encode(updatedFrom).finish();
        const newToBytes = types.Account.encode(updatedTo).finish();
        const newFeePoolBytes = types.Pool.encode(updatedPool).finish();

        // execute writes to the database
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        let writeResp: any;
        let writeErr: IPluginError | null;

        // if the from account is drained - delete the from account
        if (newFromAmount.isZero()) {
            [writeResp, writeErr] = await contract.plugin.StateWrite(contract, {
                sets: [
                    { key: feePoolKey, value: newFeePoolBytes },
                    { key: toKey, value: newToBytes },
                ],
                deletes: [{ key: fromKey }],
            });
        } else {
            [writeResp, writeErr] = await contract.plugin.StateWrite(contract, {
                sets: [
                    { key: feePoolKey, value: newFeePoolBytes },
                    { key: toKey, value: newToBytes },
                    { key: fromKey, value: newFromBytes },
                ],
            });
        }

        if (writeErr) {
            return { error: writeErr };
        }
        if (writeResp?.error) {
            return { error: writeResp.error };
        }

        return {};
    }
}

const accountPrefix = Buffer.from([1]); // store key prefix for accounts
const poolPrefix = Buffer.from([2]);    // store key prefix for pools
const paramsPrefix = Buffer.from([7]);  // store key prefix for governance parameters

// KeyForAccount() returns the state database key for an account
export function KeyForAccount(addr: Uint8Array): Uint8Array {
    return JoinLenPrefix(accountPrefix, Buffer.from(addr));
}

// KeyForFeeParams() returns the state database key for governance controlled 'fee parameters'
export function KeyForFeeParams(): Uint8Array {
    return JoinLenPrefix(paramsPrefix, Buffer.from("/f/"));
}

// KeyForFeePool() returns the state database key for governance controlled 'fee parameters'
export function KeyForFeePool(chainId: Long): Uint8Array {
    return JoinLenPrefix(poolPrefix, formatUint64(chainId));
}

function formatUint64(u: Long): Buffer {
    const b = Buffer.alloc(8);
    b.writeBigUInt64BE(BigInt(u.toString()));
    return b;
}
