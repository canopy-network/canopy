import React from "react";
import type { Field, FieldOp } from "@/manifest/types";
import { cx } from "@/ui/cx";
import { validateField } from "./validators";
import { useSession } from "@/state/session";
import { FieldControl } from "@/actions/FieldControl";
import { motion } from "framer-motion";

const Grid: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <motion.div className="grid grid-cols-12 gap-3 sm:gap-3.5 md:gap-4">{children}</motion.div>
);

type Props = {
  fields: Field[];
  value: Record<string, any>;
  onChange: (patch: Record<string, any>) => void;
  ctx?: Record<string, any>;
  onErrorsChange?: (errors: Record<string, string>, hasErrors: boolean) => void;
  onFormOperation?: (fieldOperation: FieldOp) => void;
  onDsChange?: React.Dispatch<React.SetStateAction<Record<string, any>>>;
};

export default function FormRenderer({
  fields,
  value,
  onChange,
  ctx,
  onErrorsChange,
  onDsChange,
}: Props) {
  const [errors, setErrors] = React.useState<Record<string, string>>({});
  const [localDs, setLocalDs] = React.useState<Record<string, any>>({});
  const session = useSession();


  // When localDs changes, notify parent (ActionRunner)
  React.useEffect(() => {
    if (onDsChange && Object.keys(localDs).length > 0) {
      onDsChange((prev) => {
        const merged = { ...prev, ...localDs };
        // Only update if actually changed
        if (JSON.stringify(prev) === JSON.stringify(merged)) return prev;
        return merged;
      });
    }
  }, [localDs, onDsChange]);

  // For DS-critical fields (option, optionCard, switch), use immediate form values
  // For text input fields, use debounced values
  const templateContext = React.useMemo(
    () => ({
      form: value, // Use immediate form values for DS reactivity
      chain: ctx?.chain,
      account: ctx?.account,
      ds: { ...(ctx?.ds || {}), ...localDs },
      fees: ctx?.fees,
      params: ctx?.params,
      layout: ctx?.layout,
      session: { password: session?.password },
    }),
    [
      value,
      ctx?.chain,
      ctx?.account,
      ctx?.ds,
      ctx?.fees,
      ctx?.params,
      ctx?.layout,
      session?.password,
      localDs,
    ],
  );


  const fieldsKeyed = React.useMemo(
    () =>
      fields.map((f: any) => ({
        ...f,
        __key: `${f.tab ?? "default"}:${f.group ?? ""}:${f.name}`,
      })),
    [fields],
  );

  /** setVal + async validation */
  const setVal = React.useCallback(
    (fOrName: Field | string, v: any) => {
      const name =
        typeof fOrName === "string" ? fOrName : (fOrName as any).name;
      onChange({ [name]: v });

      void (async () => {
        const f =
          typeof fOrName === "string"
            ? (fieldsKeyed.find((x) => x.name === fOrName) as Field | undefined)
            : (fOrName as Field);

        const e = await validateField((f as any) ?? {}, v, templateContext);
        const errorMessage = !e.ok ? e.message : "";
        setErrors((prev) =>
          prev[name] === errorMessage
            ? prev
            : { ...prev, [name]: errorMessage },
        );
      })();
    },
    [onChange, ctx?.chain, fieldsKeyed],
  );

  const hasActiveErrors = React.useMemo(() => {
    const anyMsg = Object.values(errors).some((m) => !!m);
    const requiredMissing = fields.some(
      (f) => f.required && (value[f.name] == null || value[f.name] === ""),
    );
    return anyMsg || requiredMissing;
  }, [errors, fields, value]);

  React.useEffect(() => {
    onErrorsChange?.(errors, hasActiveErrors);
  }, [errors, hasActiveErrors, onErrorsChange]);

  const tabs = React.useMemo(
    () =>
      Array.from(
        new Set(fieldsKeyed.map((f: any) => f.tab).filter(Boolean)),
      ) as string[],
    [fieldsKeyed],
  );
  const [activeTab, setActiveTab] = React.useState(tabs[0] ?? "default");
  const fieldsInTab = React.useCallback(
    (t?: string) =>
      fieldsKeyed.filter((f: any) => (tabs.length ? f.tab === t : true)),
    [fieldsKeyed, tabs],
  );

  return (
    <>
      {tabs.length > 0 && (
        <div className="mb-3 flex gap-2 border-b border-neutral-800">
          {tabs.map((t) => (
            <button
              key={t}
              className={cx(
                "px-3 py-2 -mb-px border-b-2",
                activeTab === t
                  ? "border-emerald-400 text-emerald-400"
                  : "border-transparent text-neutral-400",
              )}
              onClick={() => setActiveTab(t)}
            >
              {t}
            </button>
          ))}
        </div>
      )}
      <Grid>
        {(tabs.length ? fieldsInTab(activeTab) : fieldsKeyed).map((f: any) => (
          <FieldControl
            key={f.__key}
            f={f}
            value={value}
            errors={errors}
            templateContext={templateContext}
            setVal={setVal}
            setLocalDs={setLocalDs}
          />
        ))}
      </Grid>
    </>
  );
}
