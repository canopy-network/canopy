import React, { useEffect, useMemo, useState } from "react";
import { motion } from "framer-motion";
import {
  Copy,
  Download,
  Key,
  AlertTriangle,
  Shield,
  Eye,
  EyeOff,
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

export const CurrentWallet = (): JSX.Element => {
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
  }, [selectedAccount?.id]);

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

    if (accounts.length === 1) {
      toast.error({
        title: "Cannot Delete",
        description: "You must have at least one account",
      });
      return;
    }

    setDeleteConfirmation("");
    setShowDeleteModal(true);
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

      // Invalidate keystore cache
      await queryClient.invalidateQueries({ queryKey: ["ds", "keystore"] });

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

  return (
    <motion.div
      variants={panelVariants}
      className="bg-bg-secondary rounded-lg p-6 border border-bg-accent"
    >
      <div className="flex items-center justify-between gap-2 mb-6">
        <h2 className="text-xl font-bold text-white">Current Wallet</h2>
        <Shield className="text-primary w-6 h-6" />
      </div>

      <div className="space-y-5">
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            Wallet Name
          </label>
          <Select
            value={selectedAccount?.id || ""}
            onValueChange={switchAccount}
          >
            <SelectTrigger className="w-full bg-bg-tertiary border-bg-accent text-white h-11 rounded-lg">
              <SelectValue placeholder="Select wallet" />
            </SelectTrigger>
            <SelectContent className="bg-bg-tertiary border-bg-accent">
              {accounts.map((account) => (
                <SelectItem
                  key={account.id}
                  value={account.id}
                  className="text-white"
                >
                  {account.nickname}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            Wallet Address
          </label>
          <div className="relative flex items-center justify-between gap-2">
            <input
              type="text"
              value={selectedAccount?.address || ""}
              readOnly
              className="w-full bg-bg-tertiary border border-bg-accent rounded-lg px-3 py-2.5 text-white pr-10"
            />
            <button
              onClick={() =>
                copyToClipboard(
                  selectedAccount?.address || "",
                  "Wallet address",
                )
              }
              className="text-primary-foreground hover:text-white bg-primary rounded-lg px-3 py-2.5"
            >
              <Copy className="w-4 h-4" />
            </button>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            Public Key
          </label>
          <div className="relative flex items-center justify-between gap-2">
            <input
              type="text"
              value={selectedAccount?.publicKey || ""}
              readOnly
              className="w-full bg-bg-tertiary border border-bg-accent rounded-lg px-3 py-2.5 text-white pr-10"
            />
            <button
              onClick={() =>
                copyToClipboard(selectedAccount?.publicKey || "", "Public key")
              }
              className="text-primary-foreground hover:text-white bg-primary rounded-lg px-3 py-2.5"
            >
              <Copy className="w-4 h-4" />
            </button>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            Private Key
          </label>
          <div className="relative flex items-center justify-between gap-2">
            <input
              type={privateKeyVisible ? "text" : "password"}
              value={privateKeyVisible ? privateKey : ""}
              readOnly
              placeholder="Hidden until unlocked"
              className="w-full bg-bg-tertiary border border-bg-accent rounded-lg px-3 py-2.5 text-white pr-10 placeholder:text-gray-500"
            />
            <button
              onClick={handleRevealPrivateKeys}
              className="hover:text-primary bg-muted rounded-lg px-3 py-2 text-white"
            >
              {privateKeyVisible ? (
                <EyeOff className="text-white w-4 h-4" />
              ) : (
                <Eye className="text-white w-4 h-4" />
              )}
            </button>
          </div>
        </div>

        <div className="flex gap-2 flex-col">
          <Button
            onClick={handleDownloadKeyfile}
            className="bg-primary text-primary-foreground hover:bg-primary/90 flex-1 py-3"
          >
            <Download className="w-4 h-4 mr-2" />
            Download Keyfile
          </Button>
          <Button
            onClick={handleRevealPrivateKeys}
            variant="destructive"
            className="flex-1 py-3"
          >
            <Key className="w-4 h-4 mr-2" />
            {privateKeyVisible ? "Hide Private Key" : "Reveal Private Key"}
          </Button>
          <Button
            onClick={handleDeleteAccount}
            variant="destructive"
            className="flex-1 py-3 bg-red-600 hover:bg-red-700"
            disabled={accounts.length === 1}
          >
            <Trash2 className="w-4 h-4 mr-2" />
            Delete Account
          </Button>
        </div>

        <div className="bg-red-900/20 border border-red-500/30 rounded-lg p-4">
          <div className="flex items-start gap-3">
            <AlertTriangle className="text-red-500 w-5 h-5 mt-0.5" />
            <div>
              <h4 className="text-red-400 font-medium mb-1">
                Security Warning
              </h4>
              <p className="text-red-300 text-sm">
                Never share your private keys. Anyone with access to them can
                control your funds.
              </p>
            </div>
          </div>
        </div>
      </div>

      {showPasswordModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
          <div className="w-full max-w-sm bg-bg-secondary border border-bg-accent rounded-xl p-5">
            <h3 className="text-lg text-white font-semibold mb-2">
              Unlock Private Key
            </h3>
            <p className="text-sm text-gray-400 mb-4">
              Enter your wallet password to reveal the private key.
            </p>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Password"
              className="w-full bg-bg-tertiary text-white border border-bg-accent rounded-lg px-3 py-2.5"
            />
            {passwordError && (
              <div className="text-sm text-red-400 mt-2">{passwordError}</div>
            )}
            <div className="flex justify-end gap-2 mt-4">
              <button
                onClick={() => setShowPasswordModal(false)}
                className="px-4 py-2 rounded-lg bg-bg-tertiary text-white hover:bg-bg-accent"
                disabled={isFetchingKey}
              >
                Cancel
              </button>
              <button
                onClick={handleFetchPrivateKey}
                className="px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90"
                disabled={isFetchingKey}
              >
                {isFetchingKey ? "Unlocking..." : "Unlock"}
              </button>
            </div>
          </div>
        </div>
      )}

      {showDeleteModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
          <div className="w-full max-w-md bg-bg-secondary border border-red-500/50 rounded-xl p-6">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-3 bg-red-500/20 rounded-full">
                <AlertTriangle className="w-6 h-6 text-red-500" />
              </div>
              <h3 className="text-xl text-white font-semibold">
                Delete Account
              </h3>
            </div>

            <div className="bg-red-900/20 border border-red-500/30 rounded-lg p-4 mb-4">
              <p className="text-red-300 text-sm font-medium mb-2">
                ⚠️ This action is permanent and irreversible
              </p>
              <p className="text-red-300 text-sm">
                Make sure you have backed up your private key before deleting this account.
                You will lose access to all funds if you haven't saved your private key.
              </p>
            </div>

            <p className="text-sm text-gray-400 mb-4">
              Type <span className="font-mono font-semibold text-white">
                {selectedKeyEntry?.keyNickname || selectedAccount?.nickname}
              </span> to confirm deletion:
            </p>

            <input
              type="text"
              value={deleteConfirmation}
              onChange={(e) => setDeleteConfirmation(e.target.value)}
              placeholder="Type wallet name to confirm"
              className="w-full bg-bg-tertiary text-white border border-bg-accent rounded-lg px-3 py-2.5 mb-4"
              autoFocus
            />

            <div className="flex justify-end gap-2">
              <button
                onClick={() => {
                  setShowDeleteModal(false);
                  setDeleteConfirmation("");
                }}
                className="px-4 py-2 rounded-lg bg-bg-tertiary text-white hover:bg-bg-accent"
                disabled={isDeleting}
              >
                Cancel
              </button>
              <button
                onClick={handleConfirmDelete}
                className="px-4 py-2 rounded-lg bg-red-600 text-white hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed"
                disabled={isDeleting || deleteConfirmation !== (selectedKeyEntry?.keyNickname || selectedAccount?.nickname)}
              >
                {isDeleting ? "Deleting..." : "Delete Permanently"}
              </button>
            </div>
          </div>
        </div>
      )}
    </motion.div>
  );
};
