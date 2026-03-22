// ActionRunner.tsx
import React from "react";
import { useConfig } from "@/app/providers/ConfigProvider";
import FormRenderer from "./FormRenderer";
import { useResolvedFees } from "@/core/fees";
import UnlockModal from "../components/UnlockModal";
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
import { AlertTriangle, Copy, CheckCircle2 } from "lucide-react";
import { ToastTemplateOptions } from "@/toast/types";
import { useActionDs } from "./useActionDs";
import { usePopulateController } from "./usePopulateController";
import { resolveRpcHost } from "@/core/rpcHost";
import type { ActionFinishResult } from "@/app/providers/ActionModalProvider";

type Stage = "form" | "confirm" | "executing" | "result";

type InlineServiceError = {
  status: number;
  statusText: string;
  message: string;
  requestMethod: string;
  requestPath: string;
  response: any;
  rawResponse: string;
  occurredAt: string;
};

const parseServiceResponse = (raw: string): any => {
  const trimmed = typeof raw === "string" ? raw.trim() : "";
  if (!trimmed) return null;
  try {
    return JSON.parse(trimmed);
  } catch {
    return raw;
  }
};

const extractServiceErrorMessage = (response: any, statusText?: string): string => {
  if (typeof response === "string" && response.trim()) return response.trim();
  if (response && typeof response === "object") {
    const candidates = [
      response?.error?.message,
      response?.error?.reason,
      response?.error?.details,
      response?.message,
      response?.reason,
      response?.detail,
      response?.description,
      typeof response?.data === "object" ? response?.data?.message : undefined,
      typeof response?.data === "string" ? response?.data : undefined,
    ];
    for (const value of candidates) {
      if (typeof value === "string" && value.trim()) return value.trim();
    }
  }
  if (typeof statusText === "string" && statusText.trim()) return statusText.trim();
  return "Unknown service error";
};

const formatServiceResponse = (response: any, rawResponse: string): string => {
  if (response == null) return rawResponse || "(empty response)";
  if (typeof response === "string") return response;
  try {
    return JSON.stringify(response, null, 2);
  } catch {
    return rawResponse || String(response);
  }
};

export default function ActionRunner({
  actionId,
  onFinish,
  className,
  prefilledData,
}: {
  actionId: string;
  onFinish?: (result: ActionFinishResult) => void;
  className?: string;
  prefilledData?: Record<string, any>;
}) {
  const toast = useToast();

  const [formHasErrors, setFormHasErrors] = React.useState(false);
  const [stage, setStage] = React.useState<Stage>("form");
  const [form, setForm] = React.useState<Record<string, any>>(
    prefilledData || {},
  );
  const [txRes, setTxRes] = React.useState<any>(null);
  const [rawTxRes, setRawTxRes] = React.useState<string>("");
  const [inlineError, setInlineError] = React.useState<InlineServiceError | null>(null);
  const [txPassword, setTxPassword] = React.useState<string>("");
  const [pendingExecutionAfterUnlock, setPendingExecutionAfterUnlock] = React.useState(false);
  const [localDs, setLocalDs] = React.useState<Record<string, any>>({});
  // Track which fields were programmatically prefilled (from prefilledData or modules)
  // These fields should hide paste button even when they have values
  const [programmaticallyPrefilled, setProgrammaticallyPrefilled] = React.useState<Set<string>>(
    new Set(prefilledData ? Object.keys(prefilledData) : []),
  );

  const { manifest, chain, params: globalParams, isLoading } = useConfig();
  const { selectedAccount } = useAccounts?.() ?? { selectedAccount: undefined };

  // Merge global params with prefilledData so templates can access both via {{ params.fieldName }}
  const params = React.useMemo(() => ({
    ...globalParams,
    ...prefilledData,
  }), [globalParams, prefilledData]);

  const action = React.useMemo(
    () => manifest?.actions.find((a) => a.id === actionId),
    [manifest, actionId],
  );

  // Keep a normalized, non-debounced form for dynamic visibility/showIf and payload coherence.
  const normalizedLiveForm = React.useMemo(
    () => normalizeFormForAction(action as any, form),
    [action, form],
  );

  // NEW: Load action-level DS (replaces per-field DS for better performance)
  const actionDsConfig = React.useMemo(() => (action as any)?.ds, [action]);

  // Build context for DS (without ds itself to avoid circular dependency)
  // Use form (not debounced) for DS context to ensure immediate reactivity with prefilledData
  // The DS hook itself handles debouncing internally where needed
  const dsCtx = React.useMemo(
    () => ({
      form: normalizedLiveForm,
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
    [normalizedLiveForm, chain, selectedAccount, params],
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
    return [];
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
      form: normalizedLiveForm,
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
      session: { password: txPassword },
      // Unique scope for this action instance to prevent cache collisions
      __scope: `action:${actionId}:${selectedAccount?.address || "no-account"}`,
      // Track programmatically prefilled fields (hide paste button for these)
      __programmaticallyPrefilled: programmaticallyPrefilled,
      // Helper functions for custom components
      __markFieldsAsPrefilled: markFieldsAsPrefilled,
      __unmarkFieldsAsPrefilled: unmarkFieldsAsPrefilled,
    }),
    [
      normalizedLiveForm,
      chain,
      selectedAccount,
      feesResolved,
      txPassword,
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
      icon: item.icon, // optional
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

  const errorPanelConfig = React.useMemo(
    () => (action as any)?.ui?.errorPanel ?? {},
    [action],
  );

  const errorPanelCopy = React.useMemo(() => {
    const data = { ...templatingCtx, error: inlineError, result: txRes };
    const render = (value: any, fallback: string) =>
      typeof value === "string" ? template(value, data) : fallback;
    return {
      title: render(errorPanelConfig.title, "Request failed"),
      description: render(
        errorPanelConfig.description,
        "The request could not be completed. Review the service response below and retry.",
      ),
      advancedLabel: render(errorPanelConfig.advancedLabel, "Advanced details"),
      statusLabel: render(errorPanelConfig.statusLabel, "Status"),
      requestLabel: render(errorPanelConfig.requestLabel, "Request"),
      responseLabel: render(errorPanelConfig.responseLabel, "Service response"),
      defaultOpen: Boolean(errorPanelConfig.defaultOpen),
    };
  }, [errorPanelConfig, templatingCtx, inlineError, txRes]);

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
    () => normalizedLiveForm,
    [normalizedLiveForm],
  );
  const payload = React.useMemo(
    () =>
      buildPayloadFromAction(action as any, {
        form: normForm,
        chain,
        session: { password: txPassword },
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
      txPassword,
      feesResolved,
      selectedAccount,
      mergedDs,
    ],
  );

  const host = React.useMemo(() => {
    if (!action || !chain) return "";
    return resolveRpcHost(chain, action?.submit?.base ?? "rpc");
  }, [action?.submit?.base, chain]);

  const doExecute = React.useCallback(async () => {
    if (!isReady) return;
    if (requiresAuth && !txPassword) {
      setPendingExecutionAfterUnlock(true);
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
    setInlineError(null);
    const submitPath =
      typeof action!.submit?.path === "string"
        ? template(action!.submit.path, templatingCtx)
        : action!.submit?.path;
    const requestMethod = action!.submit?.method ?? "POST";
    const requestPath =
      typeof submitPath === "string"
        ? submitPath
        : String(submitPath ?? "");

    let responseOk = false;
    let responseStatus = 0;
    let responseStatusText = "";
    let rawResponse = "";
    let res: any = null;

    try {
      const httpRes = await fetch(host + requestPath, {
        method: requestMethod,
        headers: action!.submit?.headers ?? {
          "Content-Type": "application/json",
        },
        body: typeof payload === "string" ? payload : JSON.stringify(payload),
      });
      responseOk = httpRes.ok;
      responseStatus = httpRes.status;
      responseStatusText = httpRes.statusText;
      rawResponse = await httpRes.text();
      res = parseServiceResponse(rawResponse);
    } catch (error: any) {
      responseOk = false;
      responseStatus = 0;
      responseStatusText = "Network Error";
      rawResponse = error?.message ? String(error.message) : "";
      res = {
        error: {
          message: rawResponse || "Network request failed",
        },
      };
    }

    setTxRes(res);
    setRawTxRes(rawResponse);

    // Success detection prioritizes HTTP status to avoid false negatives on valid payloads
    // like {"approve": false, ...} or {"address":"..."}.
    const hasExplicitError =
      !!res?.error ||
      res?.ok === false ||
      res?.success === false ||
      (typeof res?.status === "number" && res.status >= 400) ||
      responseStatus >= 400;

    const isSuccess =
      responseOk &&
      (typeof res === "string" ||
        res == null ||
        (typeof res === "object" && !hasExplicitError));
    const executionResult: ActionFinishResult = {
      actionId,
      success: isSuccess,
      result: res,
    };

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
    setTxPassword("");

    if (!isSuccess) {
      setInlineError({
        status: responseStatus,
        statusText: responseStatusText,
        message: extractServiceErrorMessage(res, responseStatusText),
        requestMethod,
        requestPath,
        response: res,
        rawResponse,
        occurredAt: new Date().toISOString(),
      });
      // Keep modal open on failure so the user can inspect/edit and retry.
      setStage("form");
      return;
    }

    // If the response is a generated transaction (submit=false), show it
    // so the user can copy it for the next steps (approve → submit).
    const isGeneratedTx =
      res &&
      typeof res === "object" &&
      typeof res.type === "string" &&
      res.signature != null;

    if (isGeneratedTx) {
      setStage("result");
      return;
    }

    // Close modal/finish action only on success, with a small delay so toast is visible.
    setTimeout(() => {
      if (onFinish) {
        onFinish(executionResult);
      } else {
        setStage("form");
        setStepIdx(0);
      }
    }, 500);
  }, [isReady, requiresAuth, txPassword, host, action, payload, actionId, onFinish, templatingCtx, toast]);

  const onContinue = React.useCallback(() => {
    if (formHasErrors) {
      // optional: show toast or shake the button
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
      // optional: toast
      return;
    }
    void doExecute();
  }, [formHasErrors, doExecute]);

  const onBackToForm = React.useCallback(() => {
    setStage("form");
  }, []);

  React.useEffect(() => {
    if (!pendingExecutionAfterUnlock || !txPassword) return;
    setPendingExecutionAfterUnlock(false);
    void doExecute();
  }, [pendingExecutionAfterUnlock, txPassword, doExecute]);

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
      const val = form[f.name];
      return isRequired && (val == null || val === "" || (Array.isArray(val) && val.length === 0));
    });
    const fieldErrors = visibleFieldsForStep.some(
      (f: any) => !!errorsMap[f.name],
    );
    return missingRequired || fieldErrors;
  }, [visibleFieldsForStep, form, errorsMap, templatingCtx]);

  const isLastStep = !wizard || stepIdx >= steps.length - 1;
  const isOrdersAction = React.useMemo(() => /^(order|dex)/i.test(actionId), [actionId]);
  const stepProgress =
    wizard && steps.length > 0 ? Math.round(((stepIdx + 1) / steps.length) * 100) : 0;

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
          className="inline-flex items-center gap-2 z-10 px-2 py-1.5 text-sm font-semibold text-muted-foreground hover:text-foreground transition-colors"
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
                {wizard && steps.length > 0 && (
                  <div className="rounded-xl border border-border/70 bg-background/70 p-3 sm:p-4 space-y-3">
                    <div className="flex items-center justify-between text-xs text-muted-foreground">
                      <div className="font-medium">
                        {steps[stepIdx]?.title ?? `Step ${stepIdx + 1}`}
                      </div>
                      <div>
                        {stepIdx + 1} / {steps.length}
                      </div>
                    </div>
                    <div className="h-1.5 rounded-full bg-muted/60 overflow-hidden">
                      <div
                        className="h-full rounded-full bg-primary transition-all duration-300"
                        style={{ width: `${stepProgress}%` }}
                      />
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {steps.map((step: any, idx: number) => {
                        const isActive = idx === stepIdx;
                        const isCompleted = idx < stepIdx;
                        return (
                          <span
                            key={String(step?.id ?? idx)}
                            className={cx(
                              "inline-flex items-center gap-1.5 px-2 py-1 rounded-md text-[11px] border",
                              isActive
                                ? "border-primary/60 bg-primary/15 text-primary"
                                : isCompleted
                                  ? "border-emerald-500/40 bg-emerald-500/10 text-emerald-400"
                                  : "border-border/70 text-muted-foreground",
                            )}
                          >
                            <span>{idx + 1}</span>
                            <span>{step?.title ?? `Step ${idx + 1}`}</span>
                          </span>
                        );
                      })}
                    </div>
                  </div>
                )}
                {inlineError && (
                  <motion.div
                    initial={{ opacity: 0, y: -6 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="rounded-xl border border-rose-500/40 bg-[linear-gradient(135deg,rgba(159,18,57,0.24),rgba(39,39,42,0.35))] p-3 sm:p-4"
                  >
                    <div className="flex items-start gap-3">
                      <span className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-rose-400/45 bg-rose-500/20 text-rose-200">
                        <AlertTriangle className="h-4 w-4" />
                      </span>
                      <div className="min-w-0 flex-1 space-y-2">
                        <div>
                          <h4 className="text-sm font-semibold text-rose-100">
                            {errorPanelCopy.title}
                          </h4>
                          <p className="mt-1 text-xs text-rose-100/85 leading-relaxed">
                            {errorPanelCopy.description}
                          </p>
                        </div>
                        <div className="rounded-lg border border-rose-500/30 bg-black/20 px-2.5 py-2">
                          <div className="text-[11px] uppercase tracking-wide text-rose-200/80">
                            Cause
                          </div>
                          <div className="mt-1 text-sm text-rose-50 break-words">
                            {inlineError.message}
                          </div>
                        </div>
                        <details
                          className="rounded-lg border border-rose-500/25 bg-black/15 p-2.5"
                          {...(errorPanelCopy.defaultOpen ? { open: true } : {})}
                        >
                          <summary className="cursor-pointer select-none text-xs font-semibold text-rose-100/90">
                            {errorPanelCopy.advancedLabel}
                          </summary>
                          <div className="mt-2 space-y-2 text-xs">
                            <div className="grid grid-cols-1 gap-1.5 md:grid-cols-2">
                              <div className="rounded-md border border-rose-500/20 bg-black/20 px-2 py-1.5">
                                <span className="text-rose-200/75">
                                  {errorPanelCopy.statusLabel}:
                                </span>{" "}
                                <span className="font-mono text-rose-50">
                                  {inlineError.status || "N/A"} {inlineError.statusText}
                                </span>
                              </div>
                              <div className="rounded-md border border-rose-500/20 bg-black/20 px-2 py-1.5">
                                <span className="text-rose-200/75">
                                  {errorPanelCopy.requestLabel}:
                                </span>{" "}
                                <span className="font-mono text-rose-50 break-all">
                                  {inlineError.requestMethod} {inlineError.requestPath}
                                </span>
                              </div>
                            </div>
                            <div>
                              <div className="mb-1 text-[11px] uppercase tracking-wide text-rose-200/75">
                                {errorPanelCopy.responseLabel}
                              </div>
                              <pre className="max-h-56 overflow-auto rounded-md border border-rose-500/20 bg-black/30 p-2 text-[11px] text-rose-50 whitespace-pre-wrap break-words">
                                {formatServiceResponse(inlineError.response, inlineError.rawResponse)}
                              </pre>
                            </div>
                          </div>
                        </details>
                      </div>
                    </div>
                  </motion.div>
                )}
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
                  <div
                    className={cx(
                      "rounded-xl border p-3 sm:p-4 md:p-5",
                      isOrdersAction
                        ? "border-primary/25 bg-primary/[0.04]"
                        : "border-border/70 bg-background/60",
                    )}
                  >
                    <FormRenderer
                      fields={visibleFieldsForStep}
                      value={form}
                      onChange={onFormChange}
                      ctx={templatingCtx}
                      onErrorsChange={handleErrorsChange}
                      onDsChange={setLocalDs}
                    />
                  </div>
                )}

                {infoItems.length > 0 && (
                  <div className="flex-col h-full p-3 sm:p-4 rounded-xl border border-border/70 bg-background/60">
                    {action?.form?.info?.title && (
                      <h4 className="text-foreground text-sm sm:text-base font-semibold mb-2">
                        {template(action?.form?.info?.title, templatingCtx)}
                      </h4>
                    )}
                    <div className="mt-3 space-y-2.5">
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
                            className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-1 sm:gap-2 text-sm"
                          >
                            <div className="flex items-center gap-2 text-muted-foreground font-medium text-[11px] sm:text-xs uppercase tracking-wide">
                              {d.icon ? (
                                <LucideIcon name={d.icon} className="w-3.5 h-3.5 sm:w-4 sm:h-4 flex-shrink-0" />
                              ) : null}
                              <span>
                                {d.label}
                                {d.value && ":"}
                              </span>
                            </div>
                            {d.value && (
                              <span className="font-normal text-foreground break-words text-sm sm:text-[15px]">
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
                  <div className="sticky bottom-0 z-20 -mx-3 mt-4 flex flex-col-reverse gap-2 border-t border-border/70 bg-card/95 px-3 pt-3 pb-[calc(env(safe-area-inset-bottom)+0.75rem)] backdrop-blur-md sm:static sm:mx-0 sm:mt-0 sm:flex-row sm:border-0 sm:bg-transparent sm:p-0 sm:backdrop-blur-0">
                    {wizard && stepIdx > 0 && (
                      <button
                        onClick={goPrev}
                        className="px-4 py-2.5 rounded-lg border border-border/80 text-foreground text-sm sm:text-base hover:bg-muted/40 transition-colors"
                      >
                        Back
                      </button>
                    )}
                    <button
                      disabled={hasStepErrors}
                      onClick={goNext}
                      className={cx(
                        "flex-1 px-4 py-2.5 sm:py-3 bg-primary-500 text-bg-accent-foreground font-semibold rounded-lg text-sm sm:text-base",
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
                <div
                  className={cx(
                    "flex-col h-full p-3 sm:p-4 rounded-xl border",
                    isOrdersAction
                      ? "border-primary/25 bg-primary/[0.04]"
                      : "border-border/70 bg-background/60",
                  )}
                >
                  {summaryTitle && (
                    <h4 className="text-foreground text-sm sm:text-base font-semibold mb-3">{summaryTitle}</h4>
                  )}

                  <div className="mt-3 space-y-2.5">
                    {resolvedSummary.map((d, i) => (
                      <div
                        key={i}
                        className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-1 sm:gap-2 text-sm"
                      >
                        <div className="flex items-center gap-2 text-muted-foreground font-medium text-[11px] sm:text-xs uppercase tracking-wide">
                          {d.icon ? (
                            <LucideIcon name={d.icon} className="w-3.5 h-3.5 sm:w-4 sm:h-4 flex-shrink-0" />
                          ) : null}
                          <span>{d.label}:</span>
                        </div>
                        <span className="font-normal text-foreground break-words text-sm sm:text-[15px] sm:text-right">
                          {String(d.value ?? "—")}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="flex flex-col gap-2">
                  <div className="sticky bottom-0 z-20 -mx-3 mt-4 border-t border-border/70 bg-card/95 px-3 pt-3 pb-[calc(env(safe-area-inset-bottom)+0.75rem)] backdrop-blur-md sm:static sm:mx-0 sm:mt-0 sm:border-0 sm:bg-transparent sm:p-0 sm:backdrop-blur-0">
                    <button
                      onClick={onConfirm}
                      className="flex-1 w-full px-4 py-2.5 sm:py-3 bg-primary-500 text-bg-accent-foreground font-semibold rounded-lg flex items-center justify-center gap-2 text-sm sm:text-base"
                    >
                      {confirmBtn.icon ? (
                        <LucideIcon name={confirmBtn.icon} className="w-4 h-4 sm:w-5 sm:h-5" />
                      ) : null}
                      <span>{confirmBtn.label}</span>
                    </button>
                  </div>
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

            {stage === "result" && txRes && (
              <GeneratedTxResult
                txRes={txRes}
                rawJson={rawTxRes}
                onDone={() => {
                  if (onFinish) {
                    onFinish({ actionId, success: true, result: txRes });
                  } else {
                    setStage("form");
                    setStepIdx(0);
                  }
                }}
              />
            )}

            <UnlockModal
              open={unlockOpen}
              onClose={() => {
                setUnlockOpen(false);
                setPendingExecutionAfterUnlock(false);
                setTxPassword("");
              }}
              onUnlock={(password) => {
                setTxPassword(password);
                setUnlockOpen(false);
              }}
            />
          </>
        )}
      </div>
    </div>
  );
}

function GeneratedTxResult({
  txRes,
  rawJson,
  onDone,
}: {
  txRes: Record<string, unknown>;
  rawJson?: string;
  onDone: () => void;
}) {
  const [copied, setCopied] = React.useState(false);
  const jsonStr = React.useMemo(
    () => rawJson?.trim() || JSON.stringify(txRes, null, 2),
    [txRes, rawJson],
  );

  const handleCopy = React.useCallback(() => {
    navigator.clipboard.writeText(jsonStr).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }, [jsonStr]);

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      className="space-y-4"
    >
      <div className="rounded-xl border border-emerald-500/30 bg-emerald-500/10 p-4">
        <div className="flex items-center gap-2 mb-2">
          <CheckCircle2 className="w-5 h-5 text-emerald-400" />
          <h4 className="text-sm font-semibold text-emerald-100">
            Step 1 Complete — Proposal Generated
          </h4>
        </div>
        <p className="text-xs text-emerald-100/80 leading-relaxed mb-3">
          Copy the JSON below, then complete the remaining steps:
        </p>
        <ol className="text-xs text-emerald-100/80 leading-relaxed space-y-1 list-decimal list-inside">
          <li><strong>Approve</strong> — Paste this JSON into the <strong>Approve / Reject Proposal</strong> action and vote Approve. This adds it to the node's approve list.</li>
          <li><strong>Submit</strong> — Paste this JSON into the <strong>Manual Raw TX Broadcast</strong> action to broadcast it to the network.</li>
        </ol>
      </div>

      <div className="relative rounded-xl border border-border/70 bg-background/60">
        <div className="flex items-center justify-between px-3 py-2 border-b border-border/50">
          <span className="text-[11px] uppercase tracking-wide text-muted-foreground font-medium">
            Signed Transaction JSON
          </span>
          <button
            onClick={handleCopy}
            className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium border border-border/80 bg-background hover:bg-muted/50 text-foreground transition-colors"
          >
            {copied ? (
              <>
                <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
                Copied
              </>
            ) : (
              <>
                <Copy className="w-3.5 h-3.5" />
                Copy JSON
              </>
            )}
          </button>
        </div>
        <pre className="max-h-72 overflow-auto p-3 text-[11px] text-foreground/90 whitespace-pre-wrap break-words font-mono">
          {jsonStr}
        </pre>
      </div>

      <button
        onClick={onDone}
        className="w-full px-4 py-2.5 rounded-lg border border-border/80 text-foreground text-sm font-medium hover:bg-muted/40 transition-colors"
      >
        Done
      </button>
    </motion.div>
  );
}

