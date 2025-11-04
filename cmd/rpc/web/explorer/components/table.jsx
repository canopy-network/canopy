import { Table } from "react-bootstrap";
import Truncate from "react-truncate-inside";
import {
  cpyObj,
  convertIfTime,
  convertTime,
  isHex,
  isNumber,
  pagination,
  upperCaseAndRepUnderscore,
  convertTx,
  toCNPY,
  formatLocaleNumber,
  defaultNetAddress,
} from "@/components/util";

// convertValue() converts the value based on its key and handles different types
function convertValue(k, v, openModal) {
  if (k === "Id" || k === "Data") return v;
  if (k === "publicKey") return <Truncate text={v} />;
  if (k === "netAddress") return <span className="net-address">{v || defaultNetAddress}</span>;
  if (isHex(v) || k === "height") {
    const content = isNumber(v) ? v : <Truncate text={v} />;
    return (
      <a href="#" onClick={() => openModal(v)} style={{ cursor: "pointer" }}>
        {content}
      </a>
    );
  }
  if (k.includes("time")) return convertTime(v);
  if (isNumber(v)) return formatLocaleNumber(v, 0, 6);
  return convertIfTime(k, v);
}

// convertTransaction() converts a transaction item into a display object
export function convertTransaction(v) {
  let value = Object.assign({}, v);
  delete value.transaction;
  return convertTx(value);
}

// sortData() sorts table data by a given column and direction
function sortData(data, column, direction) {
  if (!column) return data;
  return [...data].sort((a, b) => {
    const aValue = a[column];
    const bValue = b[column];
    if (aValue < bValue) return direction === "asc" ? -1 : 1;
    if (aValue > bValue) return direction === "asc" ? 1 : -1;
    return 0;
  });
}

// filterData() filters table data based on the filterText
function filterData(data, filterText) {
  if (!filterText) return data;
  return data.filter((row) =>
    Object.values(row).some((value) => value?.toString().toLowerCase().includes(filterText.toLowerCase())),
  );
}

// convertBlock() processes block header, removing specific fields for table
function convertBlock(v) {
  let {
    lastQuorumCertificate,
    nextValidatorRoot,
    stateRoot,
    transactionRoot,
    validatorRoot,
    lastBlockHash,
    networkID,
    totalVDFIterations,
    vdf,
    ...value
  } = cpyObj(v.blockHeader);
  value.numTxs = "numTxs" in v.blockHeader ? v.blockHeader.numTxs : "0";
  value.totalTxs = "totalTxs" in v.blockHeader ? v.blockHeader.totalTxs : "0";
  return JSON.parse(JSON.stringify(value, ["height", "hash", "time", "numTxs", "totalTxs", "proposerAddress"], 4));
}

// convertValidator() processes validator details, converting uCNPY values to CNPY
function convertValidator(v) {
  let value = Object.assign({}, v);
  value.stakedAmount = toCNPY(value.stakedAmount);
  value.committees = value.committees.join(",");
  return value;
}

// converAccount() processes account details, converting uCNPY values to CNPY
function convertAccount(v) {
  let value = Object.assign({}, v);
  value.amount = toCNPY(value.amount);
  return value;
}

// convertParams() processes different consensus parameters for table structure
function convertGovernanceParams(v) {
  if (!v.consensus) return ["0"];
  let value = cpyObj(v);
  let toCNPYParams = [
    "sendFee",
    "stakeFee",
    "editStakeFee",
    "unstakeFee",
    "pauseFee",
    "unpauseFee",
    "changeParameterFee",
    "daoTransferFee",
    "subsidyFee",
    "createOrderFee",
    "editOrderFee",
    "deleteOrderFee",
    "minimumOrderSize",
  ];
  return ["consensus", "validator", "fee", "governance"].flatMap((space) =>
    Object.entries(value[space] || {}).map(([k, v]) => ({
      ParamName: k,
      ParamValue: toCNPYParams.includes(k) ? toCNPY(v) : v,
      ParamSpace: space,
    })),
  );
}

// convertOrder() transforms order details into a table-compatible convert
function convertOrder(v) {
  const exchangeRate = v.requestedAmount / v.amountForSale;
  return {
    Id: v.id ?? "error",
    Chain: v.committee,
    Data: v.data,
    AmountForSale: toCNPY(v.amountForSale),
    Rate: exchangeRate.toFixed(2),
    RequestedAmount: toCNPY(v.requestedAmount),
    SellerReceiveAddress: v.sellerReceiveAddress,
    SellersSendAddress: v.sellersSendAddress,
    BuyerSendAddress: v.buyerSendAddress,
    Status: "buyerReceiveAddress" in v ? "Reserved" : "Open",
    BuyerReceiveAddress: v.buyerReceiveAddress,
    BuyerChainDeadline: v.buyerChainDeadline,
  };
}

// convertCommitteeSupply() calculates supply percentage for table display
function convertCommitteeSupply(v, total) {
  const percent = 100 * (v.amount / total);
  return {
    Chain: v.id,
    stake_cut: `${percent}%`,
    total_restake: toCNPY(v.amount),
  };
}

// getHeader() returns the appropriate header for the table based on the object type
function getHeader(v) {
  if (v.type === "tx-results-page") return "Transactions";
  if (v.type === "pending-results-page") return "Pending";
  if (v.type === "block-results-page") return "Blocks";
  if (v.type === "accounts") return "Accounts";
  if (v.type === "validators") return "Validators";
  if ("consensus" in v) return "Governance";
  if ("committeeStaked" in v) return "Committees";
  return "Sell Orders";
}

// getTableBody() determines the body of the table based on the provided object type
function getTableBody(v) {
  let empty = [{ Results: "null" }];
  if ("consensus" in v) return convertGovernanceParams(v);
  if ("committeeStaked" in v) return v.committeeStaked.map((item) => convertCommitteeSupply(item, v.staked));
  if (!v.hasOwnProperty("type"))
    return v[0]?.orders?.filter((order) => order.sellersSendAddress).map(convertOrder) || empty;
  if (v.results === null) return empty;
  const converters = {
    "tx-results-page": convertTransaction,
    "pending-results-page": convertTransaction,
    "block-results-page": convertBlock,
    accounts: convertAccount,
    // validators: (item) => item,
    validators: convertValidator,
  };
  let results = v.results.map(converters[v.type] || (() => []));
  return results.length === 0 ? empty : results;
}

// DTable() renders the main data table with sorting, filtering, and pagination
export default function DTable(props) {
  const { filterText, sortColumn, sortDirection, category, committee, tableData, tableLoading } = props.state;
  const sortedData = sortData(filterData(getTableBody(tableData), filterText), sortColumn, sortDirection);
  return (
    <div className="data-table">
      <div className="data-table-content">
        {category === 6 && (
          <input
            type="number"
            value={committee}
            min="1"
            onChange={(e) => e.target.value && props.selectTable(6, 0, Number(e.target.value))}
            className="chain-table mb-3"
            style={{ backgroundImage: 'url("./chain.png")' }}
          />
        )}
        <input
          type="text"
          value={filterText}
          onChange={(e) => props.setState({ ...props.state, filterText: e.target.value })}
          className="search-table mb-3"
          style={{ backgroundImage: 'url("./filter.png")' }}
        />
        <h5 className="data-table-head">{getHeader(tableData)}</h5>
      </div>

      <Table
        responsive
        bordered
        hover
        size="sm"
        className="table"
        style={{ opacity: tableLoading ? 0.6 : 1, transition: "opacity 0.2s" }}
      >
        <thead>
          <tr>
            {Object.keys(getTableBody(tableData)[0]).map((s, i) => (
              <th
                key={i}
                className="table-head"
                onClick={() => {
                  if (!tableLoading) {
                    const direction = sortColumn === s && sortDirection === "asc" ? "desc" : "asc";
                    props.setState({ ...props.state, sortColumn: s, sortDirection: direction });
                  }
                }}
                style={{ cursor: tableLoading ? "wait" : "pointer" }}
              >
                {upperCaseAndRepUnderscore(s)}
                {sortColumn === s && (sortDirection === "asc" ? " ↑" : " ↓")}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {sortedData.map((val, idx) => (
            <tr key={idx}>
              {Object.keys(val).map((k, i) => (
                <td
                  key={i}
                  className={k === "Id" ? "large-table-col" : k === "netAddress" ? "net-address-col" : "table-col"}
                >
                  {convertValue(k, val[k], props.openModal)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </Table>

      {pagination(tableData, (i) => props.selectTable(props.state.category, i))}
    </div>
  );
}
