import React, {useEffect, useState} from "react"
import Truncate from "react-truncate-inside"
import JsonView from "@uiw/react-json-view"
import {Bar} from "react-chartjs-2"
import {Chart as ChartJS, BarElement, CategoryScale, LinearScale, Tooltip, Legend} from "chart.js"
import {Accordion, Button, Carousel, Col, Container, Form, InputGroup, Modal, Spinner, Table} from "react-bootstrap"
import {AddVote, DelVote, Params, Poll, Proposals, RawTx, StartPoll, TxChangeParameter, TxDAOTransfer, VotePoll} from "@/components/api"
import {copy, getFormInputs, objEmpty, onFormSubmit, placeholders, renderToast, withTooltip} from "@/components/util"

ChartJS.register(BarElement, CategoryScale, LinearScale, Tooltip, Legend)

export default function Governance({account: accountWithTxs}) {
    const [state, setState] = useState({
        txResult: {}, rawTx: {}, showPropModal: false, apiResults: {}, paramSpace: "", voteOnPollAccord: "1", voteOnProposalAccord: "1",
        propAccord: "1", txPropType: 0, toast: "", voteJSON: {}, pwd: "",
    })

    // queryAPI() executes the page API calls
    function queryAPI() {
        Promise.all([Poll(), Proposals(), Params(0)]).then((r => {
                setState({...state, apiResults: {poll: r[0], proposals: r[1], params: r[2]}})
            }
        ))
    }

    // execute every 4 seconds
    useEffect(() => () => clearInterval(setInterval(() => queryAPI(), 4000)), [])
    // set spinner if loading
    if (objEmpty(state.apiResults)) return (queryAPI(), <Spinner id="spinner"/>)
    // set placeholders
    if (objEmpty(state.apiResults.poll)) state.apiResults.poll = placeholders.poll
    else state.apiResults.poll["PLACEHOLDER EXAMPLE"] = placeholders.poll["PLACEHOLDER EXAMPLE"]
    if (objEmpty(state.apiResults.proposals)) state.apiResults.proposals = placeholders.proposals

    // addVoteAPI() executes an 'Add Vote' API call and sets the state when complete
    function addVoteAPI(json, approve) {
        return AddVote(json, approve).then(_ => setState({...state, voteOnProposalAccord: "1", toast: "Voted!"}))
    }

    // delVoteAPI() executes a 'Delete Vote' API call and sets the state when complete
    function delVoteAPI(json) {
        return DelVote(json).then(_ => setState({...state, voteOnProposalAccord: "1", toast: "Deleted!"}))
    }

    // startPollAPI() executes a 'Start Poll' API call and sets the state when complete
    function startPollAPI(address, json, password) {
        return StartPoll(address, json, password).then(_ => setState({...state, voteOnPollAccord: "1", toast: "Started Poll!"}))
    }

    // votePollAPI() executes a 'Vote Poll' API call and sets the state when complete
    function votePollAPI(address, json, approve, password) {
        return VotePoll(address, json, approve, password).then(_ => setState({...state, voteOnPollAccord: "1", toast: "Voted!"}))
    }

    // handlePropClose() closes the proposal modal from a button or modal x
    function handlePropClose() {
        setState({...state, paramSpace: "", txResult: {}, showPropModal: false})
    }

    // handlePropOpen() opens the proposal modal
    function handlePropOpen(type) {
        setState({...state, txPropType: type, showPropModal: true, paramSpace: "", txResult: {}})
    }

    // sendRawTx() executes the RawTx API call and sets the state when complete
    function sendRawTx(j) {
        return RawTx(j).then(r => {
            copy(state, setState, r, "tx hash copied to keyboard!")
        })
    }

    // createDAOTransferTx() executes a dao transfer transaction API call and sets the state when complete
    function createDAOTransferTx(address, amount, startBlock, endBlock, memo, fee, password) {
        TxDAOTransfer(address, amount, startBlock, endBlock, memo, fee, password, false).then(res => {
            setState({...state, txResult: res})
        })
    }

    // createParamChangeTx() executes a param change transaction API call and sets the state when completed
    function createParamChangeTx(address, paramSpace, paramKey, paramValue, startBlock, endBlock, memo, fee, password) {
        TxChangeParameter(address, paramSpace, paramKey, paramValue, startBlock, endBlock, memo, fee, password, false).then(res => {
            setState({...state, txResult: res})
        })
    }

    // onPropSubmit() handles the proposal submit form callback
    function onPropSubmit(e) {
        onFormSubmit(state, e, (r) => {
            if (state.txPropType === 0) {
                createParamChangeTx(r.sender, r.param_space, r.param_key, r.param_value, r.start_block, r.end_block, r.memo, r.fee, r.password)
            } else {
                createDAOTransferTx(r.sender, r.amount, r.start_block, r.end_block, r.memo, r.fee, r.password)
            }
        })
    }

    // renderJSONViewer() returns a raw json display
    function renderJSONViewer() {
        return objEmpty(state.txResult) ? <></> : <JsonView value={state.txResult} shortenTextAfterLength={100} displayDataTypes={false}/>
    }

    // renderButtons() renders the 'submit' buttons for the governance modal footer
    function renderButtons() {
        return <>
            <Button style={{display: objEmpty(state.txResult) ? "" : "none"}} id="import-pk-button" variant="outline-secondary" type="submit">
                Generate New Proposal
            </Button>
            <Button variant="secondary" onClick={handlePropClose}>
                Close
            </Button>
        </>
    }

    // renderHeader() renders the section header in the governance tab
    function renderHeader(title, img) {
        return <>
            <img className="governance-header-image" alt="vote" src={img}/>
            <span id="propose-title">{title}</span><span id="propose-subtitle">  on CANOPY</span>
            <br/><br/>
            <hr className="gov-header-hr"/>
            <br/><br/>
        </>
    }

    // renderAccord() renders an accordion object for governance polling and proposals
    function renderAccord(title, keyName, targetName, buttons, showPwd, placeholder = placeholders.params) {
        const handleChange = (key, value) => setState({...state, [key]: value, ...(key === targetName && {voteJSON: value})})
        return (
            <Accordion className="accord" activeKey={state[keyName]} onSelect={(i) => handleChange(keyName, i)}>
                <Accordion.Item className="accord-item" eventKey="0">
                    <Accordion.Header>{title}</Accordion.Header>
                    <Accordion.Body>
                        <Form.Control
                            className="accord-body-container"
                            defaultValue={JSON.stringify(placeholder, null, 2)}
                            as="textarea"
                            onChange={(e) => handleChange(targetName, e.target.value)}
                        />
                        {showPwd && (
                            <InputGroup className="accord-pass-container" size="lg">
                                <InputGroup.Text>Password</InputGroup.Text>
                                <Form.Control type="password" onChange={(e) => handleChange("pwd", e.target.value)} required/>
                            </InputGroup>
                        )}
                        {buttons.map((btn, idx) => (
                            <Button key={idx} className="propose-button" onClick={btn.onClick} variant="outline-dark">
                                {btn.title}
                            </Button>
                        ))}
                    </Accordion.Body>
                </Accordion.Item>
            </Accordion>
        )
    }

    return <>
        <div className="content-container">
            {renderHeader("poll", "./poll.png")}
            <Carousel className="poll-carousel" interval={null} data-bs-theme="dark">
                {Array.from(Object.entries(state.apiResults.poll)).map((entry, idx) => {
                    const [key, val] = entry
                    return <Carousel.Item key={idx}>
                        <h6 className="poll-prop-hash">{val.proposalHash}</h6>
                        <a href={val.proposalURL} className="poll-prop-url">{val.proposalURL}</a>
                        <Container className="poll-carousel-container" fluid>
                            <Bar data={{
                                labels: [val.accounts.votedPercent + '% Accounts Reporting', val.validators.votedPercent + '% Validators Reporting'], // Categories
                                datasets: [
                                    {label: '% Voted YES', data: [val.accounts.approvedPercent, val.validators.approvedPercent], backgroundColor: '#5447ac'},
                                    {label: '% Voted NO', data: [val.accounts.rejectPercent, val.validators.rejectPercent], backgroundColor: '#000'},
                                ],
                            }} options={{
                                responsive: true, plugins: {tooltip: {enabled: true}}, scales: {y: {beginAtZero: true, max: 100}},
                            }}/>
                            <br/>
                        </Container>
                    </Carousel.Item>
                })}
            </Carousel>
            <br/>
            {renderAccord("START OR VOTE ON POLL", "voteOnPollAccord", "voteJSON", [
                {title: "START NEW", onClick: () => startPollAPI(accountWithTxs.account.address, state.voteJSON, state.pwd)},
                {title: "APPROVE", onClick: () => votePollAPI(accountWithTxs.account.address, state.voteJSON, true, state.pwd)},
                {title: "REJECT", onClick: () => votePollAPI(accountWithTxs.account.address, state.voteJSON, false, state.pwd)},
            ], true, placeholders.pollJSON)}
            {renderHeader("propose", "./proposal.png")}
            <Table className="vote-table" bordered responsive hover>
                <thead>
                <tr>
                    <th>VOTE</th>
                    <th>PROPOSAL ID</th>
                    <th>ENDS</th>
                </tr>
                </thead>
                <tbody>
                {Array.from(Object.entries(state.apiResults.proposals).map((entry, idx) => {
                    const [key, value] = entry
                    return <tr key={idx}>
                        <td>{value.approve ? "YES" : "NO"}</td>
                        <td>
                            <div className="vote-table-col">
                                <Truncate text={"#" + key}/>
                            </div>
                        </td>
                        <td>{value.proposal["end_height"]}</td>
                    </tr>
                }))}
                </tbody>
            </Table>
            {renderAccord("VOTE ON PROPOSAL", "voteOnProposalAccord", "voteJSON", [
                {title: "APPROVE", onClick: () => addVoteAPI(state.voteJSON, true)},
                {title: "REJECT", onClick: () => addVoteAPI(state.voteJSON, false)},
                {title: "DELETE", onClick: () => delVoteAPI(state.voteJSON)},
            ], false)}
            {renderToast(state, setState)}
            <br/>
            {renderAccord("SUBMIT PROPOSAL", "propAccord", "rawTx", [
                {title: "SUBMIT", onClick: () => sendRawTx(state.rawTx)}
            ], false, placeholders.rawTx)}
            <Button className="propose-button" onClick={() => handlePropOpen(0)} variant="outline-dark">New Protocol Change</Button>
            <Button className="propose-button" onClick={() => handlePropOpen(1)} variant="outline-dark">New Treasury Subsidy</Button>
            <br/><br/>
            <Modal show={state.showPropModal} size="lg" onHide={handlePropClose}>
                <Form onSubmit={onPropSubmit}>
                    <Modal.Header closeButton>
                        <Modal.Title>{state.txPropType === 0 ? "Change Parameter" : "Treasury Subsidy"}</Modal.Title>
                    </Modal.Header>
                    <Modal.Body style={{overflowWrap: "break-word"}}>
                        {getFormInputs(state.txPropType === 0 ? "change-param" : "dao-transfer", accountWithTxs.account).map((v, i) => {
                            let show = objEmpty(state.txResult) ? "" : "none", onChange = (e) => setState({...state, paramSpace: e.target.value})
                            if (v.type === "select") {
                                switch (v.label) {
                                    case "param_key":
                                        let paramKeys = [], ps = state.apiResults.params[state.paramSpace]
                                        if (ps) {
                                            paramKeys = Object.keys(ps)
                                        }
                                        return (
                                            <InputGroup style={{display: show}} key={i} className="mb-3" size="lg">
                                                <Form.Select size="lg" aria-label={v.label}>
                                                    <option>param key</option>
                                                    {paramKeys.map((v, i) => (<option key={i} value={v}>{v}</option>))}
                                                </Form.Select></InputGroup>
                                        )
                                    case "param_space":
                                        return (
                                            <InputGroup
                                                style={{display: show}} key={i} className="mb-3" size="lg">
                                                <Form.Select size="lg" onChange={onChange} aria-label={v.label}>
                                                    <option>param space</option>
                                                    <option value="Consensus">consensus</option>
                                                    <option value="Validator">validator</option>
                                                    <option value="Governance">governance</option>
                                                    <option value="Fee">fee</option>
                                                </Form.Select>
                                            </InputGroup>
                                        )
                                }
                            }
                            return (
                                <InputGroup style={{display: show}} key={i} className="mb-3" size="lg">
                                    {withTooltip(
                                        <InputGroup.Text className="param-input">
                                            {v.inputText}
                                        </InputGroup.Text>, v.tooltip, i)}
                                    <Form.Control placeholder={v.placeholder} required={v.required} defaultValue={v.defaultValue}
                                                  type={v.type} min={0} minLength={v.minLength} maxLength={v.maxLength} aria-label={v.label}
                                    />
                                </InputGroup>
                            )
                        })}
                        {renderJSONViewer()}
                    </Modal.Body>
                    <Modal.Footer>
                        {renderButtons()}
                    </Modal.Footer>
                </Form>
            </Modal>
        </div>
    </>
}