import React from "react";
import { template } from "@/core/templater";

/**
 * Populate Controller - manages form initialization phases
 *
 * Phases:
 * - "waiting": Critical DS is loading, form shows skeleton/loading state
 * - "initializing": DS ready, running initial populate
 * - "ready": Form is interactive, DS only updates display/validation (not form values)
 *
 * This solves race conditions between DS loading and form population.
 */

export type PopulatePhase = "waiting" | "initializing" | "ready";

export interface PopulateControllerConfig {
  /** All fields from the current step/form */
  fields: any[];
  /** Current form values */
  form: Record<string, any>;
  /** Data sources (merged) */
  ds: Record<string, any>;
  /** Is DS currently loading? */
  isDsLoading: boolean;
  /** Critical DS keys that must load before showing form (e.g., ['keystore', 'validator']) */
  criticalDsKeys: string[];
  /** Fetch status per DS key - tells us if a DS has completed (success OR error) */
  dsFetchStatus?: Record<string, { isFetched: boolean; isLoading: boolean; hasError: boolean }>;
  /** Template context for resolving field values */
  templateContext: Record<string, any>;
  /** Callback to update form values */
  onFormChange: (patch: Record<string, any>) => void;
  /** Prefilled data passed to the form (edit mode) */
  prefilledData?: Record<string, any>;
  /** Whether this is an edit operation (affects which DS are critical) */
  isEditMode?: boolean;
}

export interface PopulateControllerResult {
  /** Current phase */
  phase: PopulatePhase;
  /** Fields that have been populated at least once */
  populatedFields: Set<string>;
  /** Whether form should show loading/skeleton */
  showLoading: boolean;
  /** Whether critical DS has loaded */
  criticalDsReady: boolean;
  /** Force re-run initial populate (for edge cases) */
  reinitialize: () => void;
}

export function usePopulateController({
  fields,
  form,
  ds,
  isDsLoading,
  criticalDsKeys,
  dsFetchStatus,
  templateContext,
  onFormChange,
  prefilledData,
  isEditMode,
}: PopulateControllerConfig): PopulateControllerResult {
  const [phase, setPhase] = React.useState<PopulatePhase>("waiting");
  const [populatedFields, setPopulatedFields] = React.useState<Set<string>>(
    new Set(prefilledData ? Object.keys(prefilledData) : [])
  );

  // Track if we've completed initial population
  const hasInitializedRef = React.useRef(false);

  // Track DS snapshot at initialization time
  const initialDsSnapshotRef = React.useRef<string | null>(null);

  // Determine which DS are critical
  const effectiveCriticalKeys = React.useMemo(() => {
    const keys = new Set(criticalDsKeys);

    // keystore is always critical for address selects
    keys.add("keystore");

    // In edit mode, validator DS is often critical
    if (isEditMode && prefilledData?.operator) {
      keys.add("validator");
    }

    return Array.from(keys);
  }, [criticalDsKeys, isEditMode, prefilledData?.operator]);

  // Check if all critical DS have loaded (completed fetch, regardless of success/error)
  const criticalDsReady = React.useMemo(() => {
    // Check each critical DS key using fetch status
    for (const key of effectiveCriticalKeys) {
      const status = dsFetchStatus?.[key];

      // If we have fetch status, use it (more accurate)
      if (status) {
        // DS is ready if it has been fetched (success or error) and is not currently loading
        if (!status.isFetched || status.isLoading) {
          return false;
        }
      } else {
        // Fallback: check if DS data exists (for backwards compatibility)
        // Also consider DS ready if it doesn't exist in fetchStatus but isDsLoading is false
        // This handles cases where the DS key doesn't exist in the config
        if (isDsLoading) {
          return false;
        }
      }
    }

    return true;
  }, [dsFetchStatus, effectiveCriticalKeys, isDsLoading]);

  // Run initial population when critical DS becomes ready
  React.useEffect(() => {
    // Skip if already initialized
    if (hasInitializedRef.current) return;

    // Skip if critical DS not ready
    if (!criticalDsReady) {
      setPhase("waiting");
      return;
    }

    // Transition to initializing
    setPhase("initializing");

    // Run initial populate
    const defaults: Record<string, any> = {};
    const newlyPopulated: string[] = [];

    for (const field of fields) {
      const fieldName = field.name;
      const fieldValue = field.value;
      const autoPopulate = field.autoPopulate ?? "always";

      // Skip fields without name (visual fields like section, divider)
      if (!fieldName) continue;

      // Skip if autoPopulate is disabled
      if (autoPopulate === false) continue;

      // Skip if already prefilled
      if (prefilledData && prefilledData[fieldName] !== undefined) {
        continue;
      }

      // Skip if form already has a value (user might have typed something)
      if (form[fieldName] !== undefined && form[fieldName] !== "" && form[fieldName] !== null) {
        continue;
      }

      // Try to resolve the default value
      if (fieldValue != null) {
        try {
          const resolved = template(fieldValue, templateContext);
          if (resolved !== undefined && resolved !== "" && resolved !== null) {
            defaults[fieldName] = resolved;
            newlyPopulated.push(fieldName);
          }
        } catch (e) {
          // Template resolution failed, skip
          console.warn(`[PopulateController] Failed to resolve default for ${fieldName}:`, e);
        }
      }
    }

    // Apply defaults to form
    if (Object.keys(defaults).length > 0) {
      onFormChange(defaults);
    }

    // Mark fields as populated
    setPopulatedFields(prev => {
      const next = new Set(prev);
      newlyPopulated.forEach(f => next.add(f));
      return next;
    });

    // Mark initialization as complete
    hasInitializedRef.current = true;
    initialDsSnapshotRef.current = JSON.stringify(ds);

    // Transition to ready (slight delay for UI smoothness)
    requestAnimationFrame(() => {
      setPhase("ready");
    });
  }, [criticalDsReady, fields, form, templateContext, onFormChange, prefilledData, ds]);

  // Handle DS changes AFTER initialization (only for autoPopulate: "always" fields)
  React.useEffect(() => {
    // Only run in "ready" phase
    if (phase !== "ready") return;

    // Skip if DS hasn't actually changed
    const currentDsSnapshot = JSON.stringify(ds);
    if (currentDsSnapshot === initialDsSnapshotRef.current) return;

    const updates: Record<string, any> = {};

    for (const field of fields) {
      const fieldName = field.name;
      const fieldValue = field.value;
      const autoPopulate = field.autoPopulate;

      // Only update fields with explicit autoPopulate: "always"
      // (default behavior after initialization is to NOT auto-populate)
      if (autoPopulate !== "always") continue;

      // Skip fields without name
      if (!fieldName) continue;

      // Resolve and update
      if (fieldValue != null) {
        try {
          const resolved = template(fieldValue, templateContext);
          if (resolved !== undefined && resolved !== null) {
            // Only update if value changed
            if (form[fieldName] !== resolved) {
              updates[fieldName] = resolved;
            }
          }
        } catch (e) {
          // Skip
        }
      }
    }

    if (Object.keys(updates).length > 0) {
      onFormChange(updates);
    }
  }, [phase, ds, fields, templateContext, form, onFormChange]);

  // Reinitialize function for edge cases
  const reinitialize = React.useCallback(() => {
    hasInitializedRef.current = false;
    initialDsSnapshotRef.current = null;
    setPhase("waiting");
    setPopulatedFields(new Set(prefilledData ? Object.keys(prefilledData) : []));
  }, [prefilledData]);

  return {
    phase,
    populatedFields,
    showLoading: phase === "waiting",
    criticalDsReady,
    reinitialize,
  };
}
