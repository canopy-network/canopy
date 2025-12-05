import {
  KeystoreGet,
  KeystoreImport,
  KeystoreNew,
  TxCloseOrder,
  TxCreateOrder,
  TxDeleteOrder,
  TxDexLimitOrder,
  TxDexLiquidityDeposit,
  TxDexLiquidityWithdrawal,
  TxEditOrder,
  TxEditStake,
  TxLockOrder,
  TxPause,
  TxSend,
  TxStake,
  TxUnpause,
  TxUnstake
} from "@/components/api";
import FormInputs from "@/components/form_inputs";
import {
  copy,
  downloadJSON,
  formatNumber,
  getActionFee,
  getFormInputs,
  numberFromCommas,
  onFormSubmit,
  renderToast,
  retryWithDelay,
  toCNPY,
  toUCNPY,
  withTooltip
} from "@/components/util";
import { KeystoreContext } from "@/pages";
import {
  CloseIcon,
  CopyIcon,
  DeleteOrderIcon,
  EditOrderIcon,
  EditStakeIcon,
  LockIcon,
  PauseIcon,
  SendIcon,
  StakeIcon,
  SwapIcon,
  UnpauseIcon,
  UnstakeIcon, DexLimitIcon, TransactionIcon, DexLiquidityWithdrawIcon, DexLiquidityDepositIcon
} from "@/components/svg_icons";
import { useContext, useEffect, useRef, useState } from "react";
import { Button, Card, Col, Form, Modal, Row, Spinner, Table } from "react-bootstrap";
import Alert from "react-bootstrap/Alert";
import Truncate from "react-truncate-inside";
import CanaJSON from "@/components/canaJSON";

function Keystore() {
  const keystore = useContext(KeystoreContext);
  return keystore;
}

// transactionButtons defines the icons for the transactions
const transactionButtons = [
  { title: "SEND", name: "send", src: SendIcon },
  { title: "STAKE", name: "stake", src: StakeIcon },
  { title: "EDIT", name: "edit-stake", src: EditStakeIcon },
  { title: "UNSTAKE", name: "unstake", src: UnstakeIcon },
  { title: "PAUSE", name: "pause", src: PauseIcon },
  { title: "PLAY", name: "unpause", src: UnpauseIcon },
  { title: "SWAP", name: "create_order", src: SwapIcon },
  { title: "LOCK", name: "lock_order", src: LockIcon },
  { title: "CLOSE", name: "close_order", src: CloseIcon },
  { title: "REPRICE", name: "edit_order", src: EditOrderIcon },
  { title: "VOID", name: "delete_order", src: DeleteOrderIcon },
  { title: "ORDER", name: "dex_limit_order", src: DexLimitIcon },
  { title: "DEPOSIT", name: "dex_liquidity_deposit", src: DexLiquidityDepositIcon },
  { title: "WITHDRAW", name: "dex_liquidity_withdrawal", src: DexLiquidityWithdrawIcon },
];

// Accounts() returns the main component of this file
export default function Accounts({ keygroup, account, validator, setActiveKey, params }) {
  const ks = Keystore();
  const ksRef = useRef(ks);

  const [state, setState] = useState({
      showModal: false,
      txType: "send",
      txResult: {},
      showSubmit: true,
      showPKModal: false,
      showPKImportModal: false,
      showNewModal: false,
      pk: {},
      toast: "",
      showSpinner: false,
      showAlert: false,
      alertMsg: "",
      primaryColor: "",
      greyColor: ""
    }),
    acc = account.account;

  const stateRef = useRef(state);
  const [buttonVariant, setButtonVariant] = useState("outline-dark");
  const [JsonViewVariant, setJsonViewVariant] = useState("json-dark");

  // Using a standalone useEffect here to isolate the color states
  useEffect(() => {
    // Check data-bs-theme on mount
    const currentTheme = document.documentElement.getAttribute("data-bs-theme");

    if (currentTheme === "dark") {
      setButtonVariant("outline-light");
      setJsonViewVariant("json-dark");
    } else {
      setButtonVariant("outline-dark");
      setJsonViewVariant("json-light");
    }
  }, [document.documentElement.getAttribute("data-bs-theme")]);

  useEffect(() => {
    // Ensure document is available
    const rootStyles = getComputedStyle(document.documentElement);
    const primaryColor = rootStyles.getPropertyValue("--primary-color").trim();
    const greyColor = rootStyles.getPropertyValue("--grey-color").trim();

    ksRef.current = ks;
    stateRef.current = state;

    // Update state with colors
    setState((prevState) => ({
      ...prevState,
      primaryColor,
      greyColor
    }));
  }, []);

  // resetState() resets the state back to its initial
  function resetState() {
    setState(prevState => ({
      ...prevState,
      pk: {},
      txResult: {},
      showSubmit: true,
      showModal: false,
      showPKModal: false,
      showNewModal: false,
      showPKImportModal: false
    }));
  }

  // onFormFieldChange() handles the form input change callback
  function onFormFieldChange(key, value, newValue) {
    if (key === "sender") {
      setActiveKey(Object.keys(ks).findIndex((v) => v === value));
    }
  }

  // showModal() makes the modal visible
  function showModal(t) {
    setState(prevState => ({ ...prevState, showModal: true, txType: t }));
  }

  // getAccountType() returns the type of validator account (custodial / non-custodial)
  function getAccountType() {
    return Object.keys(validator).length === 0 || validator.address === validator.output
      ? "CUSTODIAL"
      : "NON-CUSTODIAL";
  }

  // getValidatorAmount() returns the formatted staked amount of the validator
  function getValidatorAmount() {
    return validator.stakedAmount == null ? "0.00" : formatNumber(validator.stakedAmount);
  }

  // getStakedStatus() returns the staking status of the validator
  function getStakedStatus() {
    switch (true) {
      case !validator.address:
        return "UNSTAKED";
      case validator.unstakingHeight !== 0:
        return "UNSTAKING";
      case validator.maxPausedHeight !== 0:
        return "PAUSED";
      case validator.delegate:
        return "DELEGATING";
      default:
        return "STAKED";
    }
  }

  // setActivePrivateKey() sets the active key to the newly added privte key if it is successfully imported
  function setActivePrivateKey(nickname, closeModal) {
    const resetState = () =>
      setState({ ...stateRef.current, showSpinner: false, ...(closeModal && { [closeModal]: false }) });

    retryWithDelay(
      () => {
        let idx = Object.keys(ksRef.current).findIndex((k) => k === nickname);
        if (idx >= 0) {
          setActiveKey(idx);
          resetState();
        } else {
          throw new Error("failed to find key");
        }
      },
      resetState,
      10,
      1000,
      false
    );
  }

  // onPKFormSubmit() handles the submission of the private key form and updates the state with the retrieved key
  function onPKFormSubmit(e) {
    onFormSubmit(state, e, ks, (r) =>
      KeystoreGet(r.sender, r.password, r.nickname).then((r) => {
        setState(prevState => ({ ...prevState, showSubmit: Object.keys(prevState.txResult).length === 0, pk: r }));
      })
    );
  }

  // onNewPKFormSubmit() handles the submission of the new private key form and updates the state with the generated key
  function onNewPKFormSubmit(e) {
    onFormSubmit(state, e, ks, (r) =>
      KeystoreNew(r.password, r.nickname).then((r) => {
        setState(prevState => ({ ...prevState, showSubmit: Object.keys(prevState.txResult).length === 0, pk: r }));
        setActivePrivateKey(r.nickname);
      })
    );
  }

  // onImportOrGenerateSubmit() handles the submission of either the import or generate form and updates the state accordingly
  function onImportOrGenerateSubmit(e) {
    onFormSubmit(state, e, ks, (r) => {
      if (r.private_key) {
        void KeystoreImport(r.private_key, r.password, r.nickname).then((_) => {
          setState(prevState => ({ ...prevState, showSpinner: true }));
          setActivePrivateKey(r.nickname, "showPKImportModal");
        });
      } else {
        void KeystoreNew(r.password, r.nickname).then((_) => {
          setState(prevState => ({ ...prevState, showSpinner: true }));
          setActivePrivateKey(r.nickname, "showPKImportModal");
        });
      }
    });
  }

  // onTxFormSubmit() handles transaction form submissions based on transaction type
  function onTxFormSubmit(e) {
    onFormSubmit(state, e, ks, (r) => {
      const submit = Object.keys(state.txResult).length !== 0;
      // Mapping transaction types to their respective functions

      const amount = toUCNPY(numberFromCommas(r.amount));
      const fee = r.fee ? toUCNPY(numberFromCommas(r.fee)) : 0;
      const receiveAmount = toUCNPY(numberFromCommas(r.receiveAmount));

      const txMap = {
        send: () => TxSend(r.sender, r.recipient, amount, r.memo, fee, r.password, submit),
        stake: () =>
          TxStake(
            r.sender,
            r.pubKey,
            r.committees,
            r.net_address,
            amount,
            r.delegate,
            r.earlyWithdrawal,
            r.output,
            r.signer,
            r.memo,
            fee,
            r.password,
            submit
          ),
        "edit-stake": () =>
          TxEditStake(
            r.sender,
            r.committees,
            r.net_address,
            amount,
            r.earlyWithdrawal,
            r.output,
            r.signer,
            r.memo,
            fee,
            r.password,
            submit
          ),
        unstake: () => TxUnstake(r.sender, r.signer, r.memo, fee, r.password, submit),
        pause: () => TxPause(r.sender, r.signer, r.memo, fee, r.password, submit),
        unpause: () => TxUnpause(r.sender, r.signer, r.memo, fee, r.password, submit),
        create_order: () =>
          TxCreateOrder(r.sender, r.chainId, r.data, amount, receiveAmount, r.receiveAddress, r.memo, fee, r.password, submit),
        close_order: () => TxCloseOrder(r.sender, r.orderId, fee, r.password, submit),
        lock_order: () => TxLockOrder(r.sender, r.receiveAddress, r.orderId, fee, r.password, submit),
        edit_order: () =>
          TxEditOrder(
            r.sender,
            r.chainId,
            r.orderId,
            r.data,
            amount,
            receiveAmount,
            r.receiveAddress,
            r.memo,
            fee,
            r.password,
            submit
          ),
        delete_order: () => TxDeleteOrder(r.sender, r.chainId, r.orderId, r.memo, fee, r.password, submit),
        dex_limit_order: () => TxDexLimitOrder(r.sender, r.chainId, amount, receiveAmount, r.memo, fee, r.password, submit),
        dex_liquidity_deposit: () => TxDexLiquidityDeposit(r.sender, r.chainId, amount, r.memo, fee, r.password, submit),
        dex_liquidity_withdrawal: () => TxDexLiquidityWithdrawal(r.sender, r.chainId, r.percent, r.memo, fee, r.password, submit)
      };

      const txFunction = txMap[state.txType];
      if (txFunction) {
        setState(prevState => ({ ...prevState, showAlert: false }));
        txFunction()
          .then((result) => {
            setState(prevState => ({ ...prevState, showSubmit: !submit, txResult: result, showAlert: false }));
          })
          .catch((e) => {
            setState(prevState => ({
              ...prevState,
              showAlert: true,
              alertMsg: "Transaction failed. Please verify the fields and try again."
            }));
          });
      }
    });
  }

  // if no private key is preset
  if (!keygroup || Object.keys(keygroup).length === 0 || !account.account) {
    return (
      <RenderModal
        show={true}
        title={"UPLOAD PRIVATE OR CREATE KEY"}
        txType={"pass-nickname-and-pk"}
        onFormSub={onImportOrGenerateSubmit}
        keyGroup={null}
        account={null}
        validator={null}
        onHide={null}
        btnType={"import-or-generate"}
        setState={setState}
        state={state}
        closeOnClick={resetState}
        keystore={ks}
        JsonViewVariant={JsonViewVariant}
        onFormFieldChange={onFormFieldChange}
      />
    );
  }
  // return the main component
  return (
    <div className="content-container">
      <span id="balance">{formatNumber(acc.amount)}</span>
      <span style={{ fontFamily: "var(--font-heading)", fontWeight: "500", color: state.primaryColor }}>{" CNPY"}</span>
      <br />
      <hr />
      <br />
      <RenderModal
        show={state.showModal}
        title={state.txType}
        txType={state.txType}
        onFormSub={onTxFormSubmit}
        keyGroup={keygroup}
        account={acc}
        validator={validator}
        onHide={resetState}
        setState={setState}
        state={state}
        closeOnClick={resetState}
        keystore={ks}
        showAlert={state.showAlert}
        alertMsg={state.alertMsg}
        JsonViewVariant={JsonViewVariant}
        onFormFieldChange={onFormFieldChange}
        params={params}
      />
      {transactionButtons.map((v, i) => (
        <RenderActionButton key={i} v={v} i={i} showModal={showModal} />
      ))}
      <Row className="account-summary-container">
        {[
          { title: "Account Type", info: getAccountType() },
          { title: "Stake Amount", info: getValidatorAmount(), after: " cnpy" },
          { title: "Staked Status", info: getStakedStatus() }
        ].map((v, i) => (
          <RenderAccountInfo key={i} v={v} i={i} color={state.primaryColor} />
        ))}
      </Row>
      <br />
      <br />
      {[
        { title: "Nickname", info: keygroup.keyNickname },
        { title: "Address", info: keygroup.keyAddress },
        { title: "Public Key", info: keygroup.publicKey }
      ].map((v, i) => (
        <KeyDetail key={i} title={v.title} info={v.info} state={state} setState={setState} />
      ))}
      <br />
      <RenderTransactions account={account} state={state} setState={setState} />
      {renderToast(state, setState)}
      <RenderModal
        show={state.showPKModal}
        title={"Private Key"}
        txType={"pass-and-addr"}
        onFormSub={onPKFormSubmit}
        keyGroup={keygroup}
        account={acc}
        validator={null}
        onHide={resetState}
        setState={setState}
        state={state}
        closeOnClick={resetState}
        btnType={"reveal-pk"}
        keystore={ks}
        onFormFieldChange={onFormFieldChange}
      />
      <RenderModal
        show={state.showPKImportModal}
        title={"Private Key"}
        txType={"pass-nickname-and-pk"}
        onFormSub={onImportOrGenerateSubmit}
        keyGroup={keygroup}
        account={acc}
        validator={null}
        onHide={resetState}
        setState={setState}
        state={state}
        closeOnClick={resetState}
        btnType={"import-pk"}
        keystore={ks}
        onFormFieldChange={onFormFieldChange}
      />
      <RenderModal
        show={state.showNewModal}
        title={"Private Key"}
        txType={"pass-and-nickname"}
        onFormSub={onNewPKFormSubmit}
        keyGroup={null}
        account={null}
        validator={null}
        onHide={resetState}
        setState={setState}
        state={state}
        closeOnClick={resetState}
        btnType={"new-pk"}
        keystore={ks}
        onFormFieldChange={onFormFieldChange}
      />
      <Button id="pk-button" variant="outline-secondary" onClick={() => setState(prevState => ({ ...prevState, showNewModal: true }))}>
        New Private Key
      </Button>
      <Button
        id="import-pk-button"
        variant="outline-secondary"
        onClick={() => setState(prevState => ({ ...prevState, showPKImportModal: true }))}
      >
        Import Private Key
      </Button>
      <Button id="reveal-pk-button" variant="outline-danger" onClick={() => setState(prevState => ({ ...prevState, showPKModal: true }))}>
        Reveal Private Key
      </Button>
      <Button
        id="import-pk-button"
        variant="outline-secondary"
        onClick={() => {
          downloadJSON(ks, "keystore");
        }}
      >
        Download Keys
      </Button>
    </div>
  );
}

// renderKeyDetail() creates a clickable summary info box with a copy functionality
function KeyDetail({ i, title, info, state, setState }) {
  return (
    <div key={i} className="account-summary-info" onClick={() => copy(state, setState, info)}>
      <span className="account-summary-info-title">{title}</span>
      <div className="account-summary-info-content-container">
        <div className="account-summary-info-content">
          <Truncate text={info} />
        </div>
        <CopyIcon />
      </div>
    </div>
  );
}

// AccSumTabCol() returns an account summary table column
function AccSumTabCol({ detail, i, state, setState }) {
  return withTooltip(
    <td onClick={() => copy(state, setState, detail)}>
      <CopyIcon />
      <div className="account-summary-info-table-column">
        <Truncate text={detail} />
      </div>
    </td>,
    detail,
    i,
    "top"
  );
}

// SubmitBtn() renders a transaction submit button with customizable text, variant, and id
function SubmitBtn({ text, variant = "outline-secondary", id = "pk-button" }) {
  return (
    <Button id={id} variant={variant} type="submit">
      {text}
    </Button>
  );
}

// CloseBtn() renders a modal close button with a default onClick function
function CloseBtn({ onClick }) {
  return (
    <Button variant="secondary" onClick={onClick}>
      Close
    </Button>
  );
}

// RenderButtons() returns buttons based on the specified type
function RenderButtons({ type, state, closeOnClick }) {
  switch (type) {
    case "import-or-generate":
      return <SubmitBtn text="Import or Generate Key" />;
    case "import-pk":
      return (
        <>
          <SubmitBtn text="Import Key" variant="outline-danger" />
          <CloseBtn onClick={closeOnClick} />
        </>
      );
    case "new-pk":
      return (
        <>
          <SubmitBtn text="Generate New Key" />
          <CloseBtn onClick={closeOnClick} />
        </>
      );
    case "reveal-pk":
      return (
        <>
          <SubmitBtn text="Get Private Key" variant="outline-danger" />
          <CloseBtn onClick={closeOnClick} />
        </>
      );
    default:
      if (Object.keys(state.txResult).length === 0) {
        return (
          <>
            <SubmitBtn text={"Generate Transaction"} />
            <CloseBtn onClick={closeOnClick} />
          </>
        );
      } else {
        const s = state.showSubmit ? <SubmitBtn text="Submit Transaction" variant="outline-secondary" /> : <></>;
        return (
          <>
            {s}
            {<CloseBtn onClick={closeOnClick} />}
          </>
        );
      }
  }
}

// RenderModal() returns the transaction modal
function RenderModal({
                       show,
                       title,
                       txType,
                       onFormSub,
                       keyGroup,
                       account,
                       validator,
                       onHide,
                       btnType,
                       setState,
                       state,
                       closeOnClick,
                       keystore,
                       showAlert = false,
                       alertMsg,
                       JsonViewVariant,
                       onFormFieldChange,
                       params
                     }) {
  return (
    <Modal show={show} size="lg" onHide={onHide}>
      <Form onSubmit={onFormSub}>
        <Modal.Header>
          <Modal.Title className="modal-title">{title}</Modal.Title>
        </Modal.Header>
        <Modal.Body className="modal-body">
          <FormInputs
            keygroup={keyGroup}
            fields={getFormInputs(txType, keyGroup, account, validator, keystore).map((formInput) => {
              let input = Object.assign({}, formInput);
              if (input.label === "sender") {
                input.options.sort((a, b) => {
                  if (a === account.nickname) return -1;
                  if (b === account.nickname) return 1;
                  return 0;
                });
              }
              if (input.label === "fee") {
                input.defaultValue = getActionFee(txType, params.fee);
              }
              return input;
            })}
            account={(function() {
              // copy accounts and extract the fee if any
              let accountCopy = Object.assign({}, account);
              accountCopy.amount -= getActionFee(txType, params?.fee) ?? 0;
              return accountCopy;
            })()}
            show={show}
            validator={validator}
            onFieldChange={onFormFieldChange}
          />
          {showAlert && <Alert variant={"danger"}>{alertMsg}</Alert>}
          <CanaJSON state={state} setState={setState} JsonViewVariant={JsonViewVariant} />
          <Spinner style={{ display: state.showSpinner ? "block" : "none", margin: "0 auto" }} />
        </Modal.Body>

        <Modal.Footer>
          <RenderButtons type={btnType} state={state} closeOnClick={closeOnClick} />
        </Modal.Footer>
      </Form>
    </Modal>
  );
}

// RenderActionButton() creates a button with an SVG and title, triggering a modal on click
function RenderActionButton({ v, i, showModal }) {
  const IconComponent = v.src;

  if (!IconComponent) {
    return null; // Or a placeholder if the SVG isn't found
  }

  return (
    <div key={i} className="send-receive-button-container" onClick={() => showModal(v.name)}>
      <IconComponent className="icon-button" />
      <br />
      <span className="action-button-title">{v.title}</span>
    </div>
  );
}

// RenderAccountInfo() generates a card displaying account summary details
function RenderAccountInfo({ v, i }, color) {
  return (
    <Col key={i}>
      <Card className="account-summary-container-card">
        <Card.Header style={{ fontWeight: "100" }}>{v.title}</Card.Header>
        <Card.Body style={{ padding: "10px" }}>
          <Card.Title style={{ fontWeight: "500", fontSize: "14px" }}>
            {v.info}
            <span style={{ fontSize: "10px", color: color }}>{v.after}</span>
          </Card.Title>
        </Card.Body>
      </Card>
    </Col>
  );
}

// RenderTransactions() displays a table of recent transactions based on account data
function RenderTransactions({ account, state, setState }) {
  return account.combined.length === 0 ? null : (
    <div className="recent-transactions-table">
      <span class="table-label">RECENT TRANSACTIONS</span>
      <Table className="table-fixed" bordered hover style={{ marginTop: "10px" }}>
        <thead>
        <tr>
          {["Height", "Amount", "Recipient", "Type", "Hash", "Status"].map((k, i) => (
            <th key={i}>{k}</th>
          ))}
        </tr>
        </thead>
        <tbody>
        {account.combined.slice(0, 5).map((v, i) => (
          <tr key={i}>
            <td>{v.height || "N/A"}</td>
            <td>{toCNPY(v.transaction.msg.amount) || toCNPY(v.transaction.msg.amountForSale) || "N/A"}</td>
            <AccSumTabCol detail={v.recipient ?? v.sender ?? v.address} i={i} state={state} setState={setState} />
            <td>{v.messageType || v.transaction.type}</td>
            <AccSumTabCol detail={v.txHash} i={i + 1} state={state} setState={setState} />
            <td>{v.status ?? ""}</td>
          </tr>
        ))}
        </tbody>
      </Table>
    </div>
  );
}
