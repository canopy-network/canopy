// cnpyConversionRate sets the conversion rate between CNPY and uCNPY
export const cnpyConversionRate = 1_000_000;

// toCNPY converts a uCNPY amount to CNPY
export function toCNPY(uCNPY: number): number {
    return uCNPY / cnpyConversionRate;
}

// toUCNPY converts a CNPY amount to uCNPY
export function toUCNPY(cnpy: number): number {
    return cnpy * cnpyConversionRate;
}

// convertNumberWCommas() formats a number with commas
export function convertNumberWCommas(x: number): string {
    return x.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

// convertNumber() formats a number with commas or in compact notation
export function convertNumber(nString: string | number, cutoff: number = 1000000, convertToCNPY: boolean = false): string {
    if (convertToCNPY) {
        nString = toCNPY(Number(nString)).toString();
    }

    if (Number(nString) < cutoff) {
        return convertNumberWCommas(Number(nString));
    }
    return Intl.NumberFormat("en", { notation: "compact", maximumSignificantDigits: 3 }).format(Number(nString));
}

// addMS() adds milliseconds to a Date object
declare global {
    interface Date {
        addMS(s: number): Date;
    }
}

Date.prototype.addMS = function (s: number): Date {
    this.setTime(this.getTime() + s);
    return this;
};

// addDate() adds a duration to a date and returns the result as a time string
export function addDate(value: number, duration: number): string {
    const milliseconds = Math.floor(value / 1000);
    const date = new Date(milliseconds);
    return date.addMS(duration).toLocaleTimeString();
}

// convertBytes() converts a byte value to a human-readable format
export function convertBytes(a: number, b: number = 2): string {
    if (!+a) return "0 Bytes";
    const c = 0 > b ? 0 : b,
        d = Math.floor(Math.log(a) / Math.log(1024));
    return `${parseFloat((a / Math.pow(1024, d)).toFixed(c))} ${["B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"][d]}`;
}

// convertTime() converts a timestamp to a time string
export function convertTime(value: number): string {
    const date = new Date(Math.floor(value / 1000));
    return date.toLocaleTimeString();
}

// convertIfTime() checks if the key is related to time and converts it if true
export function convertIfTime(key: string, value: any): any {
    if (key.includes("time")) {
        return convertTime(value);
    }
    if (typeof value === "boolean") {
        return String(value);
    }
    return value;
}

// convertIfNumber() attempts to convert a string to a number
export function convertIfNumber(str: string): number | string {
    if (!isNaN(Number(str)) && !isNaN(parseFloat(str))) {
        return Number(str);
    } else {
        return str;
    }
}

// isNumber() checks if the value is a number
export function isNumber(n: any): boolean {
    return !isNaN(parseFloat(n)) && !isNaN(n - 0);
}

// isHex() checks if the string is a valid hex color code
export function isHex(h: string): boolean {
    if (isNumber(h)) {
        return false;
    }
    let hexRe = /[0-9A-Fa-f]{6}/g;
    return hexRe.test(h);
}

// upperCaseAndRepUnderscore() capitalizes each word in a string and replaces underscores with spaces
export function upperCaseAndRepUnderscore(str: string): string {
    let i: number,
        frags = str.split("_");
    for (i = 0; i < frags.length; i++) {
        frags[i] = frags[i].charAt(0).toUpperCase() + frags[i].slice(1);
    }
    return frags.join(" ");
}

// cpyObj() creates a shallow copy of an object
export function cpyObj<T>(v: T): T {
    return Object.assign({}, v);
}

// isEmpty() checks if an object is empty
export function isEmpty(obj: object): boolean {
    return Object.keys(obj).length === 0;
}

// copy() copies text to clipboard and triggers a toast notification
export function copy(state: any, setState: (state: any) => void, detail: string, toastText: string = "Copied!"): void {
    if (navigator.clipboard && window.isSecureContext) {
        // if HTTPS - use Clipboard API
        navigator.clipboard
            .writeText(detail)
            .then(() => setState({ ...state, toast: toastText }))
            .catch(() => fallbackCopy(state, setState, detail, toastText));
    } else {
        fallbackCopy(state, setState, detail, toastText);
    }
}

// fallbackCopy() copies text to clipboard if clipboard API is unavailable
export function fallbackCopy(state: any, setState: (state: any) => void, detail: string, toastText: string = "Copied!"): void {
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
export function convertTx(tx: any): any {
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
export const formatLocaleNumber = (num: number, minFractionDigits: number = 0, maxFractionDigits: number = 2): string => {
    if (isNaN(num)) {
        return "0";
    }

    return num.toLocaleString("en-US", {
        maximumFractionDigits: maxFractionDigits,
        minimumFractionDigits: minFractionDigits,
    });
};
