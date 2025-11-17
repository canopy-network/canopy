import React from "react";
import { Field } from "@/manifest/types";
import { TextField } from "./TextField";
import { AmountField } from "./AmountField";
import { AddressField } from "./AddressField";
import { SelectField } from "./SelectField";
import { AdvancedSelectField } from "./AdvancedSelectField";
import { SwitchField } from "./SwitchField";
import { OptionField } from "./OptionField";
import { OptionCardField } from "./OptionCardField";
import { TableSelectField } from "./TableSelectField";
import { DynamicHtmlField } from "./DynamicHtmlField";

type FieldRenderer = React.FC<{
  field: Field;
  value: any;
  error?: string;
  errors?: Record<string, string>;
  templateContext: Record<string, any>;
  dsValue?: any;
  onChange: (value: any) => void;
  resolveTemplate: (s?: any) => any;
  setVal?: (fieldId: string, v: any) => void;
}>;

export const fieldRegistry: Record<string, FieldRenderer> = {
  text: TextField,
  textarea: TextField,
  amount: AmountField,
  address: AddressField,
  select: SelectField,
  advancedSelect: AdvancedSelectField,
  switch: SwitchField,
  option: OptionField,
  optionCard: OptionCardField,
  tableSelect: TableSelectField as any,
  dynamicHtml: DynamicHtmlField,
};

export const getFieldRenderer = (fieldType: string): FieldRenderer | null => {
  return fieldRegistry[fieldType] || null;
};
