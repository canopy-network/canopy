import { TextField } from './TextField';
import { AmountField } from './AmountField';
import { AddressField } from './AddressField';
import { SelectField } from './SelectField';
import { AdvancedSelectField } from './AdvancedSelectField';
import { SwitchField } from './SwitchField';
import { OptionField } from './OptionField';
import { OptionCardField } from './OptionCardField';
import { TableSelectField } from './TableSelectField';
import { DynamicHtmlField } from './DynamicHtmlField';
export const fieldRegistry = {
    text: TextField,
    textarea: TextField,
    amount: AmountField,
    address: AddressField,
    select: SelectField,
    advancedSelect: AdvancedSelectField,
    switch: SwitchField,
    option: OptionField,
    optionCard: OptionCardField,
    tableSelect: TableSelectField,
    dynamicHtml: DynamicHtmlField,
};
export const getFieldRenderer = (fieldType) => {
    return fieldRegistry[fieldType] || null;
};
