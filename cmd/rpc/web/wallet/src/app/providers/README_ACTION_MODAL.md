# Action Modal Integration

## Overview

The `ActionModalProvider` provides a global modal system for running actions (like send, stake, etc.) from anywhere in the application.

## Setup

The provider is already integrated in `src/app/App.tsx`:

```tsx
<ActionModalProvider>
  <Theme>
    <RouterProvider router={router}/>
  </Theme>
</ActionModalProvider>
```

## Usage

### 1. Import the hook

```tsx
import { useActionModal } from '@/app/providers/ActionModalProvider';
```

### 2. Use in your component

```tsx
export const YourComponent = () => {
  const { openAction } = useActionModal();

  const handleSend = () => {
    openAction('send', {
      onFinish: () => {
        console.log('Send completed!');
        // Refresh data, show toast, etc.
      },
      onClose: () => {
        console.log('Modal closed');
      }
    });
  };

  return (
    <button onClick={handleSend}>
      Send Tokens
    </button>
  );
};
```

### 3. Available Actions

Actions are defined in `public/plugin/canopy/manifest.json`. Common actions include:

- `send` - Send tokens to another address
- `stake` - Stake tokens
- `unstake` - Unstake tokens
- `editStake` - Edit stake amount
- `receive` - Show receive address

### 4. Setting the Selected Account

Before opening an action, make sure to set the correct account:

```tsx
import { useAccounts } from '@/hooks/useAccounts';
import { useActionModal } from '@/app/providers/ActionModalProvider';

export const AccountList = () => {
  const { accounts, setSelectedAccount } = useAccounts();
  const { openAction } = useActionModal();

  const handleSendFromAccount = (accountAddress: string) => {
    // Find and set the account
    const account = accounts.find(a => a.address === accountAddress);
    if (account && setSelectedAccount) {
      setSelectedAccount(account);
    }

    // Open the send action
    openAction('send', {
      onFinish: () => {
        // Refresh balances, show success message, etc.
      }
    });
  };

  return (
    <div>
      {accounts.map(account => (
        <button key={account.address} onClick={() => handleSendFromAccount(account.address)}>
          Send from {account.nickname}
        </button>
      ))}
    </div>
  );
};
```

## Example: Complete Integration

See `src/app/pages/Accounts.tsx` for a complete example of how to:

1. Import and use the hook
2. Set the selected account before opening an action
3. Handle callbacks (onFinish, onClose)
4. Integrate with existing UI components

## API Reference

### `useActionModal()`

Returns an object with:

- `openAction(actionId: string, options?: ActionModalOptions)` - Opens an action modal
- `closeAction()` - Closes the current action modal
- `isOpen: boolean` - Whether a modal is currently open
- `currentActionId: string | null` - The ID of the currently open action

### `ActionModalOptions`

```typescript
interface ActionModalOptions {
  onFinish?: () => void;  // Called when action completes successfully
  onClose?: () => void;   // Called when modal is closed (any reason)
}
```

## Styling

The modal uses the following classes from your theme:

- `bg-bg-secondary` - Modal background
- `bg-bg-tertiary` - Button backgrounds
- `border-bg-accent` - Borders
- `text-text-muted` - Icon colors

You can customize the modal appearance by editing `src/app/providers/ActionModalProvider.tsx`.
