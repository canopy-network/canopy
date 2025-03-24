import { useState, useEffect, forwardRef } from "react";
import { Button, Form, InputGroup, Dropdown, FormControl as MultiSelectControl, Alert } from "react-bootstrap";
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
    if (input.type === "multiselect") {
      return (
        <FormMultiSelect
          input={input}
          key={input.label}
          idx={i}
          formValues={formValues}
          onChange={handleInputChange}
          account={account}
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
          onClick={() => onClick(input.label, Math.ceil(amount).toString(), input.type)}
          variant="link"
          bsPrefix="max-amount-btn"
        >
          MAX
        </Button>
      </Form.Text>
    </div>
  );
};

// Custom input toggle component for the multi select dropdown
const CustomToggleInput = forwardRef(({ value, onChange, onFocus }, ref) => (
  <MultiSelectControl
    ref={ref}
    value={value}
    onChange={onChange}
    onFocus={onFocus}
    onClick={(e) => {
      e.preventDefault();
      e.stopPropagation();
    }}
    className="input-text-field"
  />
));

// Multi select component
const FormMultiSelect = ({ input, validate, placeholder }) => {
  const [selectedValues, setSelectedValues] = useState([]);
  const [inputValue, setInputValue] = useState("");
  const [error, setError] = useState("");
  const [searchTerm, setSearchTerm] = useState("");
  const [showDropdown, setShowDropdown] = useState(false);

  // Pulling default values from inputs to use as options
  const options = input.defaultValue.split(",") || "";

  // Filter options based on searchTerm (case-insensitive)
  const getFilteredOptions = () => {
    return options.filter((opt) => opt.toLowerCase().includes(searchTerm.toLowerCase())).sort();
  };

  // Update input value and selected values based on user input
  const handleInputChange = (e) => {
    const val = e.target.value;
    setInputValue(val);
    setError("");
    const values = val
      .split(",")
      .map((v) => v.trim())
      .filter((v) => v !== "");
    const validationMessage = validate(values, options);
    if (validationMessage) {
      setError(validationMessage);
      return;
    }
    setSelectedValues(values);
  };

  // Handle checkbox toggling in the dropdown menu
  const handleCheckboxChange = (option, isChecked, e) => {
    e.stopPropagation();
    let updatedSelected;
    if (isChecked) {
      if (selectedValues.includes(option)) {
        alert(`"${option}" is already selected.`);
        return;
      }
      updatedSelected = [...selectedValues, option];
    } else {
      updatedSelected = selectedValues.filter((v) => v !== option);
    }
    setSelectedValues(updatedSelected);
    setInputValue(updatedSelected.join(", "));
    setError("");
  };

  const filteredOptions = getFilteredOptions();
  const selectedOptions = filteredOptions.filter((opt) => selectedValues.includes(opt));
  const unselectedOptions = filteredOptions.filter((opt) => !selectedValues.includes(opt));

  return (
    <FormGroup input={input}>
      {error && <Alert variant="danger">{error}</Alert>}
      <Dropdown autoClose="outside" show={showDropdown} onToggle={setShowDropdown}>
        <Dropdown.Toggle
          as={CustomToggleInput}
          value={inputValue}
          onChange={handleInputChange}
          placeholder={placeholder || "Enter values"}
          onFocus={() => setShowDropdown(true)}
        />
        <Dropdown.Menu className="p-0">
          {/* Search field inside dropdown */}
          <MultiSelectControl
            placeholder="Search..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
          <div className="overflow-auto" style={{ maxHeight: "200px" }}>
            {selectedOptions.length > 0 && (
              <>
                {selectedOptions.map((option) => (
                  <Dropdown.Item key={option} as="div" className="px-3 py-2">
                    <div className="d-flex justify-content-between align-items-center">
                      <span>{option}</span>
                      <Form.Check
                        type="checkbox"
                        id={`chk-${option}`}
                        checked={true}
                        onChange={(e) => handleCheckboxChange(option, e.target.checked, e)}
                        onClick={(e) => e.stopPropagation()}
                      />
                    </div>
                  </Dropdown.Item>
                ))}
                <Dropdown.Divider />
              </>
            )}
            {unselectedOptions.map((option) => (
              <Dropdown.Item key={option} className="px-3 py-2">
                <div className="d-flex justify-content-between align-items-center">
                  <span>{option}</span>
                  <Form.Check
                    type="checkbox"
                    id={`chk-${option}`}
                    checked={false}
                    onChange={(e) => handleCheckboxChange(option, e.target.checked, e)}
                    onClick={(e) => e.stopPropagation()}
                  />
                </div>
              </Dropdown.Item>
            ))}
          </div>
        </Dropdown.Menu>
      </Dropdown>
    </FormGroup>
  );
};
