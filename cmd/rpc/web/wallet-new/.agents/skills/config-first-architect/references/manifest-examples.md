# Annotated Manifest Examples

Real-world examples across different domains showing config-first patterns in practice.

---

## Example 1: Dynamic Form Manifest (Checkout Flow)

```json
{
  "version": "1.0.0",
  "id": "checkout-shipping",
  "title": "Shipping Information",
  "fields": [
    {
      "id": "country",
      "type": "select",
      "label": "Country",
      "required": true,
      "options": {
        "type": "config",
        "key": "supportedCountries"   // resolved from system config at runtime
      }
    },
    {
      "id": "state",
      "type": "select",
      "label": "State / Province",
      "required": { "when": "country", "operator": "in", "value": ["US", "CA"] },
      "hidden": { "when": "country", "operator": "notIn", "value": ["US", "CA"] },
      "options": {
        "type": "api",
        "endpoint": "statesEndpoint",    // key resolved from config, not raw URL
        "transform": "countryStates"     // registered transform in engine
      },
      "dependsOn": ["country"]
    },
    {
      "id": "postalCode",
      "type": "text",
      "label": "Postal Code",
      "validation": [
        {
          "type": "pattern",
          "value": "^[0-9]{5}(-[0-9]{4})?$",
          "message": "Enter a valid US postal code",
          "when": { "field": "country", "operator": "eq", "value": "US" }
        }
      ]
    }
  ],
  "submission": {
    "endpoint": "checkoutApiEndpoint",
    "method": "POST",
    "payloadTransform": "checkoutShippingPayload",
    "onSuccess": "navigateToPayment",
    "onError": "showInlineError"
  }
}
```

**Key patterns shown:**
- Options sourced from config (not hardcoded arrays)
- Conditional required/hidden driven by other field values
- Validation rules with conditions
- Submission pointing to config keys, not raw URLs

---

## Example 2: Multi-Step Flow Manifest

```json
{
  "version": "1.0.0",
  "id": "onboarding-flow",
  "title": "User Onboarding",
  "steps": [
    {
      "id": "profile",
      "title": "Your Profile",
      "form": "onboarding-profile-form",   // references a form manifest
      "guards": [],
      "transitions": {
        "next": "preferences",
        "skip": { "target": "finish", "condition": { "featureFlag": "allowSkipOnboarding" } }
      }
    },
    {
      "id": "preferences",
      "title": "Your Preferences",
      "form": "onboarding-preferences-form",
      "guards": [
        { "permission": "canSetPreferences", "fallback": "finish" }
      ],
      "transitions": {
        "next": "finish",
        "back": "profile"
      }
    },
    {
      "id": "finish",
      "type": "confirmation",
      "title": "All done!",
      "actions": [
        { "type": "dispatch", "key": "completeOnboarding" }
      ]
    }
  ]
}
```

**Key patterns shown:**
- Step transitions are data, not code
- Feature flags control flow variations
- Permission guards with fallback targets
- Forms are referenced by ID, not embedded

---

## Example 3: System Configuration

```json
{
  "version": "1.0.0",
  "environment": "production",
  "features": {
    "allowSkipOnboarding": false,
    "experimentalDashboard": true,
    "maxUploadSizeMb": 10
  },
  "endpoints": {
    "checkoutApiEndpoint": "https://api.example.com/v2/checkout",
    "statesEndpoint": "https://api.example.com/v1/geo/states",
    "authEndpoint": "https://auth.example.com/token"
  },
  "supportedCountries": ["US", "CA", "MX", "GB", "DE"],
  "permissions": {
    "canSetPreferences": ["admin", "user"],
    "canSkipOnboarding": ["admin"]
  },
  "theme": {
    "primaryColor": "#2563EB",
    "borderRadius": "0.5rem",
    "fontFamily": "Inter, sans-serif"
  }
}
```

**Key patterns shown:**
- Feature flags as first-class citizens
- All endpoints centralized (single place to change)
- Permissions defined declaratively
- Theme tokens in config (not in CSS or components)

---

## Example 4: Plugin / Extension Manifest

```json
{
  "version": "1.0.0",
  "id": "crm-integration",
  "type": "integration-plugin",
  "triggers": ["form.submit", "user.created"],
  "actions": [
    {
      "on": "form.submit",
      "filter": { "formId": "contact-form" },
      "execute": {
        "type": "http",
        "endpoint": "crmWebhookEndpoint",
        "method": "POST",
        "payloadTransform": "crmContactPayload"
      }
    }
  ],
  "requiredConfig": ["crmWebhookEndpoint"],
  "optionalConfig": ["crmTagPrefix"]
}
```

**Key patterns shown:**
- Plugin behavior described entirely in manifest
- Event-based triggers (decoupled)
- Required config declared (engine can validate at boot)
- No code in the manifest — pure declarative intent

---

## Reading These Examples

Notice what's **absent** in every manifest:
- No `if/else` statements
- No function references
- No hardcoded URLs or values
- No UI concerns (colors, classes, layout details)

And what's always **present**:
- `version` field
- References to config keys (not raw values)
- Declarative intent ("what", never "how")
- IDs that enable referencing from other manifests
