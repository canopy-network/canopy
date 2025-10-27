import { useState, useEffect } from "react";
import { Button, Form, InputGroup } from "react-bootstrap";
import {
  formatNumber,
  sanitizeNumberInput,
  numberFromCommas,
  sanitizeTextInput,
  withTooltip,
  toUCNPY,
  formatLocaleNumber,
} from "@/components/util";

// FormInputs() is a component that renders form inputs based on the fields passed to it
export default function FormInputs({ keygroup, account, validator, fields, show, onFieldChange }) {
  // Manage all form input values in a single state object to allow for dynamic form generation
  // and state management
  const [formValues, setFormValues] = useState({});

  // sets the default form values based on the fields every time the modal is opened
  useEffect(() => {
    const initialValues = fields.reduce((form, field) => {
      const value = field.defaultValue || "";
      form[field.label] =
        field.type === "number" || field.type === "currency" ? sanitizeNumberInput(value.toString()) : value;
      return form;
    }, {});

    setFormValues(initialValues);
  }, [show]);

  const handleInputChange = (key, value, type) => {
    const newValue =
      type === "number" || type === "currency"
        ? sanitizeNumberInput(value, type === "currency")
        : sanitizeTextInput(value);

    setFormValues((prev) => {
      return {
        ...prev,
        [key]: newValue,
      };
    });

    if (onFieldChange) {
      onFieldChange(key, value, newValue);
    }
  };

  const renderFormInputs = (input, i) => {
    if (input.label === "net_address" && (formValues["delegate"] === "true" || validator?.delegate === true))
      return null;

    if (input.type === "select") {
      return (
        <FormSelect
          input={input}
          key={input.label}
          idx={i}
          formValues={formValues[input.label]}
          onChange={handleInputChange}
        />
      );
    }
    return (
      <FormControl
        input={input}
        key={input.label}
        idx={i}
        formValues={formValues}
        onChange={handleInputChange}
        account={account}
      />
    );
  };

  return <>{fields.map(renderFormInputs)}</>;
}

const FormGroup = ({ input, children, subChildren, idx }) => (
  <Form.Group className="mb-3" key={idx}>
    <InputGroup size="lg">
      {withTooltip(
        <InputGroup.Text className="input-text">{input.inputText}</InputGroup.Text>,
        input.tooltip,
        input.index,
        "auto",
      )}
      {children}
    </InputGroup>
    {subChildren}
  </Form.Group>
);

const FormSelect = ({ onChange, input, value }) => {
  return (
    <FormGroup input={input}>
      <Form.Select
        className="input-text-field"
        onChange={(e) => onChange(input.label, e.target.value, input.type)}
        defaultValue={input.defaultValue}
        value={value}
        aria-label={input.label}
      >
        {input.options && Array.isArray(input.options) && input.options.length > 0 ? (
          input.options.map((key) => (
            <option key={key} value={key}>
              {key}
            </option>
          ))
        ) : (
          <option disabled>No options available</option>
        )}
      </Form.Select>
    </FormGroup>
  );
};

const FormControl = ({ input, formValues, onChange, account }) => {
  return (
    <FormGroup
      input={input}
      subChildren={
        input.type === "currency" &&
        input.displayBalance &&
        RenderAmountInput({
          amount: account.amount,
          input,
          onClick: onChange,
          inputValue: formValues[input.label],
        })
      }
    >
      <Form.Control
        className="input-text-field"
        onChange={(e) => onChange(input.label, e.target.value, input.type)}
        type={input.type == "number" || input.type == "currency" ? "text" : input.type}
        value={formValues[input.label]}
        placeholder={input.placeholder}
        required={input.required}
        min={0}
        minLength={input.minLength}
        maxLength={input.maxLength}
        aria-label={input.label}
        aria-describedby="emailHelp"
      />
    </FormGroup>
  );
};

// RenderAmountInput() renders the amount input with the option to set the amount to max
const RenderAmountInput = ({ amount, onClick, input, inputValue }) => {
  return (
    <div className="d-flex justify-content-between">
      <Form.Text className="text-start fw-bold">
        uCNPY: {formatLocaleNumber(toUCNPY(numberFromCommas(inputValue)))}
      </Form.Text>
      <Form.Text className="text-end">
        Available: <span className="fw-bold">{formatNumber(amount)} CNPY </span>
        <Button
          aria-label="max-button"
          onClick={() => {
            // Convert uCNPY to CNPY, then format it properly as a string
            const cnpyAmount = amount / 1000000;
            const formattedAmount = formatLocaleNumber(cnpyAmount, 6, 6);
            onClick(input.label, formattedAmount, input.type);
          }}
          variant="link"
          bsPrefix="max-amount-btn"
        >
          MAX
        </Button>
      </Form.Text>
    </div>
  );
};
