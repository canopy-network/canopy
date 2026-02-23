import React, { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { useDSFetcher } from "@/core/dsFetch";
import { useConfig } from "@/app/providers/ConfigProvider";
import { useAccounts } from "@/app/providers/AccountsProvider";
import { AlertModal } from "./AlertModal";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Label } from "@/components/ui/Label";

interface PauseUnpauseModalProps {
  isOpen: boolean;
  onClose: () => void;
  validatorAddress: string;
  validatorNickname?: string;
  action: "pause" | "unpause";
  allValidators?: Array<{
    address: string;
    nickname?: string;
  }>;
  isBulkAction?: boolean;
}

export const PauseUnpauseModal: React.FC<PauseUnpauseModalProps> = ({
  isOpen,
  onClose,
  validatorAddress,
  validatorNickname,
  action,
  allValidators = [],
  isBulkAction = false,
}) => {
  const { accounts } = useAccounts();
  const { chain } = useConfig();
  const [formData, setFormData] = useState({
    account: validatorNickname || accounts[0]?.nickname || "",
    signer: validatorNickname || accounts[0]?.nickname || "",
    memo: "",
    fee: 0.01,
    password: "",
  });

  // Update form data when validator changes
  React.useEffect(() => {
    if (validatorNickname) {
      setFormData((prev) => ({
        ...prev,
        account: validatorNickname,
        signer: validatorNickname,
      }));
    }
  }, [validatorNickname]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [selectedValidators, setSelectedValidators] = useState<string[]>([]);
  const [selectAll, setSelectAll] = useState(false);
  const [alertModal, setAlertModal] = useState<{
    isOpen: boolean;
    title: string;
    message: string;
    type: "success" | "error" | "warning" | "info";
  }>({
    isOpen: false,
    title: "",
    message: "",
    type: "info",
  });

  const handleInputChange = (field: string, value: string | number) => {
    setFormData((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  const handleValidatorSelect = (validatorAddress: string) => {
    setSelectedValidators((prev) => {
      if (prev.includes(validatorAddress)) {
        return prev.filter((addr) => addr !== validatorAddress);
      } else {
        return [...prev, validatorAddress];
      }
    });
  };

  const handleSelectAll = () => {
    if (selectAll) {
      setSelectedValidators([]);
      setSelectAll(false);
    } else {
      const allAddresses = sortedValidators.map((v) => v.address);
      setSelectedValidators(allAddresses);
      setSelectAll(true);
    }
  };

  // Sort validators by node number
  const sortedValidators = React.useMemo(() => {
    if (!allValidators || allValidators.length === 0) return [];

    return [...allValidators].sort((a, b) => {
      // Extract node number from nickname (e.g., "node_1" -> 1, "node_2" -> 2)
      const getNodeNumber = (validator: any) => {
        const nickname = validator.nickname || "";
        const match = nickname.match(/node_(\d+)/);
        return match ? parseInt(match[1]) : 999; // Put nodes without numbers at the end
      };

      return getNodeNumber(a) - getNodeNumber(b);
    });
  }, [allValidators]);

  // Initialize selected validators when modal opens
  React.useEffect(() => {
    if (isBulkAction && sortedValidators.length > 0) {
      setSelectedValidators(sortedValidators.map((v) => v.address));
      setSelectAll(true);
    } else {
      setSelectedValidators([validatorAddress]);
      setSelectAll(false);
    }
  }, [isBulkAction, sortedValidators, validatorAddress]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);

    try {
      // Find the account by nickname
      const account = accounts.find(
        (acc: any) => acc.nickname === formData.account,
      );
      const signer = accounts.find(
        (acc: any) => acc.nickname === formData.signer,
      );

      if (!account || !signer) {
        setAlertModal({
          isOpen: true,
          title: "Account Not Found",
          message:
            "The selected account or signer was not found. Please check your selection.",
          type: "error",
        });
        return;
      }

      if (selectedValidators.length === 0) {
        setAlertModal({
          isOpen: true,
          title: "No Validators Selected",
          message: "Please select at least one validator to proceed.",
          type: "warning",
        });
        return;
      }

      const feeInMicroUnits = formData.fee * 1000000; // Convert to micro-units

      // Process each selected validator
      const promises = selectedValidators.map(async (validatorAddr) => {
        // Note: These transaction endpoints would need to be added to chain.json DS config
        // For now, using direct admin endpoint calls with DS pattern structure
        const txEndpoint = action === "pause" ? "tx-pause" : "tx-unpause";

        try {
          // This would ideally use DS pattern once tx endpoints are added to chain.json
          const response = await fetch(
            `${chain?.rpc?.admin}/v1/admin/${txEndpoint}`,
            {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({
                address: validatorAddr,
                pubKey: "",
                netAddress: "",
                committees: "",
                amount: 0,
                delegate: false,
                earlyWithdrawal: false,
                output: "",
                signer: signer.address,
                memo: formData.memo,
                fee: feeInMicroUnits,
                submit: true,
                password: formData.password,
              }),
            },
          );

          if (!response.ok) {
            throw new Error(`Transaction failed: ${response.status}`);
          }

          return await response.json();
        } catch (error) {
          console.error(`Error executing ${action} transaction:`, error);
          throw error;
        }
      });

      await Promise.all(promises);

      setSuccess(true);
      setTimeout(() => {
        onClose();
        setSuccess(false);
        setFormData({
          account: validatorNickname || accounts[0]?.nickname || "",
          signer: validatorNickname || accounts[0]?.nickname || "",
          memo: "",
          fee: 0.01,
          password: "",
        });
        setSelectedValidators([]);
        setSelectAll(false);
      }, 2000);
    } catch (err) {
      setAlertModal({
        isOpen: true,
        title: "Transaction Failed",
        message:
          err instanceof Error
            ? err.message
            : "An unexpected error occurred while processing the transaction.",
        type: "error",
      });
    } finally {
      setIsLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <AnimatePresence>
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
        onClick={onClose}
      >
        <motion.div
          initial={{ scale: 0.9, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          exit={{ scale: 0.9, opacity: 0 }}
          className="bg-card rounded-xl border border-border p-6 w-full max-w-md"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-xl font-semibold text-foreground capitalize">
              {action} Validator
            </h2>
            <Button
              type="button"
              onClick={onClose}
              variant="clear2"
              size="icon"
              className="text-muted-foreground hover:text-foreground"
            >
              <i className="fa-solid fa-times text-lg"></i>
            </Button>
          </div>

          {success ? (
            <motion.div
              initial={{ scale: 0.9, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              className="text-center py-8"
            >
              <div className="w-16 h-16 bg-green-500/20 rounded-full flex items-center justify-center mx-auto mb-4">
                <i className="fa-solid fa-check text-green-400 text-2xl"></i>
              </div>
              <h3 className="text-lg font-semibold text-foreground mb-2">
                Transaction Successful!
              </h3>
              <p className="text-muted-foreground">
                Validator {action}d successfully
              </p>
            </motion.div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              {/* Validator Selection */}
              {isBulkAction && sortedValidators.length > 0 && (
                <div className="bg-muted/30 rounded-lg p-4 border border-border">
                  <div className="flex items-center justify-between mb-3">
                    <label className="text-sm font-medium text-foreground">
                      Select Validators
                    </label>
                    <span className="text-xs text-muted-foreground bg-accent px-2 py-1 rounded-full">
                      {selectedValidators.length} of {sortedValidators.length}{" "}
                      selected
                    </span>
                  </div>

                  {/* Simple Select All */}
                  <div className="mb-3">
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={selectAll}
                        onChange={handleSelectAll}
                        className="w-4 h-4 text-primary bg-card border-border rounded focus:ring-primary focus:ring-2"
                      />
                      <span className="text-sm text-foreground font-medium">
                        Select All ({sortedValidators.length} validators)
                      </span>
                    </label>
                  </div>

                  {/* Simple Validator List */}
                  <div className="space-y-2 max-h-32 overflow-y-auto">
                    {sortedValidators.map((validator) => {
                      const matchingAccount = accounts?.find(
                        (acc: any) => acc.address === validator.address,
                      );
                      const displayName =
                        matchingAccount?.nickname ||
                        validator.nickname ||
                        `Node ${validator.address.substring(0, 8)}`;
                      const isSelected = selectedValidators.includes(
                        validator.address,
                      );

                      return (
                        <label
                          key={validator.address}
                          className="flex items-center gap-2 cursor-pointer p-1 rounded hover:bg-accent/30 transition-colors"
                        >
                          <input
                            type="checkbox"
                            checked={isSelected}
                            onChange={() =>
                              handleValidatorSelect(validator.address)
                            }
                            className="w-4 h-4 text-primary bg-card border-border rounded focus:ring-primary focus:ring-2"
                          />
                          <span className="text-sm text-foreground">
                            {displayName}
                          </span>
                          <span className="text-xs text-muted-foreground font-mono">
                            ({validator.address.substring(0, 8)}...
                            {validator.address.substring(
                              validator.address.length - 4,
                            )}
                            )
                          </span>
                        </label>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* Form Fields */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* Account */}
                <div>
                  <Label className="block text-sm font-medium text-foreground mb-2">
                    <i className="fa-solid fa-user mr-2 text-primary"></i>
                    Account
                  </Label>
                  <select
                    value={formData.account}
                    onChange={(e) =>
                      handleInputChange("account", e.target.value)
                    }
                    className="w-full px-3 py-2 bg-muted border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors"
                    required
                  >
                    {accounts.map((account: any) => (
                      <option key={account.address} value={account.nickname}>
                        {account.nickname}
                      </option>
                    ))}
                  </select>
                </div>

                {/* Signer */}
                <div>
                  <Label className="block text-sm font-medium text-foreground mb-2">
                    <i className="fa-solid fa-signature mr-2 text-primary"></i>
                    Signer
                  </Label>
                  <select
                    value={formData.signer}
                    onChange={(e) =>
                      handleInputChange("signer", e.target.value)
                    }
                    className="w-full px-3 py-2 bg-muted border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors"
                    required
                  >
                    {accounts.map((account: any) => (
                      <option key={account.address} value={account.nickname}>
                        {account.nickname}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              {/* Memo */}
              <div>
                <Label className="block text-sm font-medium text-foreground mb-2">
                  <i className="fa-solid fa-sticky-note mr-2 text-primary"></i>
                  Memo
                </Label>
                <Input
                  type="text"
                  value={formData.memo}
                  onChange={(e) => handleInputChange("memo", e.target.value)}
                  placeholder="Optional note attached with the transaction"
                  className="w-full px-3 py-2 bg-muted border border-border rounded-lg text-foreground"
                  maxLength={200}
                />
                <p className="text-xs text-muted-foreground mt-1">
                  {formData.memo.length}/200 characters
                </p>
              </div>

              {/* Transaction Fee */}
              <div>
                <Label className="block text-sm font-medium text-foreground mb-2">
                  <i className="fa-solid fa-coins mr-2 text-primary"></i>
                  Transaction Fee
                </Label>
                <div className="relative">
                  <Input
                    type="number"
                    value={formData.fee}
                    onChange={(e) =>
                      handleInputChange("fee", parseFloat(e.target.value) || 0)
                    }
                    step="0.001"
                    min="0"
                    className="w-full px-3 py-2 pr-12 bg-muted border border-border rounded-lg text-foreground"
                    required
                  />
                  <div className="absolute right-3 top-1/2 transform -translate-y-1/2">
                    <span className="text-xs text-muted-foreground font-medium">
                      CNPY
                    </span>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground mt-1">
                  Recommended: 0.01 CNPY
                </p>
              </div>

              {/* Password */}
              <div>
                <Label className="block text-sm font-medium text-foreground mb-2">
                  <i className="fa-solid fa-lock mr-2 text-primary"></i>
                  Password
                </Label>
                <Input
                  type="password"
                  value={formData.password}
                  onChange={(e) =>
                    handleInputChange("password", e.target.value)
                  }
                  placeholder="Enter your key password"
                  className="w-full px-3 py-2 bg-muted border border-border rounded-lg text-foreground"
                  required
                />
              </div>

              <div className="flex gap-3 pt-4">
                <Button
                  type="submit"
                  disabled={isLoading}
                  className="flex-1 text-muted font-medium py-3 px-4 flex items-center justify-center gap-2"
                >
                  {isLoading ? (
                    <>
                      <i className="fa-solid fa-spinner fa-spin"></i>
                      Processing...
                    </>
                  ) : (
                    <>
                      <i className="fa-solid fa-play"></i>
                      Generate Transaction
                    </>
                  )}
                </Button>
              </div>
            </form>
          )}
        </motion.div>
      </motion.div>

      {/* Alert Modal */}
      <AlertModal
        isOpen={alertModal.isOpen}
        onClose={() => setAlertModal((prev) => ({ ...prev, isOpen: false }))}
        title={alertModal.title}
        message={alertModal.message}
        type={alertModal.type}
      />
    </AnimatePresence>
  );
};
