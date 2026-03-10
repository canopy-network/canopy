import { TextField } from "./TextField";
import { AmountField } from "./AmountField";
import { NumberField } from "./NumberField";
import { RangeField } from "./RangeField";
import { AddressField } from "./AddressField";
import { SelectField } from "./SelectField";
import { AdvancedSelectField } from "./AdvancedSelectField";
import { SwitchField } from "./SwitchField";
import { OptionField } from "./OptionField";
import { OptionCardField } from "./OptionCardField";
import { TableSelectField } from "./TableSelectField";
import { DynamicHtmlField } from "./DynamicHtmlField";
import { SectionField } from "./SectionField";
import { DividerField } from "./DividerField";
import { SpacerField } from "./SpacerField";
import { HeadingField } from "./HeadingField";
import { CollapsibleGroupField } from "./CollapsibleGroupField";
import type { FC } from "react";

type FieldComponent = FC<any>;

/**
 * Central registry for all field types used in the manifest-driven forms.
 * Maps field type strings to their corresponding React components.
 *
 * IMPORTANT: All imports must be kept even if the linter marks them as unused.
 * These components are used dynamically based on manifest configuration.
 */
export const fieldRegistry: Record<string, FieldComponent> = {
    text: TextField,
    textarea: TextField,
    amount: AmountField,
    number: NumberField,
    range: RangeField,
    address: AddressField,
    select: SelectField,
    advancedSelect: AdvancedSelectField,
    switch: SwitchField,
    option: OptionField,
    optionCard: OptionCardField,
    tableSelect: TableSelectField,
    dynamicHtml: DynamicHtmlField,
    // Layout and structural fields - DO NOT REMOVE
    section: SectionField,
    divider: DividerField,
    spacer: SpacerField,
    heading: HeadingField,
    collapsibleGroup: CollapsibleGroupField,
};

/**
 * Gets the renderer component for a given field type
 * @param fieldType - The type of field to render (e.g., "text", "amount", "section")
 * @returns The field component or null if not found
 */
export const getFieldRenderer = (fieldType: string): FieldComponent | null => {
    return fieldRegistry[fieldType] || null;
};
