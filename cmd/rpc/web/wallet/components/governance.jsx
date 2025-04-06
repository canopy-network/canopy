import React, { useEffect, useState, useContext } from "react";
import Truncate from "react-truncate-inside";
import CanaJSON from "@/components/canaJSON";
import { Bar } from "react-chartjs-2";
import { Chart as ChartJS, BarElement, CategoryScale, LinearScale, Tooltip, Legend } from "chart.js";
import { Accordion, Button, Carousel, Container, Form, InputGroup, Modal, Spinner, Table } from "react-bootstrap";
import {
  AddVote,
  DelVote,
  Params,
  Poll,
  Proposals,
  RawTx,
  StartPoll,
  TxChangeParameter,
  TxDAOTransfer,
  VotePoll,
} from "@/components/api";
import {
  isValidJSON,
  copy,
  getFormInputs,
  objEmpty,
  onFormSubmit,
  placeholders,
  renderToast,
  toUCNPY,
  numberFromCommas,
  formatLocaleNumber,
} from "@/components/util";
import { KeystoreContext } from "@/pages";
import FormInputs from "@/components/form_inputs";
import { PollIcon, ProposalIcon } from "@/components/svg_icons";

function useKeystore() {
  const keystore = useContext(KeystoreContext);
  return keystore;
}

ChartJS.register(BarElement, CategoryScale, LinearScale, Tooltip, Legend);

export default function Governance({ keygroup, account: accountWithTxs, validator }) {
  const ks = useKeystore();
  const [state, setState] = useState({
    txResult: {},
    rawTx: {},
    showPropModal: false,
    apiResults: {},
    paramSpace: "",
    voteOnPollAccord: "1",
    voteOnProposalAccord: "1",
    propAccord: "1",
    txPropType: 0,
    toast: "",
    voteJSON: {},
    pwd: "",
  });
  const [primaryColor, setPrimaryColor] = useState("");
  const [secondaryColor, setSecondaryColor] = useState("");
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

  // onFormChange() handles the form input change callback
  function onFormChange(key, value, newValue) {
    if (key === "param_space") {
      setState({ ...state, paramSpace: newValue });
    }
  }

  // onPropSubmit() handles the proposal submit form callback
  function onPropSubmit(e) {
    onFormSubmit(state, e, ks, (r) => {
      if (state.txPropType === 0) {
        createParamChangeTx(
          r.sender,
          r.param_space,
          r.param_key,
          r.param_value,
          numberFromCommas(r.start_block),
          numberFromCommas(r.end_block),
          r.memo,
          toUCNPY(numberFromCommas(r.fee)),
          r.password,
        );
      } else {
        createDAOTransferTx(
          r.sender,
          toUCNPY(numberFromCommas(r.amount)),
          numberFromCommas(r.start_block),
          numberFromCommas(r.end_block),
          r.memo,
          r.fee,
          r.password,
        );
      }
    });
  }

  // queryAPI() executes the page API calls
  function queryAPI() {
    return Promise.all([Poll(), Proposals(), Params(0)]).then((r) => {
      setState((prevState) => ({
        ...prevState,
        apiResults: {
          poll: r[0],
          proposals: r[1],
          params: r[2],
        },
      }));
    });
  }

  useEffect(() => {
    queryAPI();

    if (typeof window !== "undefined") {
      const root = document.documentElement;
      const primaryColorValue = getComputedStyle(root).getPropertyValue("--primary-color").trim();
      const secondaryColorValue = getComputedStyle(root).getPropertyValue("--secondary-color").trim();

      setPrimaryColor(primaryColorValue);
      setSecondaryColor(secondaryColorValue);
    }
    const interval = setInterval(queryAPI, 4000);
    return () => clearInterval(interval);
  }, []);

  // set spinner if loading
  if (objEmpty(state.apiResults)) {
    return <Spinner id="spinner" />;
  }
  // set placeholders
  if (objEmpty(state.apiResults.poll)) state.apiResults.poll = placeholders.poll;
  else state.apiResults.poll["PLACEHOLDER EXAMPLE"] = placeholders.poll["PLACEHOLDER EXAMPLE"];
  if (objEmpty(state.apiResults.proposals)) state.apiResults.proposals = placeholders.proposals;

  // addVoteAPI() executes an 'Add Vote' API call and sets the state when complete
  function addVoteAPI(json, approve) {
    return AddVote(json, approve).then((_) => setState({ ...state, voteOnProposalAccord: "1", toast: "Voted!" }));
  }

  // submitProposalAPI() executes a 'Raw Tx' API call and sets the state when complete
  function submitProposalAPI() {
    return sendRawTx(
      state.rawTx,
      {
        ...state,
        propAccord: "1",
      },
      setState,
    );
  }

  // delVoteAPI() executes a 'Delete Vote' API call and sets the state when complete
  function delVoteAPI(json) {
    return DelVote(json).then((_) => setState({ ...state, voteOnProposalAccord: "1", toast: "Deleted!" }));
  }

  // startPollAPI() executes a 'Start Poll' API call and sets the state when complete
  function startPollAPI(address, json, password) {
    return StartPoll(address, json, password).then((_) =>
      setState({ ...state, voteOnPollAccord: "1", toast: "Started Poll!" }),
    );
  }

  // votePollAPI() executes a 'Vote Poll' API call and sets the state when complete
  function votePollAPI(address, json, approve, password) {
    return VotePoll(address, json, approve, password).then((_) =>
      setState({ ...state, voteOnPollAccord: "1", toast: "Voted!" }),
    );
  }

  // handlePropClose() closes the proposal modal from a button or modal x
  function handlePropClose() {
    setState({ ...state, paramSpace: "", txResult: {}, showPropModal: false });
  }

  // handlePropOpen() opens the proposal modal
  function handlePropOpen(type) {
    setState({ ...state, txPropType: type, showPropModal: true, paramSpace: "", txResult: {} });
  }

  // sendRawTx() executes the RawTx API call and sets the state when complete
  function sendRawTx(j, state, setState) {
    return RawTx(j).then((r) => {
      copy(state, setState, r, "tx hash copied to keyboard!");
    });
  }

  // createDAOTransferTx() executes a dao transfer transaction API call and sets the state when complete
  function createDAOTransferTx(address, amount, startBlock, endBlock, memo, fee, password) {
    TxDAOTransfer(address, amount, startBlock, endBlock, memo, fee, password, false).then((res) => {
      setState({ ...state, txResult: res });
    });
  }

  // createParamChangeTx() executes a param change transaction API call and sets the state when completed
  function createParamChangeTx(address, paramSpace, paramKey, paramValue, startBlock, endBlock, memo, fee, password) {
    TxChangeParameter(address, paramSpace, paramKey, paramValue, startBlock, endBlock, memo, fee, password, false).then(
      (res) => {
        setState({ ...state, txResult: res });
      },
    );
  }

  return (
    <div className="content-container">
      <Header title="poll" svg={PollIcon} />
      <div className="poll-container">
        <Carousel className="poll-carousel" interval={null} data-bs-theme="dark">
          {Array.from(Object.entries(state.apiResults.poll)).map((entry, idx) => {
            const [key, val] = entry;
            return (
              <Carousel.Item key={idx}>
                <h6 className="poll-prop-hash">{val.proposalHash}</h6>
                <a href={val.proposalURL} className="poll-prop-url">
                  {val.proposalURL}
                </a>
                <Container className="poll-carousel-container" fluid>
                  <Bar
                    data={{
                      labels: [
                        val.accounts.votedPercent + "% Accounts Reporting",
                        val.validators.votedPercent + "% Validators Reporting",
                      ], // Categories
                      datasets: [
                        {
                          label: "% Voted YES",
                          data: [val.accounts.approvedPercent, val.validators.approvedPercent],
                          backgroundColor: primaryColor,
                        },
                        {
                          label: "% Voted NO",
                          data: [val.accounts.rejectPercent, val.validators.rejectPercent],
                          backgroundColor: secondaryColor,
                        },
                      ],
                    }}
                    options={{
                      responsive: true,
                      plugins: { tooltip: { enabled: true } },
                      scales: { y: { beginAtZero: true, max: 100 } },
                    }}
                  />
                  <br />
                </Container>
              </Carousel.Item>
            );
          })}
        </Carousel>
        <br />
        <Accord
          state={state}
          setState={setState}
          title="START OR VOTE ON POLL"
          keyName="voteOnPollAccord"
          targetName="voteJSON"
          buttonVariant={buttonVariant}
          buttons={[
            {
              title: "START NEW",
              onClick: () => startPollAPI(accountWithTxs.account.address, state.voteJSON, state.pwd),
            },
            {
              title: "APPROVE",
              onClick: () => votePollAPI(accountWithTxs.account.address, state.voteJSON, true, state.pwd),
            },
            {
              title: "REJECT",
              onClick: () => votePollAPI(accountWithTxs.account.address, state.voteJSON, false, state.pwd),
            },
          ]}
          showPwd={true}
          placeholder={placeholders.pollJSON}
        />
      </div>
      <br />
      <br />
      <hr />
      <Header title="propose" svg={ProposalIcon} />
      <Table className="vote-table" bordered responsive hover>
        <thead>
          <tr>
            <th>VOTE</th>
            <th>PROPOSAL ID</th>
            <th>ENDS</th>
          </tr>
        </thead>
        <tbody>
          {Array.from(
            Object.entries(state.apiResults.proposals).map((entry, idx) => {
              const [key, value] = entry;
              return (
                <tr key={idx}>
                  <td>{value.approve ? "YES" : "NO"}</td>
                  <td>
                    <div className="vote-table-col">
                      <Truncate text={"#" + key} />
                    </div>
                  </td>
                  <td>{formatLocaleNumber(value.proposal.msg["endHeight"], 0, 0)}</td>
                </tr>
              );
            }),
          )}
        </tbody>
      </Table>
      <Accord
        state={state}
        setState={setState}
        title="VOTE ON PROPOSAL"
        keyName="voteOnProposalAccord"
        targetName="voteJSON"
        buttonVariant={buttonVariant}
        buttons={[
          { title: "APPROVE", onClick: () => addVoteAPI(state.voteJSON, true) },
          { title: "REJECT", onClick: () => addVoteAPI(state.voteJSON, false) },
          { title: "DELETE", onClick: () => delVoteAPI(state.voteJSON) },
        ]}
        showPwd={false}
      />
      <Accord
        state={state}
        setState={setState}
        title="SUBMIT PROPOSAL"
        keyName="propAccord"
        targetName="rawTx"
        buttonVariant={buttonVariant}
        buttons={[
          {
            title: "SUBMIT",
            onClick: () => {
              submitProposalAPI();
            },
          },
        ]}
        showPwd={false}
        placeholder={placeholders.rawTx}
      />
      <Button className="propose-button" onClick={() => handlePropOpen(0)} variant={buttonVariant}>
        New Protocol Change
      </Button>
      <Button className="propose-button" onClick={() => handlePropOpen(1)} variant={buttonVariant}>
        New Treasury Subsidy
      </Button>
      <br />
      <br />
      <Modal show={state.showPropModal} size="lg" onHide={handlePropClose} JsonViewVariant={JsonViewVariant}>
        <Form onSubmit={onPropSubmit}>
          <Modal.Header closeButton>
            <Modal.Title>{state.txPropType === 0 ? "Change Parameter" : "Treasury Subsidy"}</Modal.Title>
          </Modal.Header>
          <Modal.Body style={{ overflowWrap: "break-word" }}>
            <FormInputs
              fields={getFormInputs(
                state.txPropType === 0 ? "change-param" : "dao-transfer",
                keygroup,
                accountWithTxs.account,
                validator,
                ks,
              ).map((formInput) => {
                let input = Object.assign({}, formInput);
                switch (formInput.label) {
                  case "sender":
                    input.options.sort((a, b) => {
                      if (a === accountWithTxs.account.nickname) return -1;
                      if (b === accountWithTxs.account.nickname) return 1;
                      return 0;
                    });
                    break;
                  case "param_space":
                    input.options = Object.keys(state.apiResults.params);
                    break;
                  case "param_key":
                    // Add the first api result as the default param space
                    const paramSpace = state.paramSpace || Object.keys(state.apiResults.params)[0];
                    const params = state.apiResults.params[paramSpace];
                    input.options = params ? Object.keys(params) : [];
                    break;
                }
                return input;
              })}
              keygroup={keygroup}
              account={accountWithTxs.account}
              show={state.showPropModal}
              validator={validator}
              onFieldChange={onFormChange}
            />
            {!objEmpty(state.txResult) && (
              <CanaJSON state={state} setState={setState} JsonViewVariant={JsonViewVariant} />
            )}
          </Modal.Body>
          <Modal.Footer>
            <SubmitBtn txResult={state.txResult} onClick={handlePropClose} />
          </Modal.Footer>
        </Form>
      </Modal>
      {renderToast(state, setState)}
    </div>
  );
}

// Header() renders the section header in the governance tab
function Header({ title, svg: SVGComponent }) {
  if (!SVGComponent) {
    return null; // Or a placeholder if the SVG isn't found
  }

  return (
    <div class="gov-header">
      <SVGComponent />
      <span id="propose-title">{title}</span>
      <span id="propose-subtitle"> on CANOPY</span>
      <hr className="gov-header-hr" />
    </div>
  );
}

// Accord() renders an accordion object for governance polling and proposals
function Accord({
  state,
  setState,
  title,
  keyName,
  targetName,
  buttons,
  showPwd,
  placeholder = placeholders.params,
  isJSON = true,
  buttonVariant,
}) {
  const handleChange = (key, value) =>
    setState((prevState) => ({
      ...prevState,
      [key]: value,
    }));

  placeholder = JSON.stringify(placeholder, null, 2);
  const handleAccordionChange = (key, value) =>
    setState((prevState) => ({
      ...prevState,
      // Set the targetName placeholder to the value of the textarea
      ...(objEmpty(state[targetName]) && { [targetName]: placeholder }),
      [key]: value,
    }));

  return (
    <Accordion className="accord" activeKey={state[keyName]} onSelect={(i) => handleAccordionChange(keyName, i)}>
      <Accordion.Item className="accord-item" eventKey="0">
        <Accordion.Header>{title}</Accordion.Header>
        <Accordion.Body>
          <Form.Control
            className="accord-body-container"
            defaultValue={placeholder}
            as="textarea"
            onChange={(e) => handleChange(targetName, e.target.value)}
          />
          {showPwd && (
            <InputGroup className="accord-pass-container" size="lg">
              <InputGroup.Text>Password</InputGroup.Text>
              <Form.Control type="password" onChange={(e) => handleChange("pwd", e.target.value)} required />
            </InputGroup>
          )}
          {buttons.map((btn, idx) => (
            <Button
              key={idx}
              className="propose-button"
              onClick={() => {
                let text = state[targetName];
                if (isJSON && (text === "" || !isValidJSON(state[targetName]))) {
                  setState({ ...state, toast: "Invalid JSON!" });
                  return;
                }
                btn.onClick();
              }}
              variant={buttonVariant}
            >
              {btn.title}
            </Button>
          ))}
        </Accordion.Body>
      </Accordion.Item>
    </Accordion>
  );
}

// SubmitBtn() renders the 'submit' buttons for the governance modal footer
function SubmitBtn({ txResult, onClick }) {
  return (
    <>
      <Button
        style={{ display: objEmpty(txResult) ? "" : "none" }}
        id="import-pk-button"
        variant="outline-secondary"
        type="submit"
      >
        Generate New Proposal
      </Button>
      <Button variant="secondary" onClick={onClick}>
        Close
      </Button>
    </>
  );
}
