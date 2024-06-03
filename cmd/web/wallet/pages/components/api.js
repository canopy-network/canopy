export const rpcURL = "http://127.0.0.1:6001"
export const adminRPCURL = "http://127.0.0.1:6002"

const keystorePath = "/v1/admin/keystore"
const keystoreGetPath = "/v1/admin/keystore-get"
const keystoreNewPath = "/v1/admin/keystore-new-key"
const keystoreImportPath = "/v1/admin/keystore-import-raw"
export const logsPath = "/v1/admin/log"
const resourcePath = "/v1/admin/resource-usage"
const txSendPath = "/v1/admin/tx-send"
const txStakePath = "/v1/admin/tx-stake"
const txEditStakePath = "/v1/admin/tx-edit-stake"
const txUnstakePath = "/v1/admin/tx-unstake"
const txPausePath = "/v1/admin/tx-pause"
const txUnpausePath = "/v1/admin/tx-unpause"
const txChangeParamPath = "/v1/admin/tx-change-param"
const txDaoTransfer = "/v1/admin/tx-dao-transfer"
export const consensusInfoPath = "/v1/admin/consensus-info"
export const configPath = "/v1/admin/config"
export const peerBookPath = "/v1/admin/peer-book"
export const peerInfoPath = "/v1/admin/peer-info"
const accountPath = "/v1/query/account"
const validatorPath = "/v1/query/validator"
const txsBySender = "/v1/query/txs-by-sender"
const txsByRec = "/v1/query/txs-by-rec"
const pollPath = "/v1/gov/poll"
const proposalsPath = "/v1/gov/proposals"
const addVotePath = "/v1/gov/add-vote"
const delVotePath = "/v1/gov/del-vote"
const paramsPath = "/v1/query/params"
const txPath = "/v1/tx"

export async function GET(url, path) {
    let resp = await fetch(url + path, {
        method: 'GET',
    })
        .catch(rejected => {
            console.log(rejected);
        });
    if (resp == null) {
        return {}
    }
    return resp.json()
}

export async function GETText(url, path) {
    let resp = await fetch(url + path, {
        method: 'GET',
    })
        .catch(rejected => {
            console.log(rejected);
        });
    if (resp == null) {
        return {}
    }
    return resp.text()
}

export async function POST(url, path, request) {
    let resp = await fetch(url + path, {
        method: 'POST',
        body: request,
    })
        .catch(rejected => {
            console.log(rejected);
        });
    if (resp == null) {
        return {}
    }
    return resp.json()
}

function heightAndAddrRequest(height, address) {
    return `{"height":` + height + `, "address":"` + address + `"}`
}

function pageAddrReq(page, addr) {
    return `{"address":"` + addr + `", "pageNumber":` + page + `, "perPage":5}`
}

function voteRequest(json, approve) {
    const request = {}
    request.approve = approve
    request.proposal = json
    return JSON.stringify(request)
}

function addressAndPwdRequest(address, password) {
    const request = {}
    request.address = address
    request.password = password
    return JSON.stringify(request)
}

function pkAndPwdRequest(pk, password) {
    const request = {}
    request.privateKey = pk
    request.password = password
    return JSON.stringify(request)
}

function newTxRequest(address, netAddress, amount, output, sequence, fee, submit, password) {
    const request = {}
    request.address = address
    request.netAddress = netAddress
    request.amount = amount
    request.output = output
    request.sequence = sequence
    request.fee = fee
    request.submit = submit
    request.password = password
    return JSON.stringify(request)
}

function newGovTxRequest(address, amount, paramSpace, paramKey, paramValue, startBlock, endBlock, sequence, fee, submit, password) {
    const request = {}
    request.address = address
    request.amount = amount
    request.paramSpace = paramSpace
    request.paramKey = paramKey
    request.paramValue = paramValue
    request.startBlock = startBlock
    request.endBlock = endBlock
    request.sequence = sequence
    request.fee = fee
    request.submit = submit
    request.password = password
    return JSON.stringify(request)
}

export async function Keystore() {
    return GET(adminRPCURL, keystorePath)
}

export async function KeystoreGet(address, password) {
    return POST(adminRPCURL, keystoreGetPath, addressAndPwdRequest(address, password))
}

export async function KeystoreNew(password) {
    return POST(adminRPCURL, keystoreNewPath, addressAndPwdRequest("", password))
}

export async function KeystoreImport(pk, password) {
    return POST(adminRPCURL, keystoreImportPath, pkAndPwdRequest(pk, password))
}

export async function Logs() {
    return GETText(adminRPCURL, logsPath)
}

export async function Account(height, address) {
    return POST(rpcURL, accountPath, heightAndAddrRequest(height, address))
}

export async function Poll() {
    return GET(rpcURL, pollPath)
}

export async function Proposals() {
    return GET(rpcURL, proposalsPath)
}

export async function AddVote(json, approve) {
    return POST(adminRPCURL, addVotePath, voteRequest(JSON.parse(json), approve))
}

export async function DelVote(json) {
    return POST(adminRPCURL, delVotePath, voteRequest(JSON.parse(json)))
}

export async function AccountWithTxs(height, address, page) {
    let result = {}
    result.account = await Account(height, address)
    result.sent_transactions = await TransactionsBySender(page, address)
    result.rec_transactions = await TransactionsByRec(page, address)
    result.combined = []
    if (result.sent_transactions.results != null && result.rec_transactions.results != null) {
        result.combined = result.combined.concat(result.rec_transactions.results, result.sent_transactions.results)
    } else if (result.sent_transactions.results != null) {
        result.combined = result.sent_transactions.results
    } else if (result.rec_transactions.results != null) {
        result.combined = result.rec_transactions.results
    } else {
        return result
    }
    result.combined.sort(function (a, b) {
        return b.height === a.height ? b.index - a.index : b.height - a.height
    });
    return result
}

export function TransactionsBySender(page, sender) {
    return POST(rpcURL, txsBySender, pageAddrReq(page, sender))
}

export function TransactionsByRec(page, rec) {
    return POST(rpcURL, txsByRec, pageAddrReq(page, rec))
}

export async function Validator(height, address) {
    return POST(rpcURL, validatorPath, heightAndAddrRequest(height, address))
}

export async function Resource() {
    return GET(adminRPCURL, resourcePath)
}

export async function TxSend(address, recipient, amount, sequence, fee, password, submit) {
    return POST(adminRPCURL, txSendPath, newTxRequest(address, "", amount, recipient, Number(sequence), Number(fee), submit, password))
}

export async function TxStake(address, netAddress, amount, output, sequence, fee, password, submit) {
    return POST(adminRPCURL, txStakePath, newTxRequest(address, netAddress, amount, output, Number(sequence), Number(fee), submit, password))
}

export async function TxEditStake(address, netAddress, amount, output, sequence, fee, password, submit) {
    return POST(adminRPCURL, txEditStakePath, newTxRequest(address, netAddress, amount, output, Number(sequence), Number(fee), submit, password))
}

export async function TxUnstake(address, sequence, fee, password, submit) {
    return POST(adminRPCURL, txUnstakePath, newTxRequest(address, "", 0, "", Number(sequence), Number(fee), submit, password))
}

export async function TxPause(address, sequence, fee, password, submit) {
    return POST(adminRPCURL, txPausePath, newTxRequest(address, "", 0, "", Number(sequence), Number(fee), submit, password))
}

export async function TxUnpause(address, sequence, fee, password, submit) {
    return POST(adminRPCURL, txUnpausePath, newTxRequest(address, "", 0, "", Number(sequence), Number(fee), submit, password))
}

export async function TxChangeParameter(address, paramSpace, paramKey, paramValue, startBlock, endBlock, sequence, fee, password, submit) {
    console.log(newGovTxRequest(address, 0, paramSpace, paramKey, paramValue, Number(startBlock), Number(endBlock), Number(sequence), Number(fee), submit, password))
    return POST(adminRPCURL, txChangeParamPath, newGovTxRequest(address, 0, paramSpace, paramKey, paramValue, Number(startBlock), Number(endBlock), Number(sequence), Number(fee), submit, password))
}

export async function TxDAOTransfer(address, amount, startBlock, endBlock, sequence, fee, password, submit) {
    return POST(adminRPCURL, txDaoTransfer, newGovTxRequest(address, Number(amount), "", "", "", Number(startBlock), Number(endBlock), Number(sequence), Number(fee), submit, password))
}

export async function RawTx(json) {
    console.log(json)
    return POST(rpcURL, txPath, json)
}

export async function Params(height) {
    return POST(rpcURL, paramsPath, heightAndAddrRequest(height, ""))
}

export async function ConsensusInfo() {
    return GET(adminRPCURL, consensusInfoPath)
}

export async function PeerInfo() {
    return GET(adminRPCURL, peerInfoPath)
}
