import React from "react";
import { Field } from "@/manifest/types";
import { collectDepsFromObject, template } from "@/core/templater";
import { templateBool } from "@/core/templater";
import { useFieldDs } from "@/actions/useFieldsDs";
import { getFieldRenderer } from "@/actions/fields/fieldRegistry";

type Props = {
  f: Field;
  value: Record<string, any>;
  errors: Record<string, string>;
  templateContext: Record<string, any>;
  setVal: (field: Field | string, v: any) => void;
  setLocalDs?: React.Dispatch<React.SetStateAction<Record<string, any>>>;
};

export const FieldControl: React.FC<Props> = ({
  f,
  value,
  errors,
  templateContext,
  setVal,
  setLocalDs,
}) => {
  const resolveTemplate = React.useCallback(
    (s?: any) => (typeof s === "string" ? template(s, templateContext) : s),
    [templateContext],
  );

  const manualWatch: string[] = React.useMemo(() => {
    const dsObj: any = (f as any)?.ds;
    const watch = dsObj?.__options?.watch;
    return Array.isArray(watch) ? watch : [];
  }, [f]);

  const autoWatchAllRoots: string[] = React.useMemo(() => {
    const dsObj: any = (f as any)?.ds;
    return collectDepsFromObject(dsObj);
  }, [f]);

  const autoWatchFormOnly: string[] = React.useMemo(() => {
    return autoWatchAllRoots
      .filter((p) => p.startsWith("form."))
      .map((p) => p.replace(/^form\.\??/, "form."));
  }, [autoWatchAllRoots]);

  const watchPaths: string[] = React.useMemo(() => {
    const merged = new Set<string>([...manualWatch, ...autoWatchFormOnly]);
    return Array.from(merged);
  }, [manualWatch, autoWatchFormOnly]);

  const { data: dsValue } = useFieldDs(f, templateContext);

  React.useEffect(() => {
    if (!setLocalDs || dsValue == null) return;

    const fieldDs = (f as any)?.ds;
    if (!fieldDs || typeof fieldDs !== "object") return;

    const declaredKeys = Object.keys(fieldDs).filter((k) => k !== "__options");
    if (declaredKeys.length === 0) return;

    setLocalDs((prev) => {
      const next = { ...(prev || {}) };
      let changed = false;

      for (const key of declaredKeys) {
        const incoming = (dsValue as any)?.[key] ?? dsValue;

        if (incoming === undefined) continue;

        const prevForKey = (prev as any)?.[key];

        try {
          const prevStr = JSON.stringify(prevForKey);
          const incomingStr = JSON.stringify(incoming);
          if (prevStr !== incomingStr) {
            next[key] = incoming;
            changed = true;
          }
        } catch {
          if (prevForKey !== incoming) {
            next[key] = incoming;
            changed = true;
          }
        }
      }

      return changed ? next : prev;
    });
  }, [dsValue, setLocalDs, f]);

  const isVisible =
    (f as any).showIf == null
      ? true
      : templateBool((f as any).showIf, templateContext);

  if (!isVisible) return null;

  const FieldRenderer = getFieldRenderer(f.type);

  if (!FieldRenderer) {
    return (
      <div className="col-span-12 text-sm text-muted-foreground">
        Unsupported field type: {f.type}
      </div>
    );
  }

  const error = errors[f.name];
  const currentValue = value[f.name] ?? "";

  return (
    <FieldRenderer
      field={f}
      value={currentValue}
      error={error}
      errors={errors}
      templateContext={templateContext}
      dsValue={dsValue}
      onChange={(val: any) => setVal(f, val)}
      resolveTemplate={resolveTemplate}
      setVal={(fieldId: string, v: any) => setVal(fieldId, v)}
    />
  );
};
