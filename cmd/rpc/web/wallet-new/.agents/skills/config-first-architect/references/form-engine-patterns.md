# Dynamic Form Engine Design Patterns

## Core Concept

A form engine takes a **manifest** (JSON/YAML) and produces a functional form.
The engine handles: rendering, validation, conditional logic, state, submission.
Components know nothing about business rules.

---

## Field Schema Contract

```typescript
interface FieldSchema {
  id: string
  type: 'text' | 'number' | 'select' | 'date' | 'boolean' | 'file' | 'group' | 'repeat'
  label: string
  placeholder?: string
  defaultValue?: unknown
  required?: boolean | ConditionalExpression
  disabled?: boolean | ConditionalExpression
  hidden?: boolean | ConditionalExpression
  validation?: ValidationRule[]
  options?: OptionSource          // for select/radio/checkbox
  dependsOn?: string[]            // fields this field reacts to
  meta?: Record<string, unknown>  // renderer hints (layout, icons, etc.)
}

interface ConditionalExpression {
  when: string        // field id
  operator: 'eq' | 'neq' | 'gt' | 'lt' | 'in' | 'notIn' | 'exists'
  value: unknown
}

interface ValidationRule {
  type: 'required' | 'min' | 'max' | 'pattern' | 'custom' | 'async'
  value?: unknown
  message: string
  ref?: string        // for 'custom': points to registered validator key
}
```

---

## Form Manifest Contract

```typescript
interface FormManifest {
  version: string           // semver, e.g. "1.0.0"
  id: string
  title?: string
  fields: FieldSchema[]
  layout?: LayoutDescriptor
  actions: ActionDescriptor[]
  submission: SubmissionDescriptor
}

interface SubmissionDescriptor {
  endpoint: string          // reference to config endpoint key, not raw URL
  method: 'POST' | 'PUT' | 'PATCH'
  payloadTransform?: string // optional transform key registered in engine
  onSuccess: string         // action key: 'redirect', 'notify', 'reset'
  onError: string           // action key
}
```

---

## Engine Implementation Pattern

```typescript
// The engine is a pure function pipeline
class FormEngine {
  constructor(
    private config: SystemConfig,
    private validators: ValidatorRegistry,
    private actions: ActionRegistry,
  ) {}

  parse(raw: unknown): FormManifest {
    // 1. Validate against JSON Schema / Zod
    // 2. Normalize defaults
    // 3. Resolve endpoint references from config
    return parsed
  }

  resolve(manifest: FormManifest, context: FormContext): ResolvedForm {
    // 1. Evaluate all conditional expressions with current field values
    // 2. Resolve dynamic options (from API or config)
    // 3. Apply permission guards
    return resolvedForm
  }

  validate(values: Record<string, unknown>, manifest: FormManifest): ValidationResult {
    // 1. Run field-level validators
    // 2. Run cross-field validators
    // 3. Return structured errors
    return { valid: boolean, errors: FieldErrors }
  }

  async submit(values: Record<string, unknown>, manifest: FormManifest): Promise<SubmitResult> {
    // 1. Final validation pass
    // 2. Apply payload transform if defined
    // 3. Resolve endpoint from config
    // 4. Execute HTTP action
    // 5. Run success/error action
  }
}
```

---

## Conditional Logic Evaluation

Never use `eval()`. Use a safe expression evaluator:

```typescript
function evaluateCondition(
  expr: ConditionalExpression,
  values: Record<string, unknown>
): boolean {
  const fieldValue = values[expr.when]
  switch (expr.operator) {
    case 'eq': return fieldValue === expr.value
    case 'neq': return fieldValue !== expr.value
    case 'in': return Array.isArray(expr.value) && expr.value.includes(fieldValue)
    case 'exists': return fieldValue !== undefined && fieldValue !== null && fieldValue !== ''
    // ...
  }
}
```

---

## Common Patterns

### Field Dependency Graph
Build a DAG from `dependsOn` fields to determine re-evaluation order.
When field A changes, only re-resolve fields that depend on A.

### Option Sources
```typescript
type OptionSource =
  | { type: 'static'; items: Option[] }
  | { type: 'config'; key: string }           // from system config
  | { type: 'api'; endpoint: string; transform: string }  // endpoint = config key
  | { type: 'context'; path: string }         // from form context/state
```

### Payload Transforms
Register named transforms in the engine, reference by key in manifest:
```typescript
engine.registerTransform('snakeCase', (values) => toSnakeCase(values))
engine.registerTransform('dateFormat', (values) => formatDates(values))
```

### Repeatable Groups
Fields of type `repeat` render N instances of a sub-schema.
Store as array in form state. Each instance is an independent field group.

---

## Anti-Patterns to Avoid

❌ `if (formId === 'checkout') { ... }` — hardcoded form-specific logic
❌ Fetching endpoints directly in components — use config keys
❌ Validation logic in components — belongs in engine
❌ `eval()` or `new Function()` for conditionals — security risk
❌ Mixing form state with app state — keep isolated
