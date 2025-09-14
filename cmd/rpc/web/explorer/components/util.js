import React from "react";
import { Tooltip, OverlayTrigger, Pagination } from "react-bootstrap";

// cnpyConversionRate sets the conversion rate between CNPY and uCNPY
export const cnpyConversionRate = 1_000_000;
export const defaultNetAddress = "tcp://delegating";

// toCNPY converts a uCNPY amount to CNPY
export function toCNPY(uCNPY) {
  return uCNPY / cnpyConversionRate;
}

// toUCNPY converts a CNPY amount to uCNPY
export function toUCNPY(cnpy) {
  return cnpy * cnpyConversionRate;
}

// convertNumberWCommas() formats a number with commas
export function convertNumberWCommas(x) {
  return x.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

// convertNumber() formats a number with commas or in compact notation
export function convertNumber(nString, cutoff = 1000000, convertToCNPY = false) {
  if (convertToCNPY) {
    nString = toCNPY(nString).toString();
  }

  if (Number(nString) < cutoff) {
    return convertNumberWCommas(nString);
  }
  return Intl.NumberFormat("en", { notation: "compact", maximumSignificantDigits: 3 }).format(nString);
}

// addMS() adds milliseconds to a Date object
Date.prototype.addMS = function (s) {
  this.setTime(this.getTime() + s);
  return this;
};

// addDate() adds a duration to a date and returns the result as a time string
export function addDate(value, duration) {
  const milliseconds = Math.floor(value / 1000);
  const date = new Date(milliseconds);
  return date.addMS(duration).toLocaleTimeString();
}

// convertBytes() converts a byte value to a human-readable format
export function convertBytes(a, b = 2) {
  if (!+a) return "0 Bytes";
  const c = 0 > b ? 0 : b,
    d = Math.floor(Math.log(a) / Math.log(1024));
  return `${parseFloat((a / Math.pow(1024, d)).toFixed(c))} ${["B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"][d]}`;
}

// convertTime() converts a timestamp to a time string
export function convertTime(value) {
  const date = new Date(Math.floor(value / 1000));
  return date.toLocaleTimeString();
}

// convertIfTime() checks if the key is related to time and converts it if true
export function convertIfTime(key, value) {
  if (key.includes("time")) {
    return convertTime(value);
  }
  if (typeof value === "boolean") {
    return String(value);
  }
  return value;
}

// convertIfNumber() attempts to convert a string to a number
export function convertIfNumber(str) {
  if (!isNaN(str) && !isNaN(parseFloat(str))) {
    return Number(str);
  } else {
    return str;
  }
}

// withTooltip() adds a tooltip to an element
export function withTooltip(obj, text, key, dir = "right") {
  return (
    <OverlayTrigger
      key={key}
      placement={dir}
      delay={{ show: 250, hide: 400 }}
      overlay={<Tooltip id="button-tooltip">{text}</Tooltip>}
    >
      {obj}
    </OverlayTrigger>
  );
}

// isNumber() checks if the value is a number
export function isNumber(n) {
  return !isNaN(parseFloat(n)) && !isNaN(n - 0);
}

// isHex() checks if the string is a valid hex color code
export function isHex(h) {
  if (isNumber(h)) {
    return false;
  }
  let hexRe = /[0-9A-Fa-f]{6}/g;
  return hexRe.test(h);
}

// upperCaseAndRepUnderscore() capitalizes each word in a string and replaces underscores with spaces
export function upperCaseAndRepUnderscore(str) {
  let i,
    frags = str.split("_");
  for (i = 0; i < frags.length; i++) {
    frags[i] = frags[i].charAt(0).toUpperCase() + frags[i].slice(1);
  }
  return frags.join(" ");
}

// cpyObj() creates a shallow copy of an object
export function cpyObj(v) {
  return Object.assign({}, v);
}

// pagination() generates pagination controls for a given data set
export function pagination(data, callback) {
  let pageSquares = [];
  if ("perPage" in data) {
    let start = data.pageNumber - 2;
    if (start <= 0) {
      start = 1;
    }
    for (let i = start; i <= Math.min(Math.ceil(data.totalPages), start + 5); i++) {
      pageSquares.push(
        <Pagination.Item key={i} onClick={() => callback(i)} active={i === data.pageNumber}>
          {i}
        </Pagination.Item>,
      );
    }
  }
  return (
    <Pagination className="pagination">
      {pageSquares}
      <Pagination.Ellipsis />
    </Pagination>
  );
}

// isEmpty() checks if an object is empty
export function isEmpty(obj) {
  return Object.keys(obj).length === 0;
}

// copy() copies text to clipboard and triggers a toast notification
export function copy(state, setState, detail, toastText = "Copied!") {
  if (navigator.clipboard && window.isSecureContext) {
    // if HTTPS - use Clipboard API
    navigator.clipboard
      .writeText(detail)
      .then(() => setState({ ...state, toast: toastText }))
      .catch(() => fallbackCopy(detail, state, setState, toastText));
  } else {
    fallbackCopy(state, setState, detail, toastText);
  }
}

// fallbackCopy() copies text to clipboard if clipboard API is unavailable
export function fallbackCopy(state, setState, detail, toastText = "Copied!") {
  // if http - use textarea
  const textArea = document.createElement("textarea");
  textArea.value = detail;
  document.body.appendChild(textArea);
  textArea.select();
  try {
    document.execCommand("copy");
    setState({ ...state, toast: toastText });
  } catch (err) {
    console.error("Fallback copy failed", err);
    setState({ ...state, toast: "Clipboard access denied" });
  }
  document.body.removeChild(textArea);
}

// convertTx() sanitizes and simplifies a transaction object
export function convertTx(tx) {
  if (tx.recipient == null) {
    tx.recipient = tx.sender;
  }
  if (!("index" in tx) || tx.index === 0) {
    tx.index = 0;
  }
  tx = JSON.parse(
    JSON.stringify(tx, ["sender", "recipient", "messageType", "height", "index", "txHash", "fee", "sequence"], 4),
  );
  return tx;
}

// formatLocaleNumber formats a number with the default en-us configuration
export const formatLocaleNumber = (num, minFractionDigits = 0, maxFractionDigits = 2) => {
  if (isNaN(num)) {
    return 0;
  }

  return num.toLocaleString("en-US", {
    maximumFractionDigits: maxFractionDigits,
    minimumFractionDigits: minFractionDigits,
  });
};
