// API methods adapted from wallet original for wallet-new
let rpcURL = "http://localhost:50002"; // default RPC URL
let adminRPCURL = "http://localhost:50003"; // default Admin RPC URL
let chainId = 1; // default chain id

if (typeof window !== "undefined") {
  if (window.__CONFIG__) {
    rpcURL = window.__CONFIG__.rpcURL;
    adminRPCURL = window.__CONFIG__.adminRPCURL;
    chainId = Number(window.__CONFIG__.chainId);
  }
  rpcURL = rpcURL.replace("localhost", window.location.hostname);
  adminRPCURL = adminRPCURL.replace("localhost", window.location.hostname);
} else {
  console.log("config undefined");
}

export function getAdminRPCURL() {
  return adminRPCURL;
}

export function getRPCURL() {
  return rpcURL;
}

// API Paths
const keystorePath = "/v1/admin/keystore";
const keystoreGetPath = "/v1/admin/keystore-get";
const keystoreNewPath = "/v1/admin/keystore-new-key";
const keystoreImportPath = "/v1/admin/keystore-import-raw";
export const logsPath = "/v1/admin/log";
const resourcePath = "/v1/admin/resource-usage";
const txSendPath = "/v1/admin/tx-send";
const txStakePath = "/v1/admin/tx-stake";
const txEditStakePath = "/v1/admin/tx-edit-stake";
const txUnstakePath = "/v1/admin/tx-unstake";
const txPausePath = "/v1/admin/tx-pause";
const txUnpausePath = "/v1/admin/tx-unpause";
const txChangeParamPath = "/v1/admin/tx-change-param";
const txDaoTransfer = "/v1/admin/tx-dao-transfer";
const txCreateOrder = "/v1/admin/tx-create-order";
const txLockOrder = "/v1/admin/tx-lock-order";
const txCloseOrder = "/v1/admin/tx-close-order";
const txEditOrder = "/v1/admin/tx-edit-order";
const txDeleteOrder = "/v1/admin/tx-delete-order";
const txStartPoll = "/v1/admin/tx-start-poll";
const txVotePoll = "/v1/admin/tx-vote-poll";
export const consensusInfoPath = "/v1/admin/consensus-info?id=1";
export const configPath = "/v1/admin/config";
export const peerBookPath = "/v1/admin/peer-book";
export const peerInfoPath = "/v1/admin/peer-info";
const accountPath = "/v1/query/account";
const validatorPath = "/v1/query/validator";
const validatorsPath = "/v1/query/validators";
const lastProposersPath = "/v1/query/last-proposers";
const ecoParamsPath = "/v1/query/eco-params";
const txsBySender = "/v1/query/txs-by-sender";
const txsByRec = "/v1/query/txs-by-rec";
const failedTxs = "/v1/query/failed-txs";
const pollPath = "/v1/gov/poll";
const proposalsPath = "/v1/gov/proposals";
const addVotePath = "/v1/gov/add-vote";
const delVotePath = "/v1/gov/del-vote";
const paramsPath = "/v1/query/params";
const orderPath = "/v1/query/order";
const txPath = "/v1/tx";
const height = "/v1/query/height";

// HTTP Methods
export async function GET(url: string, path: string) {
  return fetch(url + path, {
    method: "GET",
  })
    .then(async (response) => {
      if (!response.ok) {
        return Promise.reject(response);
      }
      return response.json();
    })
    .catch((rejected) => {
      console.log(rejected);
      return Promise.reject(rejected);
    });
}

export async function GETText(url: string, path: string) {
  return fetch(url + path, {
    method: "GET",
  })
    .then(async (response) => {
      if (!response.ok) {
        return Promise.reject(response);
      }
      return response.text();
    })
    .catch((rejected) => {
      console.log(rejected);
      return Promise.reject(rejected);
    });
}

export async function POST(url: string, path: string, request: string) {
  return fetch(url + path, {
    method: "POST",
    body: request,
  })
    .then(async (response) => {
      if (!response.ok) {
        return Promise.reject(response);
      }
      return response.json();
    })
    .catch((rejected) => {
      console.log(rejected);
      return Promise.reject(rejected);
    });
}

// Helper functions
function heightAndAddrRequest(height: number, address: string) {
  return JSON.stringify({ height: height, address: address });
}

function pageAddrReq(page: number, addr: string) {
  return JSON.stringify({ pageNumber: page, address: addr, perPage: 5 });
}

// API Functions
export async function Keystore() {
  return GET(adminRPCURL, keystorePath);
}

export async function KeystoreGet(address: string, password: string, nickname: string) {
  const request = JSON.stringify({ address: address, password: password, nickname: nickname, submit: true });
  return POST(adminRPCURL, keystoreGetPath, request);
}

export async function KeystoreNew(password: string, nickname: string) {
  const request = JSON.stringify({ address: "", password: password, nickname: nickname, submit: true });
  return POST(adminRPCURL, keystoreNewPath, request);
}

export async function KeystoreImport(pk: string, password: string, nickname: string) {
  const request = JSON.stringify({ privateKey: pk, password: password, nickname: nickname });
  return POST(adminRPCURL, keystoreImportPath, request);
}

export async function Logs() {
  return GETText(adminRPCURL, logsPath);
}

export async function Account(height: number, address: string) {
  return POST(rpcURL, accountPath, heightAndAddrRequest(height, address));
}

export async function Height() {
  return POST(rpcURL, height, JSON.stringify({}));
}


export async function TransactionsBySender(page: number, sender: string) {
  return POST(rpcURL, txsBySender, pageAddrReq(page, sender));
}

export async function TransactionsByRec(page: number, rec: string) {
  return POST(rpcURL, txsByRec, pageAddrReq(page, rec));
}

export async function FailedTransactions(page: number, sender: string) {
  return POST(rpcURL, failedTxs, pageAddrReq(page, sender));
}

export async function Validator(height: number, address: string) {
  return POST(rpcURL, validatorPath, heightAndAddrRequest(height, address));
}

export async function Validators(height: number) {
  return POST(rpcURL, validatorsPath, heightAndAddrRequest(height, ""));
}

export async function LastProposers(height: number) {
  return POST(rpcURL, lastProposersPath, heightAndAddrRequest(height, ""));
}

export async function EcoParams(height: number) {
  return POST(rpcURL, ecoParamsPath, heightAndAddrRequest(height, ""));
}

export async function Resource() {
  return GET(adminRPCURL, resourcePath);
}

export async function ConsensusInfo() {
  return GET(adminRPCURL, consensusInfoPath);
}

export async function PeerInfo() {
  return GET(adminRPCURL, peerInfoPath);
}

export async function Params(height: number) {
  return POST(rpcURL, paramsPath, heightAndAddrRequest(height, ""));
}

export async function Poll() {
  return GET(rpcURL, pollPath);
}

export async function Proposals() {
  return GET(rpcURL, proposalsPath);
}

// Transaction functions
export async function TxSend(address: string, recipient: string, amount: number, memo: string, fee: number, password: string, submit: boolean) {
  const request = JSON.stringify({
    address: address,
    pubKey: "",
    netAddress: "",
    committees: "",
    amount: amount,
    delegate: false,
    earlyWithdrawal: false,
    output: recipient,
    signer: "",
    memo: memo,
    fee: Number(fee),
    submit: submit,
    password: password,
  });
  return POST(adminRPCURL, txSendPath, request);
}

export async function TxStake(
  address: string,
  pubKey: string,
  committees: string,
  netAddress: string,
  amount: number,
  delegate: boolean,
  earlyWithdrawal: boolean,
  output: string,
  signer: string,
  memo: string,
  fee: number,
  password: string,
  submit: boolean,
) {
  const request = JSON.stringify({
    address: address,
    pubKey: pubKey,
    netAddress: netAddress,
    committees: committees,
    amount: amount,
    delegate: delegate,
    earlyWithdrawal: earlyWithdrawal,
    output: output,
    signer: signer,
    memo: memo,
    fee: Number(fee),
    submit: submit,
    password: password,
  });
  return POST(adminRPCURL, txStakePath, request);
}

export async function TxUnstake(address: string, signer: string, memo: string, fee: number, password: string, submit: boolean) {
  const request = JSON.stringify({
    address: address,
    pubKey: "",
    netAddress: "",
    committees: "",
    amount: 0,
    delegate: false,
    earlyWithdrawal: false,
    output: "",
    signer: signer,
    memo: memo,
    fee: Number(fee),
    submit: submit,
    password: password,
  });
  return POST(adminRPCURL, txUnstakePath, request);
}

export async function TxPause(address: string, signer: string, memo: string, fee: number, password: string, submit: boolean) {
  const request = JSON.stringify({
    address: address,
    pubKey: "",
    netAddress: "",
    committees: "",
    amount: 0,
    delegate: false,
    earlyWithdrawal: false,
    output: "",
    signer: signer,
    memo: memo,
    fee: Number(fee),
    submit: submit,
    password: password,
  });
  return POST(adminRPCURL, txPausePath, request);
}

export async function TxUnpause(address: string, signer: string, memo: string, fee: number, password: string, submit: boolean) {
  const request = JSON.stringify({
    address: address,
    pubKey: "",
    netAddress: "",
    committees: "",
    amount: 0,
    delegate: false,
    earlyWithdrawal: false,
    output: "",
    signer: signer,
    memo: memo,
    fee: Number(fee),
    submit: submit,
    password: password,
  });
  return POST(adminRPCURL, txUnpausePath, request);
}

// Combined account data with transactions
export async function AccountWithTxs(height: number, address: string, nickname: string, page: number) {
  let result: any = {};
  result.account = await Account(height, address);
  result.account.nickname = nickname;

  const setStatus = (status: string) => (tx: any) => {
    tx.status = status;
  };

  result.sent_transactions = await TransactionsBySender(page, address);
  result.sent_transactions.results?.forEach(setStatus("included"));

  result.rec_transactions = await TransactionsByRec(page, address);
  result.rec_transactions.results?.forEach(setStatus("included"));

  result.failed_transactions = await FailedTransactions(page, address);
  result.failed_transactions.results?.forEach((tx: any) => {
    tx.status = "failure: ".concat(tx.error.msg);
  });

  result.combined = (result.rec_transactions.results || [])
    .concat(result.sent_transactions.results || [])
    .concat(result.failed_transactions.results || []);

  result.combined.sort(function (a: any, b: any) {
    return a.transaction.time !== b.transaction.time
      ? b.transaction.time - a.transaction.time
      : a.height !== b.height
        ? b.height - a.height
        : b.index - a.index;
  });

  return result;
}
