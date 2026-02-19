import React from "react";
import type { Action } from "@/manifest/types";
import FormRenderer from "./FormRenderer";
import Confirm from "./Confirm";
import Result from "./Result";
import { template } from "@/core/templater";
import { useResolvedFees } from "@/core/fees";
import { useSession, attachIdleRenew } from "@/state/session";
import UnlockModal from "../components/UnlockModal";
import { useConfig } from "@/app/providers/ConfigProvider";

type Stage = "form" | "confirm" | "executing" | "result";

export default function WizardRunner({ action }: { action: Action }) {
  const { chain } = useConfig();
  const [stage, setStage] = React.useState<Stage>("form");
  const [stepIndex, setStepIndex] = React.useState(0);
  const step = action.steps?.[stepIndex];
  const [form, setForm] = React.useState<Record<string, any>>({});
  const [txRes, setTxRes] = React.useState<any>(null);

  const session = useSession();
  const ttlSec = chain?.session?.unlockTimeoutSec ?? 900;
  React.useEffect(() => {
    attachIdleRenew(ttlSec);
  }, [ttlSec]);

  const requiresAuth =
    (action?.auth?.type ??
      (action?.rpc?.base === "admin" ? "sessionPassword" : "none")) ===
    "sessionPassword";
  const [unlockOpen, setUnlockOpen] = React.useState(false);

  const feesResolved = useResolvedFees(chain?.fees, {
    actionId: action?.id,
    bucket: "avg",
    ctx: { form, chain, action },
  });
  const fee = feesResolved.amount;

  const host = React.useMemo(
    () =>
      action.rpc?.base === "admin"
        ? (chain?.rpc.admin ?? chain?.rpc.base ?? "")
        : (chain?.rpc.base ?? ""),
    [action.rpc?.base, chain?.rpc.admin, chain?.rpc.base],
  );

  const payload = React.useMemo(
    () =>
      template(action.rpc?.payload ?? {}, {
        form,
        chain,
        session: { password: session.password },
      }),
    [action.rpc?.payload, form, chain, session.password],
  );

  const confirmSummary = React.useMemo(
    () =>
      (action.confirm?.summary ?? []).map((s) => ({
        label: s.label,
        value: template(s.value, {
          form,
          chain,
          fees: { effective: fee },
        }),
      })),
    [action.confirm?.summary, form, chain, fee],
  );

  const onNext = React.useCallback(() => {
    if ((action.steps?.length ?? 0) > stepIndex + 1) setStepIndex((i) => i + 1);
    else setStage("confirm");
  }, [action.steps?.length, stepIndex]);

  const onPrev = React.useCallback(() => {
    setStepIndex((i) => (i > 0 ? i - 1 : i));
    if (stepIndex === 0) setStage("form");
  }, [stepIndex]);

  const onFormChange = React.useCallback((patch: Record<string, any>) => {
    setForm((prev) => ({ ...prev, ...patch }));
  }, []);

  const doExecute = React.useCallback(async () => {
    if (requiresAuth && !session.isUnlocked()) {
      setUnlockOpen(true);
      return;
    }
    setStage("executing");
    const res = await fetch(host + action.rpc?.path, {
      method: action.rpc?.method,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    })
      .then((r) => r.json())
      .catch(() => ({ hash: "0xDEMO" }));
    setTxRes(res);
    setStage("result");
  }, [
    requiresAuth,
    session,
    host,
    action.rpc?.method,
    action.rpc?.path,
    payload,
  ]);

  React.useEffect(() => {
    if (unlockOpen && session.isUnlocked()) {
      setUnlockOpen(false);
      void doExecute();
    }
  }, [unlockOpen, session, doExecute]);

  if (!step) return <div>Invalid wizard</div>;

  const asideOn = step.form?.layout?.aside?.show;
  const asideWidth = step.form?.layout?.aside?.width ?? 5;
  const mainWidth = 12 - (asideOn ? asideWidth : 0);

  return (
    <div className="space-y-6">
      <div className="bg-card border border-border rounded p-4">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg">{step.title ?? "Step"}</h3>
          <div className="text-sm text-muted-foreground">
            Step {stepIndex + 1} / {action.steps?.length ?? 1}
          </div>
        </div>

        <div className="grid grid-cols-12 gap-4">
          <div className={`col-span-${mainWidth}`}>
            <FormRenderer
              fields={step.form?.fields ?? []}
              value={form}
              onChange={onFormChange}
            />
            <div className="flex justify-end mt-4 gap-2">
              {stepIndex > 0 && (
                <button
                  onClick={onPrev}
                  className="px-3 py-2 bg-muted rounded"
                >
                  Back
                </button>
              )}
              <button
                onClick={onNext}
                className="px-3 py-2 bg-emerald-500 text-black rounded"
              >
                {stepIndex + 1 < (action.steps?.length ?? 1)
                  ? "Continue"
                  : "Review"}
              </button>
            </div>
          </div>

          {asideOn && (
            <div className={`col-span-${asideWidth}`}>
              <div className="bg-background border border-border rounded p-3">
                <div className="text-sm text-muted-foreground mb-2">Sidebar</div>
                <div className="text-xs text-muted-foreground">
                  Add widget: {step.aside?.widget ?? "custom"}
                </div>
              </div>
            </div>
          )}
        </div>

        {stage === "confirm" && (
          <Confirm
            summary={confirmSummary}
            ctaLabel={action.confirm?.ctaLabel ?? "Confirm"}
            danger={!!action.confirm?.danger}
            showPayload={!!action.confirm?.showPayload}
            payload={
              action.confirm?.payloadSource === "rpc.payload"
                ? payload
                : action.confirm?.payloadTemplate
            }
            onBack={() => setStage("form")}
            onConfirm={doExecute}
          />
        )}

        <UnlockModal
          address={form.address || ""}
          ttlSec={ttlSec}
          open={unlockOpen}
          onClose={() => setUnlockOpen(false)}
        />

        {stage === "result" && (
          <Result
            message={template(action.success?.message ?? "Done", {
              form,
              chain,
            })}
            link={
              action.success?.links?.[0]
                ? {
                    label: action.success.links[0].label,
                    href: template(action.success.links[0].href, {
                      result: txRes,
                    }),
                  }
                : undefined
            }
            onDone={() => {
              setStepIndex(0);
              setStage("form");
            }}
          />
        )}
      </div>
    </div>
  );
}

