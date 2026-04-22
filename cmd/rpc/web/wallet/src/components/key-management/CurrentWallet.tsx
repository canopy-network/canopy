import React, { useEffect, useMemo, useState } from "react";
import { motion } from "framer-motion";
import {
  Copy,
  Download,
  Key,
  AlertTriangle,
  Eye,
  EyeOff,
  Pencil,
  Trash2,
} from "lucide-react";
import { Button } from "@/components/ui/Button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/Select";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { useToast } from "@/toast/ToastContext";
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useDSFetcher } from "@/core/dsFetch";
import { useDS } from "@/core/useDs";
import { downloadJson } from "@/helpers/download";
import { useQueryClient } from "@tanstack/react-query";

export const CurrentWallet = ({ embedded = false }: { embedded?: boolean }): JSX.Element => {
  const { accounts, selectedAccount, switchAccount } = useAccounts();

  const [privateKey, setPrivateKey] = useState("");
  const [privateKeyVisible, setPrivateKeyVisible] = useState(false);
  const [showPasswordModal, setShowPasswordModal] = useState(false);
  const [password, setPassword] = useState("");
  const [passwordError, setPasswordError] = useState("");
  const [isFetchingKey, setIsFetchingKey] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleteConfirmation, setDeleteConfirmation] = useState("");
  const [isDeleting, setIsDeleting] = useState(false);
  const [isRenameOpen, setIsRenameOpen] = useState(false);
  const [renameNickname, setRenameNickname] = useState("");
  const [isRenaming, setIsRenaming] = useState(false);
  const { copyToClipboard } = useCopyToClipboard();
  const toast = useToast();
  const dsFetch = useDSFetcher();
  const queryClient = useQueryClient();
  const { data: keystore } = useDS("keystore", {});

  const panelVariants = {
    hidden: { opacity: 0, y: 20 },
    visible: {
      opacity: 1,
      y: 0,
      transition: { duration: 0.4 },
    },
  };

  const truncateMiddle = (value: string, start = 12, end = 10) => {
    if (!value) return "";
    if (value.length <= start + end + 3) return value;
    return `${value.slice(0, start)}...${value.slice(-end)}`;
  };

  const selectedKeyEntry = useMemo(() => {
    if (!keystore || !selectedAccount) return null;
    return keystore.addressMap?.[selectedAccount.address] ?? null;
  }, [keystore, selectedAccount]);

  useEffect(() => {
    setPrivateKey("");
    setPrivateKeyVisible(false);
    setShowPasswordModal(false);
    setPassword("");
    setPasswordError("");
    setIsRenameOpen(false);
    setRenameNickname(selectedKeyEntry?.keyNickname || selectedAccount?.nickname || "");
  }, [selectedAccount?.id, selectedAccount?.nickname, selectedKeyEntry?.keyNickname]);

  const invalidateKeystore = async () => {
    const invalidate = () =>
      queryClient.invalidateQueries({
        predicate: (query) =>
          Array.isArray(query.queryKey) &&
          query.queryKey[0] === "ds" &&
          query.queryKey[2] === "keystore",
      });

    await invalidate();
    setTimeout(() => {
      void invalidate();
    }, 500);
  };

  const handleDownloadKeyfile = () => {
    if (!selectedAccount) {
      toast.error({
        title: "No Account Selected",
        description: "Please select an active account first",
      });
      return;
    }

    if (!keystore) {
      toast.error({
        title: "Keyfile Unavailable",
        description: "Keystore data is not ready yet.",
      });
      return;
    }

    if (!selectedKeyEntry) {
      toast.error({
        title: "Keyfile Unavailable",
        description: "Selected wallet data is missing in the keystore.",
      });
      return;
    }

    const nickname = selectedKeyEntry.keyNickname || selectedAccount.nickname;
    const nicknameValue =
      (keystore.nicknameMap ?? {})[nickname] ?? selectedKeyEntry.keyAddress;
    const keyfilePayload = {
      addressMap: {
        [selectedKeyEntry.keyAddress]: selectedKeyEntry,
      },
      nicknameMap: {
        [nickname]: nicknameValue,
      },
    };

    downloadJson(keyfilePayload, `keyfile-${nickname}`);
    toast.success({
      title: "Download Started",
      description: "Your keyfile JSON is downloading.",
    });
  };

  const handleRevealPrivateKeys = () => {
    if (!selectedAccount) {
      toast.error({
        title: "No Account Selected",
        description: "Please select an active account first",
      });
      return;
    }

    if (privateKeyVisible) {
      setPrivateKey("");
      setPrivateKeyVisible(false);
      toast.success({
        title: "Private Key Hidden",
        description: "Your private key is hidden again.",
        icon: <EyeOff className="h-5 w-5" />,
      });
      return;
    }

    setPassword("");
    setPasswordError("");
    setShowPasswordModal(true);
  };

  const handleFetchPrivateKey = async () => {
    if (!selectedAccount) return;
    if (!password) {
      setPasswordError("Password is required.");
      return;
    }

    setIsFetchingKey(true);
    setPasswordError("");

    try {
      const response = await dsFetch("keystoreGet", {
        address: selectedKeyEntry?.keyAddress ?? selectedAccount.address,
        password,
        nickname: selectedKeyEntry?.keyNickname,
      });
      const extracted =
        (response as any)?.privateKey ??
        (response as any)?.private_key ??
        (response as any)?.PrivateKey ??
        (response as any)?.Private_key ??
        (typeof response === "string" ? response.replace(/"/g, "") : "");

      if (!extracted) {
        throw new Error("Private key not found.");
      }

      setPrivateKey(extracted);
      setPrivateKeyVisible(true);
      setShowPasswordModal(false);
      setPassword("");
      toast.success({
        title: "Private Key Revealed",
        description: "Be careful! Your private key is now visible.",
        icon: <Eye className="h-5 w-5" />,
      });
    } catch (error) {
      setPasswordError("Unable to unlock with that password.");
      toast.error({
        title: "Unlock Failed",
        description: String(error),
      });
    } finally {
      setIsFetchingKey(false);
    }
  };

  const handleDeleteAccount = () => {
    if (!selectedAccount) {
      toast.error({
        title: "No Account Selected",
        description: "Please select an account to delete",
      });
      return;
    }

    setDeleteConfirmation("");
    setShowDeleteModal(true);
  };

  const handleRenameAccount = async () => {
    if (!selectedAccount || !selectedKeyEntry) return;

    const nextNickname = renameNickname.trim();
    const currentNickname = selectedKeyEntry.keyNickname || selectedAccount.nickname;

    if (!nextNickname) {
      toast.error({
        title: "Missing wallet name",
        description: "Please enter a nickname.",
      });
      return;
    }

    if (nextNickname === currentNickname) {
      setIsRenameOpen(false);
      return;
    }

    setIsRenaming(true);
    try {
      await dsFetch("keystoreImport", {
        nickname: nextNickname,
        address: selectedKeyEntry.keyAddress,
        publicKey: selectedKeyEntry.publicKey,
        salt: selectedKeyEntry.salt,
        encrypted: selectedKeyEntry.encrypted,
        keyAddress: selectedKeyEntry.keyAddress,
      });

      await invalidateKeystore();

      toast.success({
        title: "Nickname updated",
        description: `Wallet renamed to "${nextNickname}".`,
      });
      setIsRenameOpen(false);
    } catch (error) {
      toast.error({
        title: "Rename failed",
        description: error instanceof Error ? error.message : String(error),
      });
    } finally {
      setIsRenaming(false);
    }
  };

  const handleConfirmDelete = async () => {
    if (!selectedAccount) return;

    const nickname = selectedKeyEntry?.keyNickname || selectedAccount.nickname;
    if (deleteConfirmation !== nickname) {
      toast.error({
        title: "Confirmation Failed",
        description: `Please type "${nickname}" to confirm deletion`,
      });
      return;
    }

    setIsDeleting(true);

    try {
      await dsFetch("keystoreDelete", {
        nickname: nickname,
      });

      await invalidateKeystore();

      toast.success({
        title: "Account Deleted",
        description: `Account "${nickname}" has been permanently deleted.`,
      });

      setShowDeleteModal(false);
      setDeleteConfirmation("");

      // Switch to another account
      const otherAccounts = accounts.filter((acc) => acc.id !== selectedAccount.id);
      if (otherAccounts.length > 0) {
        setTimeout(() => {
          switchAccount(otherAccounts[0].id);
        }, 500);
      } else {
        switchAccount(null);
      }
    } catch (error) {
      toast.error({
        title: "Delete Failed",
        description: error instanceof Error ? error.message : String(error),
      });
    } finally {
      setIsDeleting(false);
    }
  };

  const content = (
    <>
      <div className="space-y-5">
        <div>
          <label className="block text-sm font-medium text-foreground/80 mb-2">
            Wallet Name
          </label>
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <Select
                value={selectedAccount?.id || ""}
                onValueChange={switchAccount}
              >
                <SelectTrigger className="w-full bg-muted border-border text-foreground h-11 rounded-lg focus:ring-2 focus:ring-primary/35">
                  <SelectValue placeholder="Select wallet" />
                </SelectTrigger>
                <SelectContent className="bg-muted border-border">
                  {accounts.map((account) => (
                    <SelectItem
                      key={account.id}
                      value={account.id}
                      className="text-foreground"
                    >
                      {account.nickname}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-11 px-3"
                onClick={() => {
                  setRenameNickname(selectedKeyEntry?.keyNickname || selectedAccount?.nickname || "");
                  setIsRenameOpen((value) => !value);
                }}
                disabled={!selectedAccount || !selectedKeyEntry}
              >
                <Pencil className="h-4 w-4" />
                Rename
              </Button>
            </div>

            {isRenameOpen && (
              <div className="flex flex-col gap-2 rounded-lg border border-border bg-muted/40 p-3 sm:flex-row">
                <input
                  type="text"
                  value={renameNickname}
                  onChange={(e) => setRenameNickname(e.target.value)}
                  placeholder="Wallet nickname"
                  className="w-full rounded-lg border border-border bg-muted px-3 py-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/25"
                  autoFocus
                />
                <div className="flex gap-2">
                  <Button
                    type="button"
                    size="sm"
                    className="h-10"
                    onClick={handleRenameAccount}
                    disabled={isRenaming}
                  >
                    {isRenaming ? "Saving..." : "Save"}
                  </Button>
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    className="h-10"
                    onClick={() => {
                      setIsRenameOpen(false);
                      setRenameNickname(selectedKeyEntry?.keyNickname || selectedAccount?.nickname || "");
                    }}
                    disabled={isRenaming}
                  >
                    Cancel
                  </Button>
                </div>
              </div>
            )}
          </div>
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label className="block text-sm font-medium text-foreground/80 mb-2">
              Wallet Address
            </label>
            <div className="relative">
              <div
                className="min-w-0 rounded-lg border border-border bg-muted px-3 py-2.5 pr-11 text-sm text-foreground"
                title={selectedAccount?.address || ""}
              >
                <span className="block truncate font-mono">
                  {truncateMiddle(selectedAccount?.address || "", 12, 10)}
                </span>
              </div>
              <button
                onClick={() =>
                  copyToClipboard(
                    selectedAccount?.address || "",
                    "Wallet address",
                  )
                }
                className="absolute right-2 top-1/2 -translate-y-1/2 rounded-md p-1.5 text-white/70 transition-colors hover:text-white"
              >
                <Copy className="w-4 h-4" />
              </button>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-foreground/80 mb-2">
              Public Key
            </label>
            <div className="relative">
              <div
                className="min-w-0 rounded-lg border border-border bg-muted px-3 py-2.5 pr-11 text-sm text-foreground"
                title={selectedAccount?.publicKey || ""}
              >
                <span className="block truncate font-mono">
                  {truncateMiddle(selectedAccount?.publicKey || "", 12, 10)}
                </span>
              </div>
              <button
                onClick={() =>
                  copyToClipboard(selectedAccount?.publicKey || "", "Public key")
                }
                className="absolute right-2 top-1/2 -translate-y-1/2 rounded-md p-1.5 text-white/70 transition-colors hover:text-white"
              >
                <Copy className="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-foreground/80 mb-2">
            Private Key
          </label>
          <div className="relative flex items-center justify-between gap-2">
            <input
              type={privateKeyVisible ? "text" : "password"}
              value={privateKeyVisible ? privateKey : ""}
              readOnly
              placeholder="Hidden until unlocked"
              className="w-full bg-muted border border-border rounded-lg px-3 py-2.5 text-foreground pr-10 placeholder:text-muted-foreground"
            />
            {privateKeyVisible && (
              <button
                onClick={() => copyToClipboard(privateKey, "Private key")}
                className="rounded-lg border border-[#272729] bg-[#0f0f0f] px-3 py-2.5 text-white/70 transition-colors hover:bg-[#272729] hover:text-white"
              >
                <Copy className="w-4 h-4" />
              </button>
            )}
            <button
              onClick={handleRevealPrivateKeys}
              className="rounded-lg border border-[#272729] bg-[#0f0f0f] px-3 py-2 text-white/70 transition-colors hover:bg-[#272729] hover:text-white"
            >
              {privateKeyVisible ? (
                <EyeOff className="text-foreground w-4 h-4" />
              ) : (
                <Eye className="text-foreground w-4 h-4" />
              )}
            </button>
          </div>
        </div>

        <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
          <Button
            onClick={handleDownloadKeyfile}
            variant="default"
            className="h-11 w-full"
          >
            <Download className="h-4 w-4" />
            Download Keyfile
          </Button>
          <Button
            onClick={handleRevealPrivateKeys}
            variant="secondary"
            className="h-11 w-full"
          >
            <Key className="h-4 w-4" />
            {privateKeyVisible ? "Hide Private Key" : "Reveal Private Key"}
          </Button>
          <Button
            onClick={handleDeleteAccount}
            variant="secondary"
            className="h-11 w-full border-[#ff1845]/30 bg-[#ff1845]/10 text-[#ff6b84] shadow-none hover:border-[#ff1845]/40 hover:bg-[#ff1845]/14 hover:text-[#ff7f96]"
            disabled={!selectedAccount}
          >
            <Trash2 className="h-4 w-4" />
            Delete Account
          </Button>
        </div>

        <div className="rounded-lg border border-[#272729] bg-[#0f0f0f] p-4">
          <div className="flex items-start gap-3">
            <AlertTriangle className="mt-0.5 h-5 w-5 text-white/60" />
            <div>
              <h4 className="mb-1 font-medium text-foreground">
                Security Warning
              </h4>
              <p className="text-sm text-muted-foreground">
                Never share your private keys. Anyone with access to them can
                control your funds.
              </p>
            </div>
          </div>
        </div>
      </div>

      {showPasswordModal && (
        <div className="fixed inset-0 z-[250] flex items-center justify-center bg-[#0f0f0f]/80 backdrop-blur-md p-3 sm:p-4">
          <div className="w-full max-w-sm max-h-[calc(100dvh-1.5rem)] sm:max-h-[calc(100dvh-2rem)] bg-[#171717] border border-[#272729] rounded-2xl p-4 sm:p-5 shadow-[0_24px_72px_rgba(0,0,0,0.55)] overflow-y-auto">
            <h3 className="text-lg text-foreground font-semibold mb-2">
              Unlock Private Key
            </h3>
            <p className="text-sm text-muted-foreground mb-4">
              Enter your wallet password to reveal the private key.
            </p>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Password"
              className="w-full bg-[#0f0f0f] text-foreground border border-[#272729] rounded-lg px-3 py-2.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#35cd48]/25"
            />
            {passwordError && (
              <div className="text-sm text-[#ff1845] mt-2">{passwordError}</div>
            )}
            <div className="flex justify-end gap-2 mt-4">
              <button
                onClick={() => setShowPasswordModal(false)}
                className="px-4 py-2 rounded-lg border border-[#272729] bg-[#0f0f0f] text-white hover:bg-[#272729]"
                disabled={isFetchingKey}
              >
                Cancel
              </button>
              <button
                onClick={handleFetchPrivateKey}
                className="px-4 py-2 rounded-lg bg-[#35cd48] text-[#0f0f0f] hover:bg-[#35cd48]/90"
                disabled={isFetchingKey}
              >
                {isFetchingKey ? "Unlocking..." : "Unlock"}
              </button>
            </div>
          </div>
        </div>
      )}

      {showDeleteModal && (
        <div className="fixed inset-0 z-[250] flex items-center justify-center bg-[#0f0f0f]/80 backdrop-blur-md p-3 sm:p-4">
          <div className="w-full max-w-md max-h-[calc(100dvh-1.5rem)] sm:max-h-[calc(100dvh-2rem)] bg-[#171717] border border-[#ff1845]/35 rounded-2xl p-4 sm:p-6 shadow-[0_24px_72px_rgba(0,0,0,0.55)] overflow-y-auto">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-3 bg-[#ff1845]/12 rounded-full">
                <AlertTriangle className="w-6 h-6 text-[#ff1845]" />
              </div>
              <h3 className="text-xl text-foreground font-semibold">
                Delete Account
              </h3>
            </div>

            <div className="bg-[#ff1845]/10 border border-[#ff1845]/25 rounded-lg p-4 mb-4">
              <p className="text-[#ff1845] text-sm font-medium mb-2">
                This action is permanent and irreversible
              </p>
              <p className="text-[#ff1845] text-sm">
                Make sure you have backed up your private key before deleting this account.
                You will lose access to all funds if you haven't saved your private key.
              </p>
            </div>

            <p className="text-sm text-muted-foreground mb-4">
              Type <span className="font-semibold text-foreground">
                {selectedKeyEntry?.keyNickname || selectedAccount?.nickname}
              </span> to confirm deletion:
            </p>

            <input
              type="text"
              value={deleteConfirmation}
              onChange={(e) => setDeleteConfirmation(e.target.value)}
              placeholder="Type wallet name to confirm"
              className="w-full bg-[#0f0f0f] text-foreground border border-[#272729] rounded-lg px-3 py-2.5 mb-4 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#ff1845]/25"
              autoFocus
            />

            <div className="flex justify-end gap-2">
              <button
                onClick={() => {
                  setShowDeleteModal(false);
                  setDeleteConfirmation("");
                }}
                className="px-4 py-2 rounded-lg border border-[#272729] bg-[#0f0f0f] text-white hover:bg-[#272729]"
                disabled={isDeleting}
              >
                Cancel
              </button>
              <button
                onClick={handleConfirmDelete}
                className="px-4 py-2 rounded-lg bg-[#ff1845] text-white hover:bg-[#ff1845]/90 disabled:opacity-50 disabled:cursor-not-allowed"
                disabled={isDeleting || deleteConfirmation !== (selectedKeyEntry?.keyNickname || selectedAccount?.nickname)}
              >
                {isDeleting ? "Deleting..." : "Delete Permanently"}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );

  if (embedded) {
    return <div className="w-full">{content}</div>;
  }

  return (
    <motion.div
      variants={panelVariants}
      className="bg-card rounded-2xl p-6 border border-border/80 shadow-[0_14px_34px_rgba(0,0,0,0.2)]"
    >
      {content}
    </motion.div>
  );
};
