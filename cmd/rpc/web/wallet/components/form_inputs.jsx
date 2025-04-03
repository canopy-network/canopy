import { useState, useEffect, forwardRef, useRef, Children } from "react";
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
          onInputChange={handleInputChange}
          account={account}
          options={input.defaultValue}
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

const MultiSelectToggle = forwardRef(({ value, onChange, placeholder, onClick, input }, ref) => (
  <Form.Control
    ref={ref}
    type="text"
    value={value}
    placeholder={placeholder}
    onChange={onChange}
    onClick={(e) => {
      e.preventDefault();
      e.stopPropagation();
      onClick(e);
    }}
    onFocus={(e) => {
      e.preventDefault();
      e.stopPropagation();
      onClick(e);
    }}
    aria-label={input.label}
    aria-describedby="emailHelp"
  />
));

// Custom Menu: Renders a search field (with its own local state) and filters dropdown items.
const MultiSelectMenu = forwardRef(({ children, style, className, "aria-labelledby": labeledBy }, ref) => {
  const [search, setSearch] = useState("");

  // Recursively extract text from a child node.
  const extractText = (child) => {
    if (typeof child === "string") return child;
    if (Array.isArray(child)) return child.map(extractText).join("");
    if (child && child.props && child.props.children) return extractText(child.props.children);
    return "";
  };

  const filteredChildren = Children.toArray(children).filter((child) => {
    const text = extractText(child);
    return !search || text.toLowerCase().includes(search.toLowerCase());
  });

  return (
    <div ref={ref} style={style} className={className} aria-labelledby={labeledBy}>
      <Form.Control
        autoFocus
        className="mx-3 my-2 w-auto"
        placeholder="Type to filter..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        aria-label="multiselectsearch"
      />
      <ul className="list-unstyled mb-0">{filteredChildren}</ul>
    </div>
  );
});

// FormMultiSelect
// Props:
//  - options: A comma-separated string of available options (ex: "1,2,3").
//  - placeholder: Placeholder text for the input.
//  - validate: Optional custom validation function (if not provided, a default function is used).
//  - onInputChange: Optional external callback to handle input changes.
const FormMultiSelect = ({ options, placeholder, validate, onInputChange, input }) => {
  // Internal state: raw input, parsed selections, error message, and dropdown open state.
  const [inputValue, setInputValue] = useState("");
  const [selectedValues, setSelectedValues] = useState([]);
  const [errorMsg, setErrorMsg] = useState("");
  const [show, setShow] = useState(false);

  // Parse available options from a comma-separated string.
  const availableOptions = options
    .split(",")
    .map((opt) => opt.trim())
    .filter((opt) => opt);

  // Default parse and validate function: splits input, trims values, and verifies each value is valid against options list.
  const defaultParseAndValidate = (input, availableOpts) => {
    const values = input
      .split(",")
      .map((v) => v.trim())
      .filter((v) => v);
    const invalid = values.filter((v) => !availableOpts.includes(v));
    let error = "";
    if (invalid.length > 0) {
      error = `Invalid option(s): ${invalid.join(", ")}`;
    }
    return { values, error };
  };

  // Use the provided validate function if available. Otherwise, use the default.
  const parseAndValidate = validate || defaultParseAndValidate;

  // Handle input change from the custom toggle.
  const handleInputChange = (e) => {
    const rawValue = e.target.value;
    setInputValue(rawValue);

    const { values, error } = parseAndValidate(rawValue, availableOptions);
    setSelectedValues(values);
    setErrorMsg(error);

    const sanitizedValue = values.join(",");
    if (onInputChange) {
      onInputChange(input.label, sanitizedValue, input.type);
    }
  };

  // Handle checkbox toggles in the dropdown.
  const handleCheckboxChange = (option, isChecked, e) => {
    e.stopPropagation();
    let updated;
    if (isChecked) {
      if (selectedValues.includes(option)) return;
      updated = [...selectedValues, option];
    } else {
      updated = selectedValues.filter((v) => v !== option);
    }
    setSelectedValues(updated);
    const newInput = updated.join(",");
    setInputValue(newInput);
    setErrorMsg("");
    if (onInputChange) {
      onInputChange(input.label, newInput, input.type);
    }
  };

  // Group available options into selected and unselected (both sorted ascending).
  const selectedOptions = availableOptions.filter((opt) => selectedValues.includes(opt)).sort();
  const unselectedOptions = availableOptions.filter((opt) => !selectedValues.includes(opt)).sort();

  return (
    <>
      {errorMsg && <div className="text-danger mb-2">{errorMsg}</div>}
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
        {/* <div> */}
        <Dropdown show={show} onToggle={(nextShow) => setShow(nextShow)} autoClose="outside">
          <Dropdown.Toggle
            as={MultiSelectToggle}
            value={inputValue}
            onChange={handleInputChange}
            placeholder={placeholder || "Enter values"}
            onClick={(e) => {
              setShow(true), e.stopPropagation(), e.preventDefault();
            }}
            input={input}
          />
          <Dropdown.Menu as={MultiSelectMenu}>
            {/* Map dropdown items that are selected */}
            {selectedOptions.map((opt) => (
              <Dropdown.Item key={opt} eventKey={opt} className="d-flex justify-content-between align-items-center">
                <span>{opt}</span>
                <Form.Check
                  type="checkbox"
                  checked={true}
                  onChange={(e) => handleCheckboxChange(opt, e.target.checked, e)}
                  onClick={(e) => e.stopPropagation()}
                  aria-label="multicheckbox"
                />
              </Dropdown.Item>
            ))}

            {/* Divider between selected and unselected */}
            {selectedOptions.length > 0 && unselectedOptions.length > 0 && <Dropdown.Divider key="divider" />}

            {/* Map dropdown items that are unselected */}
            {unselectedOptions.map((opt) => (
              <Dropdown.Item key={opt} eventKey={opt} className="d-flex justify-content-between align-items-center">
                <span>{opt}</span>
                <Form.Check
                  type="checkbox"
                  checked={false}
                  onChange={(e) => handleCheckboxChange(opt, e.target.checked, e)}
                  onClick={(e) => e.stopPropagation()}
                  aria-label="multicheckbox"
                />
              </Dropdown.Item>
            ))}
          </Dropdown.Menu>
        </Dropdown>
        {/* </div> */}
      </FormGroup>
    </>
  );
};
