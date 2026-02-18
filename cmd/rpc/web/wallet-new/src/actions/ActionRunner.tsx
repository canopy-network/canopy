// ActionRunner.tsx
import React from "react";
import { useConfig } from "@/app/providers/ConfigProvider";
import FormRenderer from "./FormRenderer";
import { useResolvedFees } from "@/core/fees";
import { useSession, attachIdleRenew } from "@/state/session";
import UnlockModal from "../components/UnlockModal";
import useDebouncedValue from "../core/useDebouncedValue";
import {
  getFieldsFromAction,
  normalizeFormForAction,
  buildPayloadFromAction,
} from "@/core/actionForm";
import { useAccounts } from "@/app/providers/AccountsProvider";
import { template, templateBool } from "@/core/templater";
import { resolveToastFromManifest } from "@/toast/manifestRuntime";
import { useToast } from "@/toast/ToastContext";
import {
  genericResultMap,
  pauseValidatorMap,
  unpauseValidatorMap,
} from "@/toast/mappers";
import { LucideIcon } from "@/components/ui/LucideIcon";
import { cx } from "@/ui/cx";
import { motion } from "framer-motion";
import { ToastTemplateOptions } from "@/toast/types";
import { useActionDs } from "./useActionDs";
import { usePopulateController } from "./usePopulateController";

type Stage = "form" | "confirm" | "executing" | "result";

export default function ActionRunner({
  actionId,
  onFinish,
  className,
  prefilledData,
}: {
  actionId: string;
  onFinish?: () => void;
  className?: string;
  prefilledData?: Record<string, any>;
}) {
  const toast = useToast();

  const [formHasErrors, setFormHasErrors] = React.useState(false);
  const [stage, setStage] = React.useState<Stage>("form");
  const [form, setForm] = React.useState<Record<string, any>>(
    prefilledData || {},
  );

  // Reduce debounce time from 250ms to 100ms for better responsiveness
  // especially important for prefilledData and DS-dependent fields
  const debouncedForm = useDebouncedValue(form, 100);
  const [txRes, setTxRes] = React.useState<any>(null);
  const [localDs, setLocalDs] = React.useState<Record<string, any>>({});
  // Track which fields were programmatically prefilled (from prefilledData or modules)
  // These fields should hide paste button even when they have values
  const [programmaticallyPrefilled, setProgrammaticallyPrefilled] = React.useState<Set<string>>(
    new Set(prefilledData ? Object.keys(prefilledData) : []),
  );

  const { manifest, chain, params: globalParams, isLoading } = useConfig();
  const { selectedAccount } = useAccounts?.() ?? { selectedAccount: undefined };
  const session = useSession();

  // Merge global params with prefilledData so templates can access both via {{ params.fieldName }}
  const params = React.useMemo(() => ({
    ...globalParams,
    ...prefilledData,
  }), [globalParams, prefilledData]);

  const action = React.useMemo(
    () => manifest?.actions.find((a) => a.id === actionId),
    [manifest, actionId],
  );

  // NEW: Load action-level DS (replaces per-field DS for better performance)
  const actionDsConfig = React.useMemo(() => (action as any)?.ds, [action]);

  // Build context for DS (without ds itself to avoid circular dependency)
  // Use form (not debounced) for DS context to ensure immediate reactivity with prefilledData
  // The DS hook itself handles debouncing internally where needed
  const dsCtx = React.useMemo(
    () => ({
      form: form,
      chain,
      account: selectedAccount
        ? {
            address: selectedAccount.address,
            nickname: selectedAccount.nickname,
            pubKey: selectedAccount.publicKey,
          }
        : undefined,
      params,
    }),
    [form, chain, selectedAccount, params],
  );

  const { ds: actionDs, isLoading: isDsLoading, fetchStatus: dsFetchStatus } = useActionDs(
    actionDsConfig,
    dsCtx,
    actionId,
    selectedAccount?.address,
  );

  // Extract critical DS keys from manifest (DS that must load before showing form)
  const criticalDsKeys = React.useMemo(() => {
    const dsOptions = actionDsConfig?.__options || {};
    const critical = dsOptions.critical;
    if (Array.isArray(critical)) return critical;
    // Default: keystore is always critical for address selects
    return ["keystore"];
  }, [actionDsConfig]);

  // Detect if this is an edit operation (prefilledData contains operator/address)
  const isEditMode = React.useMemo(() => {
    return !!(prefilledData?.operator || prefilledData?.address);
  }, [prefilledData]);

  // Merge action-level DS with field-level DS (for backwards compatibility)
  const mergedDs = React.useMemo(
    () => ({
      ...actionDs,
      ...localDs,
    }),
    [actionDs, localDs],
  );
  const feesResolved = useResolvedFees(chain?.fees, {
    actionId: action?.id,
    bucket: "avg",
    ctx: { chain },
  });

  const ttlSec = chain?.session?.unlockTimeoutSec ?? 900;
  React.useEffect(() => {
    attachIdleRenew(ttlSec);
  }, [ttlSec]);

  const requiresAuth =
    (action?.auth?.type ??
      (action?.submit?.base === "admin" ? "sessionPassword" : "none")) ===
    "sessionPassword";
  const [unlockOpen, setUnlockOpen] = React.useState(false);

  // Check if submit button should be hidden (for view-only actions like "receive")
  const hideSubmit = (action as any)?.ui?.hideSubmit ?? false;

  /**
   * Helper function for modules/components to mark fields as programmatically prefilled
   * This will hide the paste button for those fields
   *
   * Usage example in a custom component:
   * ```tsx
   * // When programmatically setting a value
   * setVal('output', someAddress);
   * ctx.__markFieldsAsPrefilled(['output']);
   * ```
   *
   * @param fieldNames - Array of field names to mark as programmatically prefilled
   */
  const markFieldsAsPrefilled = React.useCallback((fieldNames: string[]) => {
    setProgrammaticallyPrefilled((prev) => {
      const newSet = new Set(prev);
      fieldNames.forEach((name) => newSet.add(name));
      return newSet;
    });
  }, []);

  /**
   * Helper function to unmark fields (allow paste button again)
   * Use this when user manually clears the field
   *
   * @param fieldNames - Array of field names to unmark
   */
  const unmarkFieldsAsPrefilled = React.useCallback((fieldNames: string[]) => {
    setProgrammaticallyPrefilled((prev) => {
      const newSet = new Set(prev);
      fieldNames.forEach((name) => newSet.delete(name));
      return newSet;
    });
  }, []);

  const templatingCtx = React.useMemo(
    () => ({
      form: debouncedForm,
      layout: (action as any)?.form?.layout,
      chain,
      account: selectedAccount
        ? {
            address: selectedAccount.address,
            nickname: selectedAccount.nickname,
            pubKey: selectedAccount.publicKey,
          }
        : undefined,
      fees: {
        ...feesResolved,
      },
      params: {
        ...params,
      },
      ds: mergedDs, // Use merged DS (action-level + field-level)
      session: { password: session?.password },
      // Unique scope for this action instance to prevent cache collisions
      __scope: `action:${actionId}:${selectedAccount?.address || "no-account"}`,
      // Track programmatically prefilled fields (hide paste button for these)
      __programmaticallyPrefilled: programmaticallyPrefilled,
      // Helper functions for custom components
      __markFieldsAsPrefilled: markFieldsAsPrefilled,
      __unmarkFieldsAsPrefilled: unmarkFieldsAsPrefilled,
    }),
    [
      debouncedForm,
      chain,
      selectedAccount,
      feesResolved,
      session?.password,
      params,
      mergedDs,
      actionId,
      programmaticallyPrefilled,
      markFieldsAsPrefilled,
      unmarkFieldsAsPrefilled,
    ],
  );

  const infoItems = React.useMemo(
    () =>
      (action?.form as any)?.info?.items?.map((it: any) => ({
        label:
          typeof it.label === "string"
            ? template(it.label, templatingCtx)
            : it.label,
        icon: it.icon,
        value:
          typeof it.value === "string"
            ? template(it.value, templatingCtx)
            : it.value,
      })) ?? [],
    [action, templatingCtx],
  );

  const rawSummary = React.useMemo(() => {
    const formSum = (action as any)?.form?.confirmation?.summary;
    return Array.isArray(formSum) ? formSum : [];
  }, [action]);

  const summaryTitle = React.useMemo(() => {
    const title = (action as any)?.form?.confirmation?.title;
    return typeof title === "string" ? template(title, templatingCtx) : title;
  }, [action, templatingCtx]);

  const resolvedSummary = React.useMemo(() => {
    return rawSummary.map((item: any) => ({
      label:
        typeof item.label === "string"
          ? template(item.label, templatingCtx)
          : item.label,
      icon: item.icon, // opcional
      value:
        typeof item.value === "string"
          ? template(item.value, templatingCtx)
          : item.value,
    }));
  }, [rawSummary, templatingCtx]);

  const hasSummary = resolvedSummary.length > 0;

  const confirmBtn = React.useMemo(() => {
    const btn =
      (action as any)?.form?.confirmation?.btns?.submit ??
      (action as any)?.form?.confirmation?.btn ??
      {};
    return {
      label:
        typeof btn.label === "string"
          ? template(btn.label, templatingCtx)
          : (btn.label ?? "Confirm"),
      icon: btn.icon ?? undefined,
    };
  }, [action, templatingCtx]);

  const isReady = React.useMemo(() => !!action && !!chain, [action, chain]);

  const didInitToastRef = React.useRef(false);
  React.useEffect(() => {
    if (!action || !isReady) return;
    if (didInitToastRef.current) return;
    const t = resolveToastFromManifest(action, "onInit", templatingCtx);
    if (t) toast.neutral(t);
    didInitToastRef.current = true;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [action, isReady]);

  const normForm = React.useMemo(
    () => normalizeFormForAction(action as any, debouncedForm),
    [action, debouncedForm],
  );
  const payload = React.useMemo(
    () =>
      buildPayloadFromAction(action as any, {
        form: normForm,
        chain,
        session: { password: session.password },
        account: selectedAccount
          ? {
              address: selectedAccount.address,
              nickname: selectedAccount.nickname,
              pubKey: selectedAccount.publicKey,
            }
          : undefined,
        fees: {
          ...feesResolved,
        },
        ds: mergedDs,
      }),
    [
      action,
      normForm,
      chain,
      session.password,
      feesResolved,
      selectedAccount,
      mergedDs,
    ],
  );

  const host = React.useMemo(() => {
    if (!action || !chain) return "";
    return action?.submit?.base === "admin"
      ? (chain.rpc.admin ?? chain.rpc.base ?? "")
      : (chain.rpc.base ?? "");
  }, [action, chain]);

  const doExecute = React.useCallback(async () => {
    if (!isReady) return;
    if (requiresAuth && !session.isUnlocked()) {
      setUnlockOpen(true);
      return;
    }
    const before = resolveToastFromManifest(
      action,
      "onBeforeSubmit",
      templatingCtx,
    );
    if (before) toast.neutral(before);
    setStage("executing");
    const submitPath =
      typeof action!.submit?.path === "string"
        ? template(action!.submit.path, templatingCtx)
        : action!.submit?.path;
    const httpRes = await fetch(host + submitPath, {
      method: action!.submit?.method,
      headers: action!.submit?.headers ?? {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    });

    let res: any = null;
    try {
      res = await httpRes.json();
    } catch {
      // Some endpoints may return plain text (e.g., tx hash) or empty body.
      try {
        const txt = await httpRes.text();
        res = txt || null;
      } catch {
        res = null;
      }
    }


    setTxRes(res);

    // Success detection prioritizes HTTP status to avoid false negatives on valid payloads
    // like {"approve": false, ...} or {"address":"..."}.
    const hasExplicitError =
      !!res?.error ||
      res?.ok === false ||
      res?.success === false ||
      (typeof res?.status === "number" && res.status >= 400);

    const isSuccess =
      httpRes.ok &&
      (typeof res === "string" ||
        res == null ||
        (typeof res === "object" && !hasExplicitError));
    const key = isSuccess ? "onSuccess" : "onError";
    const t = resolveToastFromManifest(action, key as any, templatingCtx, res);

    if (t) {
      toast.toast(t);
    } else {
      // Select appropriate mapper based on action ID
      let mapper = genericResultMap;
      if (action?.id === "pauseValidator") {
        mapper = pauseValidatorMap;
      } else if (action?.id === "unpauseValidator") {
        mapper = unpauseValidatorMap;
      }

      toast.fromResult({
        result: typeof res === "string" ? res : { ...res, ok: isSuccess },
        ctx: templatingCtx,
        map: (r, c) => mapper(r, c),
        fallback: {
          title: "Processed",
          variant: "neutral",
          ctx: templatingCtx,
        } as ToastTemplateOptions,
      });
    }
    const fin = resolveToastFromManifest(
      action,
      "onFinally",
      templatingCtx,
      res,
    );
    if (fin) toast.info(fin);

    // Close modal/finish action after execution with a small delay
    // to allow toast to be visible before modal closes
    setTimeout(() => {
      if (onFinish) {
        onFinish();
      } else {
        // If no onFinish callback, reset to form stage
        setStage("form");
        setStepIdx(0);
      }
    }, 500);
  }, [isReady, requiresAuth, session, host, action, payload]);

  const onContinue = React.useCallback(() => {
    if (formHasErrors) {
      // opcional: mostrar toast o vibrar el botón
      return;
    }
    if (hasSummary) {
      setStage("confirm");
    } else {
      void doExecute();
    }
  }, [formHasErrors, hasSummary, doExecute]);

  const onConfirm = React.useCallback(() => {
    if (formHasErrors) {
      // opcional: toast
      return;
    }
    void doExecute();
  }, [formHasErrors, doExecute]);

  const onBackToForm = React.useCallback(() => {
    setStage("form");
  }, []);

  React.useEffect(() => {
    if (unlockOpen && session.isUnlocked()) {
      setUnlockOpen(false);
      void doExecute();
    }
  }, [unlockOpen, session]);

  const onFormChange = React.useCallback((patch: Record<string, any>) => {
    setForm((prev) => ({ ...prev, ...patch }));
  }, []);

  const [errorsMap, setErrorsMap] = React.useState<Record<string, string>>({});
  const [stepIdx, setStepIdx] = React.useState(0);

  const wizard = React.useMemo(() => (action as any)?.form?.wizard, [action]);
  const allFields = React.useMemo(() => getFieldsFromAction(action), [action]);

  const steps = React.useMemo(() => {
    if (!wizard) return [];
    const declared = Array.isArray(wizard.steps) ? wizard.steps : [];
    if (declared.length) return declared;
    const uniq = Array.from(
      new Set(allFields.map((f: any) => f.step).filter(Boolean)),
    );
    return uniq.map((id: any, i) => ({ id, title: `Step ${i + 1}` }));
  }, [wizard, allFields]);

  const fieldsForStep = React.useMemo(() => {
    if (!wizard || !steps.length) return allFields;
    const cur = steps[stepIdx]?.id ?? stepIdx + 1;
    return allFields.filter(
      (f: any) => (f.step ?? 1) === cur || String(f.step) === String(cur),
    );
  }, [wizard, steps, stepIdx, allFields]);

  const visibleFieldsForStep = React.useMemo(() => {
    const list = fieldsForStep ?? [];
    return list.filter((f: any) => {
      if (!f?.showIf) return true;
      try {
        return templateBool(f.showIf, { ...templatingCtx, form });
      } catch (e) {
        console.warn("Error evaluating showIf", f.name, e);
        return true;
      }
    });
  }, [fieldsForStep, templatingCtx, form]);

  // Use PopulateController for phase-based form initialization
  // This replaces the old auto-populate useEffect with a cleaner approach
  const { phase: populatePhase, showLoading: showPopulateLoading } = usePopulateController({
    fields: allFields, // Use all fields, not just visible ones, for initial populate
    form,
    ds: mergedDs,
    isDsLoading,
    criticalDsKeys,
    dsFetchStatus, // Pass fetch status to check if DS completed (success or error)
    templateContext: templatingCtx,
    onFormChange: (patch) => setForm(prev => ({ ...prev, ...patch })),
    prefilledData,
    isEditMode,
  });

  const handleErrorsChange = React.useCallback(
    (errs: Record<string, string>, hasErrors: boolean) => {
      setErrorsMap(errs);
      setFormHasErrors(hasErrors);
    },
    [],
  );

  const hasStepErrors = React.useMemo(() => {
    const evalCtx = { ...templatingCtx, form };
    const missingRequired = visibleFieldsForStep.some((f: any) => {
      // Evaluate required - can be boolean or template string
      let isRequired = false;
      if (typeof f.required === "boolean") {
        isRequired = f.required;
      } else if (typeof f.required === "string") {
        try {
          isRequired = templateBool(f.required, evalCtx);
        } catch {
          isRequired = false;
        }
      }
      return isRequired && (form[f.name] == null || form[f.name] === "");
    });
    const fieldErrors = visibleFieldsForStep.some(
      (f: any) => !!errorsMap[f.name],
    );
    return missingRequired || fieldErrors;
  }, [visibleFieldsForStep, form, errorsMap, templatingCtx]);

  const isLastStep = !wizard || stepIdx >= steps.length - 1;

  const goNext = React.useCallback(() => {
    if (hasStepErrors) return;
    if (!wizard || isLastStep) {
      if (hasSummary) setStage("confirm");
      else void doExecute();
    } else {
      setStepIdx((i) => i + 1);
    }
  }, [wizard, isLastStep, hasStepErrors, hasSummary, doExecute]);

  const goPrev = React.useCallback(() => {
    if (!wizard) return;
    setStepIdx((i) => Math.max(0, i - 1));
  }, [wizard]);

  return (
    <div className="space-y-6">
      {stage === "confirm" && (
        <button
          onClick={onBackToForm}
          className="flex  justify-between items-center gap-2 z-10 p-1 font-bold text-canopy-50 "
        >
          <LucideIcon name="arrow-left" />
          Go back
        </button>
      )}
      <div className={cx("flex flex-col gap-4", className)}>
        {isLoading && <div>Loading…</div>}
        {!isLoading && !isReady && (
          <div>No action "{actionId}" found in manifest</div>
        )}

        {!isLoading && isReady && (
          <>
            {stage === "form" && (
              <motion.div className="space-y-4">
                {/* Show skeleton loading while waiting for critical DS */}
                {showPopulateLoading && (
                  <div className="space-y-4 animate-pulse">
                    <div className="h-10 bg-muted/50 rounded-lg w-full" />
                    <div className="h-10 bg-muted/50 rounded-lg w-full" />
                    <div className="h-10 bg-muted/50 rounded-lg w-3/4" />
                    <div className="flex justify-center pt-4">
                      <div className="text-sm text-muted-foreground">Loading form data...</div>
                    </div>
                  </div>
                )}
                {!showPopulateLoading && (
                  <FormRenderer
                    fields={visibleFieldsForStep}
                    value={form}
                    onChange={onFormChange}
                    ctx={templatingCtx}
                    onErrorsChange={handleErrorsChange}
                    onDsChange={setLocalDs}
                  />
                )}

                {wizard && steps.length > 0 && (
                  <div className="flex items-center justify-between text-xs text-muted-foreground">
                    <div>{steps[stepIdx]?.title ?? `Step ${stepIdx + 1}`}</div>
                    <div>
                      {stepIdx + 1} / {steps.length}
                    </div>
                  </div>
                )}

                {infoItems.length > 0 && (
                  <div className="flex-col h-full p-3 sm:p-4 rounded-lg bg-background">
                    {action?.form?.info?.title && (
                      <h4 className="text-canopy-50 text-base sm:text-lg mb-2">
                        {template(action?.form?.info?.title, templatingCtx)}
                      </h4>
                    )}
                    <div className="mt-3 space-y-2">
                      {infoItems.map(
                        (
                          d: {
                            icon: string | undefined;
                            label:
                              | string
                              | number
                              | boolean
                              | React.ReactElement<
                                  any,
                                  string | React.JSXElementConstructor<any>
                                >
                              | Iterable<React.ReactNode>
                              | React.ReactPortal
                              | null
                              | undefined;
                            value: any;
                          },
                          i: React.Key | null | undefined,
                        ) => (
                          <div
                            key={i}
                            className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-1 sm:gap-2 text-sm sm:text-md"
                          >
                            <div className="flex items-center gap-2 text-muted-foreground font-light text-xs sm:text-sm">
                              {d.icon ? (
                                <LucideIcon name={d.icon} className="w-3.5 h-3.5 sm:w-4 sm:h-4 flex-shrink-0" />
                              ) : null}
                              <span>
                                {d.label}
                                {d.value && ":"}
                              </span>
                            </div>
                            {d.value && (
                              <span className="font-normal text-canopy-50 break-words text-sm sm:text-base">
                                {String(d.value ?? "—")}
                              </span>
                            )}
                          </div>
                        ),
                      )}
                    </div>
                  </div>
                )}

                {!hideSubmit && (
                  <div className="flex gap-2">
                    {wizard && stepIdx > 0 && (
                      <button
                        onClick={goPrev}
                        className="px-4 py-2 sm:py-2.5 rounded border border-muted text-canopy-50 text-sm sm:text-base"
                      >
                        Back
                      </button>
                    )}
                    <button
                      disabled={hasStepErrors}
                      onClick={goNext}
                      className={cx(
                        "flex-1 px-4 py-2.5 sm:py-3 bg-primary-500 text-bg-accent-foreground font-bold rounded text-sm sm:text-base",
                        hasStepErrors && "opacity-50 cursor-not-allowed",
                      )}
                    >
                      {!wizard || isLastStep ? "Continue" : "Next"}
                    </button>
                  </div>
                )}
              </motion.div>
            )}

            {stage === "confirm" && (
              <motion.div className="space-y-4">
                <div className="flex-col h-full p-3 sm:p-4 rounded-lg bg-background">
                  {summaryTitle && (
                    <h4 className="text-canopy-50 text-base sm:text-lg mb-3">{summaryTitle}</h4>
                  )}

                  <div className="mt-3 space-y-2.5">
                    {resolvedSummary.map((d, i) => (
                      <div
                        key={i}
                        className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-1 sm:gap-2 text-sm sm:text-md"
                      >
                        <div className="flex items-center gap-2 text-muted-foreground font-light text-xs sm:text-sm">
                          {d.icon ? (
                            <LucideIcon name={d.icon} className="w-3.5 h-3.5 sm:w-4 sm:h-4 flex-shrink-0" />
                          ) : null}
                          <span>{d.label}:</span>
                        </div>
                        <span className="font-normal text-canopy-50 break-words text-sm sm:text-base sm:text-right">
                          {String(d.value ?? "—")}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="flex flex-col gap-2">
                  <button
                    onClick={onConfirm}
                    className="flex-1 px-4 py-2.5 sm:py-3 bg-primary-500 text-bg-accent-foreground font-bold rounded flex items-center justify-center gap-2 text-sm sm:text-base"
                  >
                    {confirmBtn.icon ? (
                      <LucideIcon name={confirmBtn.icon} className="w-4 h-4 sm:w-5 sm:h-5" />
                    ) : null}
                    <span>{confirmBtn.label}</span>
                  </button>
                </div>
              </motion.div>
            )}

            {stage === "executing" && (
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                className="flex flex-col items-center justify-center py-12 space-y-4"
              >
                <div className="relative">
                  <div className="w-16 h-16 border-4 border-primary/30 border-t-primary rounded-full animate-spin"></div>
                </div>
                <div className="text-center space-y-2">
                  <h3 className="text-lg font-semibold text-foreground">
                    Processing Transaction...
                  </h3>
                  <p className="text-sm text-muted-foreground">
                    Please wait while your transaction is being processed
                  </p>
                </div>
              </motion.div>
            )}

            <UnlockModal
              address={form.address || ""}
              ttlSec={ttlSec}
              open={unlockOpen}
              onClose={() => setUnlockOpen(false)}
            />
          </>
        )}
      </div>
    </div>
  );
}

