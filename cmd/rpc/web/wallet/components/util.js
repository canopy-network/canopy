import {OverlayTrigger, Toast, ToastContainer, Tooltip} from "react-bootstrap";

// getFormInputs() returns the form input based on the type
// account and validator is passed to assist with auto fill
export function getFormInputs(type, keyGroup, account, validator, keyStore) {
    let amount = null;
    let netAddr = validator && validator.address ? validator.netAddress : "";
    let delegate = validator && validator.address ? validator.delegate : false;
    let compound = validator && validator.address ? validator.compound : false;
    let output = validator && validator.address ? validator.output : "";
    let defaultNick = account != null ? account.nickname : "";
    let defaultNickSigner = account != null ? account.nickname : "";
    let committeeList =
        validator && validator.address && validator.committees && validator.committees.length !== 0
            ? validator.committees.join(",")
            : "";
    defaultNick = type !== "send" && validator && validator.nickname ? validator.nickname : defaultNick;
    defaultNick = type === "stake" && validator && validator.nickname ? "WARNING: validator already staked" : defaultNick;
    if (type === "edit-stake" || type === "stake") {
        amount = validator && validator.address ? validator.stakedAmount : null;
    }
    let a = {
        privateKey: {
            placeholder: "opt: private key hex to import",
            defaultValue: "",
            tooltip: "the raw private key to import if blank - will generate a new key",
            label: "private_key",
            inputText: "key",
            feedback: "please choose a private key to import",
            required: false,
            type: "password",
            minLength: 64,
            maxLength: 128,
        },
        account: {
            tooltip: "the short public key id or nickname of the account",
            label: "sender",
            defaultValue: defaultNick,
            inputText: "account",
            type: "select",
            options: keyStore ? Object.keys(keyStore) : [],
        },
        committees: {
            placeholder: "1, 22, 50",
            defaultValue: committeeList,
            tooltip: "comma separated list of committee chain IDs to stake for",
            label: "committees",
            inputText: "committees",
            feedback: "please input atleast 1 committee",
            required: true,
            type: "text",
            minLength: 1,
            maxLength: 200,
        },
        netAddr: {
            placeholder: "url of the node",
            defaultValue: netAddr,
            tooltip: "the url of the validator for consensus and polling",
            label: "net_address",
            inputText: "net-addr",
            feedback: "please choose a net address for the validator",
            required: true,
            type: "text",
            minLength: 5,
            maxLength: 50,
        },
        earlyWithdrawal: {
            placeholder: "early withdrawal rewards for 20% penalty",
            defaultValue: !compound,
            tooltip: "validator NOT reinvesting their rewards to their stake, incurring a 20% penalty",
            label: "earlyWithdrawal",
            inputText: "withdrawal",
            feedback: "please choose if your validator to earlyWithdrawal or not",
            required: true,
            type: "text",
            minLength: 4,
            maxLength: 5,
        },
        delegate: {
            placeholder: "validator delegation status",
            defaultValue: (delegate || "true"),
            tooltip:
                "validator is passively delegating rather than actively validating. NOTE: THIS FIELD IS FIXED AND CANNOT BE UPDATED WITH EDIT-STAKE",
            label: "delegate",
            inputText: "delegate",
            feedback: "please choose if your validator is delegating or not",
            required: true,
            type: "select",
            options: ["true", "false"],
        },
        rec: {
            placeholder: "recipient of the tx",
            defaultValue: "",
            tooltip: "the recipient of the transaction",
            label: "recipient",
            inputText: "recipient",
            feedback: "please choose a recipient for the transaction",
            required: true,
            type: "text",
            minLength: 40,
            maxLength: 40,
        },
        amount: {
            placeholder: "amount value for the tx in CNPY",
            defaultValue: amount,
            tooltip: "the amount of currency being sent / sold",
            label: "amount",
            inputText: "amount",
            feedback: "please choose an amount for the tx",
            required: true,
            type: "currency",
            minLength: 1,
            maxLength: 100,
            displayBalance: true,
        },
        receiveAmount: {
            placeholder: "amount of counter asset to receive",
            defaultValue: amount,
            tooltip: "the amount of counter asset being received",
            label: "receiveAmount",
            inputText: "rec-amount",
            feedback: "please choose a receive amount for the tx",
            required: true,
            type: "currency",
            minLength: 1,
            maxLength: 100,
        },
        orderId: {
            placeholder: "the id of the existing order",
            tooltip: "the unique identifier of the order",
            label: "orderId",
            inputText: "order-id",
            feedback: "please input an order id",
            required: true,
            type: "number",
            minLength: 1,
            maxLength: 100,
        },
        chainId: {
            placeholder: "the id of the committee / counter asset",
            tooltip: "the unique identifier of the committee / counter asset",
            label: "chainId",
            inputText: "commit-Id",
            feedback: "please input a chainId id",
            required: true,
            type: "number",
            minLength: 1,
            maxLength: 100,
        },
        receiveAddress: {
            placeholder: "the address where the counter asset will be sent",
            tooltip: "the sender of the transaction",
            label: "receiveAddress",
            inputText: "rec-addr",
            feedback: "please choose an address to receive the counter asset to",
            required: true,
            type: "text",
            minLength: 40,
            maxLength: 40,
        },
        buyersReceiveAddress: {
            placeholder: "the canopy address where CNPY will be received",
            tooltip: "the sender of the transaction",
            label: "receiveAddress",
            inputText: "rec-addr",
            feedback: "please choose an address to receive the CNPY",
            required: true,
            type: "text",
            minLength: 40,
            maxLength: 40,
        },
        output: {
            placeholder: "output of the node",
            defaultValue: output,
            tooltip: "the non-custodial address where rewards and stake is directed to",
            label: "output",
            inputText: "output",
            feedback: "please choose an output address for the validator",
            required: true,
            type: "text",
            minLength: 40,
            maxLength: 40,
        },
        signer: {
            tooltip: "the signing address that authorizes the transaction",
            label: "signer",
            inputText: "signer",
            defaultValue: defaultNickSigner,
            type: "select",
            options: keyStore ? Object.keys(keyStore) : [],
        },
        paramSpace: {
            placeholder: "",
            defaultValue: "",
            tooltip: "the category 'space' of the parameter",
            label: "param_space",
            inputText: "space",
            feedback: "please choose a space for the parameter change",
            required: true,
            type: "select",
            minLength: 1,
            maxLength: 100,
        },
        paramKey: {
            placeholder: "",
            defaultValue: "",
            tooltip: "the identifier of the parameter",
            label: "param_key",
            inputText: "key",
            feedback: "please choose a key for the parameter change",
            required: true,
            type: "select",
            minLength: 1,
            maxLength: 100,
        },
        paramValue: {
            placeholder: "",
            defaultValue: "",
            tooltip: "the newly proposed value of the parameter",
            label: "param_value",
            inputText: "value",
            feedback: "please choose a value for the parameter change",
            required: true,
            type: "text",
            minLength: 1,
            maxLength: 100,
        },
        startBlock: {
            placeholder: "1",
            defaultValue: "",
            tooltip: "the block when voting starts",
            label: "start_block",
            inputText: "start blk",
            feedback: "please choose a height for start block",
            required: true,
            type: "number",
            minLength: 0,
            maxLength: 40,
        },
        endBlock: {
            placeholder: "100",
            defaultValue: "",
            tooltip: "the block when voting is counted",
            label: "end_block",
            inputText: "end blk",
            feedback: "please choose a height for end block",
            required: true,
            type: "number",
            minLength: 0,
            maxLength: 40,
        },
        memo: {
            placeholder: "opt: note attached with the transaction",
            defaultValue: "",
            tooltip: "an optional note attached to the transaction - blank is recommended",
            label: "memo",
            inputText: "memo",
            required: false,
            minLength: 0,
            maxLength: 200,
        },
        fee: {
            placeholder: "transaction fee in CNPY",
            defaultValue: "",
            tooltip: " a small amount of CNPY deducted from the account to process any transaction blank = default fee",
            label: "fee",
            inputText: "txn-fee",
            feedback: "please choose a valid number",
            required: false,
            type: "currency",
            minLength: 0,
            maxLength: 40,
        },
        password: {
            placeholder: "key password",
            defaultValue: "",
            tooltip: "the password for the private key sending the transaction",
            label: "password",
            inputText: "password",
            feedback: "please choose a valid password",
            required: true,
            type: "password",
            minLength: 0,
            maxLength: 40,
        },
        nickname: {
            placeholder: "key nickname",
            defaultValue: "",
            tooltip: "nickname assigned to key for easier identification",
            label: "nickname",
            inputText: "nickname",
            feedback: "nickname too long",
            required: false,
            type: "nickname",
            minLength: 0,
            maxLength: 40,
        },
    };
    switch (type) {
        case "send":
            return [a.account, a.rec, a.amount, a.memo, a.fee, a.password];
        case "stake":
            return [
                a.account,
                a.delegate,
                a.committees,
                a.netAddr,
                a.amount,
                a.earlyWithdrawal,
                a.output,
                a.signer,
                a.memo,
                a.fee,
                a.password,
            ];
        case "create_order":
            return [a.account, a.chainId, a.amount, a.receiveAmount, a.receiveAddress, a.memo, a.fee, a.password];
        case "lock_order":
            return [a.account, a.buyersReceiveAddress, a.orderId, a.fee, a.password];
        case "edit_order":
            return [
                a.account,
                a.chainId,
                a.orderId,
                a.amount,
                a.receiveAmount,
                a.receiveAddress,
                a.memo,
                a.fee,
                a.password,
            ];
        case "delete_order":
            return [a.account, a.chainId, a.orderId, a.memo, a.fee, a.password];
        case "edit-stake":
            return [
                a.account,
                a.committees,
                a.netAddr,
                a.amount,
                a.earlyWithdrawal,
                a.output,
                a.signer,
                a.memo,
                a.fee,
                a.password,
            ];
        case "change-param":
            return [a.account, a.paramSpace, a.paramKey, a.paramValue, a.startBlock, a.endBlock, a.memo, a.fee, a.password];
        case "dao-transfer":
            return [a.account, a.amount, a.startBlock, a.endBlock, a.memo, a.fee, a.password];
        case "pause":
        case "unpause":
        case "unstake":
            return [a.account, a.signer, a.memo, a.fee, a.password];
        case "pass-and-addr":
            return [a.account, a.password];
        case "pass-and-pk":
            return [a.privateKey, a.password];
        case "pass-only":
            return [a.password];
        case "pass-nickname-and-addr":
            return [a.account, a.password, a.nickname];
        case "pass-nickname-and-pk":
            return [a.privateKey, a.password, a.nickname];
        case "pass-and-nickname":
            return [a.password, a.nickname];
        default:
            return [a.account, a.memo, a.fee, a.password];
    }
}

// placeholders is a dummy object to assist in the user experience and provide consistency
export const placeholders = {
    poll: {
        "PLACEHOLDER EXAMPLE": {
            proposalHash: "PLACEHOLDER EXAMPLE",
            proposalURL: "https://discord.com/channels/1310733928436600912/1323330593701761204",
            accounts: {
                approvedPercent: 38,
                rejectPercent: 62,
                votedPercent: 35,
            },
            validators: {
                approvedPercent: 76,
                rejectPercent: 24,
                votedPercent: 77,
            },
        },
    },
    pollJSON: {
        proposal: "canopy network is the best",
        endBlock: 100,
        URL: "https://discord.com/link-to-thread",
    },
    proposals: {
        "2cbb73b8abdacf233f4c9b081991f1692145624a95004f496a95d3cce4d492a4": {
            proposal: {
                parameterSpace: "cons|fee|val|gov",
                parameterKey: "protocolVersion",
                parameterValue: "example",
                startHeight: 1,
                endHeight: 1000000,
                signer: "4646464646464646464646464646464646464646464646464646464646464646",
            },
            approve: false,
        },
    },
    params: {
        parameterSpace: "consensus",
        parameterKey: "protocolVersion",
        parameterValue: "1/150",
        startHeight: 1,
        endHeight: 100,
        signer: "303739303732333263...",
    },
    rawTx: {
        type: "changeParameter",
        msg: {
            parameterSpace: "cons",
            parameterKey: "blockSize",
            parameterValue: 1000,
            startHeight: 1,
            endHeight: 100,
            signer: "1fe1e32edc41d688...",
        },
        signature: {
            publicKey: "a88b9c0c7b77e7f8ac...",
            signature: "8f6d016d04e350...",
        },
        memo: "",
        fee: 10000,
    },
};

// numberWithCommas() formats a number with commas as thousand separators
export function numberWithCommas(x) {
    return x.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

// formatNumber() formats a number with optional division and compact notation
export function formatNumber(nString, div = true, cutoff = 1000000000000000) {
    if (nString == null) {
        return "zero";
    }
    if (div) {
        nString /= 1000000;
    }
    if (Number(nString) < cutoff) {
        return formatLocaleNumber(nString, 0, 8);
    }
    return Intl.NumberFormat("en", {notation: "compact", maximumSignificantDigits: 8}).format(nString);
}

// copy() copies text to clipboard and triggers a toast notification
export function copy(state, setState, detail, toastText = "Copied!") {
    if (navigator.clipboard && typeof navigator.clipboard.writeText === "function") {
        // if available use Clipboard API
        navigator.clipboard.writeText(detail)
            .then(() => setState({...state, toast: toastText}))
            .catch(() => fallbackCopy(detail, state, setState, toastText));
    } else {
        fallbackCopy(state, setState, detail, toastText)
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
        setState({...state, toast: toastText});
    } catch (err) {
        console.error("Fallback copy failed", err);
        setState({...state, toast: "Clipboard access denied"});
    }
    document.body.removeChild(textArea);
}

// renderToast() displays a toast notification with a customizable message
export function renderToast(state, setState) {
    return (
        <ToastContainer id="toast" position={"bottom-end"}>
            <Toast
                bg={"dark"}
                onClose={() => setState({...state, toast: ""})}
                show={state.toast != ""}
                delay={2000}
                autohide
            >
                <Toast.Body>{state.toast}</Toast.Body>
            </Toast>
        </ToastContainer>
    );
}

// onFormSubmit() handles form submission and passes form data to a callback
export function onFormSubmit(state, e, ks, callback) {
    e.preventDefault();
    let r = {};
    for (let i = 0; ; i++) {
        if (!e.target[i] || !e.target[i].ariaLabel) {
            break;
        }
        r[e.target[i].ariaLabel] = e.target[i].value;
    }
    if (r.sender) {
        r.sender = ks[r.sender].keyAddress;
    }
    if (r.signer) {
        r.signer = ks[r.signer].keyAddress;
    }
    callback(r);
}

// withTooltip() wraps an element with a tooltip component
export function withTooltip(obj, text, key, dir = "right") {
    return (
        <OverlayTrigger
            key={key}
            placement={dir}
            delay={{show: 250, hide: 400}}
            overlay={<Tooltip id="button-tooltip">{text}</Tooltip>}
        >
            {obj}
        </OverlayTrigger>
    );
}

// getRatio() calculates the simplest ratio between two numbers
export function getRatio(a, b) {
    const [bg, sm] = a > b ? [a, b] : [b, a];
    for (let i = 1; i < 1000000; i++) {
        const d = sm / i;
        const res = bg / d;
        if (Math.abs(res - Math.round(res)) < 0.1) {
            return a > b ? `${Math.round(res)}:${i}` : `${i}:${Math.round(res)}`;
        }
    }
}

// objEmpty() checks if an object is null, undefined, or empty
export function objEmpty(o) {
    return !o || Object.keys(o).length === 0;
}

// disallowedCharacters is a string of characters that are not allowed in form inputs.
export const disallowedCharacters = ["\t", '"'];

// sanitizeTextInput removes disallowed characters from the given event target value.
// It is meant to be used as an onChange event handler
export const sanitizeTextInput = (value) => {
    disallowedCharacters.forEach((char) => {
        value = value.split(char).join("");
    });
    return value;
};

// sanitizeNumberInput removes all non-digit characters from the given value.
// it also converts the value to a CNPY representation if toCnpy is true.
// It is meant to be used as an onChange event handler.
export const sanitizeNumberInput = (value, toCnpy = true) => {
    let input = value.replace(/[^\d]/g, ""); // Remove all non-digit characters
    // Allow a single zero but remove leading zeros once other numbers are added
    if (input.length > 1) {
        input = input.replace(/^0+/, "");
    }
    if (input === "") {
        return "";
    }
    if (toCnpy) {
        return formatLocaleNumber(toCNPY(input), 6, 6);
    }
    return formatLocaleNumber(Number(input));
};

// cnpyConversionRate sets the conversion rate between CNPY and uCNPY
export const cnpyConversionRate = 1_000_000;

// toCNPY converts a uCNPY amount to CNPY
export function toCNPY(uCNPY) {
    return uCNPY / cnpyConversionRate;
}

// toUCNPY converts a CNPY amount to uCNPY
export function toUCNPY(cnpy) {
    return cnpy * cnpyConversionRate;
}

// numberFromCommas removes commas from a string and returns a number
export const numberFromCommas = (str) => {
    return Number(parseFloat(str?.replace(/,/g, ""), 10));
};

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

// isValidJSON() checks if a given string is a valid JSON
export function isValidJSON(text) {
    try {
        JSON.parse(text);
        return true;
    } catch (e) {
        return false;
    }
}

// downloadJSON() downloads a JSON payload as a JSON file
export function downloadJSON(payload, filename) {
    const blob = new Blob([JSON.stringify(payload, null, 2)], {type: "application/json"});
    const blobUrl = URL.createObjectURL(blob);

    const anchor = document.createElement("a");
    anchor.href = blobUrl;
    anchor.target = "_blank";
    anchor.download = `${filename}.json`;

    // Auto click on a element, trigger the file download
    anchor.click();
    // Free up the memory by revoking the object URL
    URL.revokeObjectURL(blobUrl);
}

// retryWithDelay() retries a function with a delay between each attempt
export async function retryWithDelay(fn, onFailure, retries = 8, delayMs = 1000, throwOnFailure = false) {
    for (let attempt = 1; attempt <= retries; attempt++) {
        try {
            return await fn();
        } catch (error) {
            if (attempt < retries) {
                await new Promise((resolve) => setTimeout(resolve, delayMs));
            } else {
                onFailure();
                if (throwOnFailure) {
                    throw new Error(`All ${retries} attempts failed`);
                } else {
                    return;
                }
            }
        }
    }
}
