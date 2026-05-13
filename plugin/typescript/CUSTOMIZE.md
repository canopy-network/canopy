# Custom Transaction Guide - TypeScript

This guide shows how to add a custom "reward" transaction to the TypeScript plugin.

## Step 1: Register the Transaction

In `src/contract/contract.ts`:

```typescript
export const ContractConfig: any = {
    name: "send",
    id: 1,
    version: 1,
    supportedTransactions: ["send", "reward"],  // Add "reward"
};
```

## Step 2: Add CheckTx Validation

CheckTx performs stateless validation and returns authorized signers.

```typescript
// In contract.ts - add to Contract class
CheckMessageReward(msg: any): any {
    if (!msg.adminAddress || msg.adminAddress.length !== 20) {
        return { error: ErrInvalidAddress() };
    }
    if (!msg.recipientAddress || msg.recipientAddress.length !== 20) {
        return { error: ErrInvalidAddress() };
    }
    if (!msg.amount || msg.amount === 0) {
        return { error: ErrInvalidAmount() };
    }
    return {
        recipient: msg.recipientAddress,
        authorizedSigners: [msg.adminAddress],
    };
}

// In ContractAsync.CheckTx - add routing
static async CheckTx(contract: Contract, request: any): Promise<any> {
    // ... existing fee validation ...

    const [msg, msgErr] = FromAny(request.tx?.msg);
    if (msgErr) return { error: msgErr };

    if (msg?.typeUrl?.endsWith('MessageSend')) {
        return contract.CheckMessageSend(msg);
    } else if (msg?.typeUrl?.endsWith('MessageReward')) {
        return contract.CheckMessageReward(msg);  // Add this
    }
    return { error: ErrInvalidMessageCast() };
}
```

## Step 3: Add DeliverTx Execution

DeliverTx reads state, applies the transaction, and writes state changes.

The "reward" transaction mints new tokens to a recipient.

```typescript
// In contract.ts ContractAsync - add to DeliverTx
static async DeliverTx(contract: Contract, request: any): Promise<any> {
    const [msg, err] = FromAny(request.tx?.msg);
    if (err) return { error: err };

    if (msg?.typeUrl?.endsWith('MessageSend')) {
        return ContractAsync.DeliverMessageSend(contract, msg, request.tx?.fee as Long);
    } else if (msg?.typeUrl?.endsWith('MessageReward')) {
        return ContractAsync.DeliverMessageReward(contract, msg, request.tx?.fee as Long);
    }
    return { error: ErrInvalidMessageCast() };
}

// Add execution function
static async DeliverMessageReward(contract: Contract, msg: any, fee: Long | number | undefined): Promise<any> {
    const adminQueryId = Long.fromNumber(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));
    const recipientQueryId = Long.fromNumber(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));
    const feeQueryId = Long.fromNumber(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));

    const adminKey = KeyForAccount(msg.adminAddress);
    const recipientKey = KeyForAccount(msg.recipientAddress);
    const feePoolKey = KeyForFeePool(Long.fromNumber(contract.Config.ChainId));

    // Read state
    const [response, readErr] = await contract.plugin.StateRead(contract, {
        keys: [
            { queryId: adminQueryId, key: adminKey },
            { queryId: recipientQueryId, key: recipientKey },
            { queryId: feeQueryId, key: feePoolKey },
        ],
    });

    if (readErr) return { error: readErr };
    if (response?.error) return { error: response.error };

    // Parse results
    let adminBytes: Uint8Array | null = null;
    let recipientBytes: Uint8Array | null = null;
    let feePoolBytes: Uint8Array | null = null;

    for (const resp of response?.results || []) {
        const qid = resp.queryId as Long;
        if (qid.equals(adminQueryId)) {
            adminBytes = resp.entries?.[0]?.value || null;
        } else if (qid.equals(recipientQueryId)) {
            recipientBytes = resp.entries?.[0]?.value || null;
        } else if (qid.equals(feeQueryId)) {
            feePoolBytes = resp.entries?.[0]?.value || null;
        }
    }

    // Unmarshal
    const [adminRaw] = Unmarshal(adminBytes || new Uint8Array(), types.Account);
    const [recipientRaw] = Unmarshal(recipientBytes || new Uint8Array(), types.Account);
    const [feePoolRaw] = Unmarshal(feePoolBytes || new Uint8Array(), types.Pool);
    const admin = adminRaw as any;
    const recipient = recipientRaw as any;
    const feePool = feePoolRaw as any;

    // Validate admin can pay fee
    const feeAmount = Long.isLong(fee) ? fee : Long.fromNumber(fee as number || 0);
    const adminAmount = Long.isLong(admin?.amount) ? admin.amount : Long.fromNumber(admin?.amount || 0);

    if (adminAmount.lessThan(feeAmount)) {
        return { error: ErrInsufficientFunds() };
    }

    // Apply state changes
    const msgAmount = Long.isLong(msg.amount) ? msg.amount : Long.fromNumber(msg.amount || 0);
    const newAdminAmount = adminAmount.subtract(feeAmount);  // Admin pays fee
    const recipientAmount = Long.isLong(recipient?.amount) ? recipient.amount : Long.fromNumber(recipient?.amount || 0);
    const newRecipientAmount = recipientAmount.add(msgAmount);  // Recipient gets reward (minted)
    const poolAmount = Long.isLong(feePool?.amount) ? feePool.amount : Long.fromNumber(feePool?.amount || 0);
    const newPoolAmount = poolAmount.add(feeAmount);

    const updatedAdmin = types.Account.create({ address: admin?.address, amount: newAdminAmount });
    const updatedRecipient = types.Account.create({ address: recipient?.address || msg.recipientAddress, amount: newRecipientAmount });
    const updatedPool = types.Pool.create({ id: feePool?.id, amount: newPoolAmount });

    const newAdminBytes = types.Account.encode(updatedAdmin).finish();
    const newRecipientBytes = types.Account.encode(updatedRecipient).finish();
    const newFeePoolBytes = types.Pool.encode(updatedPool).finish();

    // Write state
    const [writeResp, writeErr] = await contract.plugin.StateWrite(contract, {
        sets: [
            { key: feePoolKey, value: newFeePoolBytes },
            { key: adminKey, value: newAdminBytes },
            { key: recipientKey, value: newRecipientBytes },
        ],
    });

    if (writeErr) return { error: writeErr };
    if (writeResp?.error) return { error: writeResp.error };

    return {};
}
```

## Adding New State Keys

If your transaction needs new state storage:

```typescript
// In contract.ts
const validatorPrefix = Buffer.from([3]);
export function KeyForValidator(addr: Uint8Array): Uint8Array {
    return JoinLenPrefix(validatorPrefix, Buffer.from(addr));
}

const delegationPrefix = Buffer.from([4]);
export function KeyForDelegation(delegator: Uint8Array, validator: Uint8Array): Uint8Array {
    return JoinLenPrefix(delegationPrefix, Buffer.from(delegator), Buffer.from(validator));
}
```

## Build

```bash
npm run build          # Compile TypeScript
npm run build:proto    # Regenerate protobuf bindings
npm run validate       # Run lint + type-check + tests
```
