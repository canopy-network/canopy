import React from "react";
import Truncate from "react-truncate-inside";
import { JsonViewer } from "@textea/json-viewer";
import { Modal, Table, Tab, Tabs, CardGroup, Card, Toast, ToastContainer, Button } from "react-bootstrap";
import * as API from "@/components/api";
import {
  copy,
  cpyObj,
  convertIfTime,
  isEmpty,
  pagination,
  upperCaseAndRepUnderscore,
  withTooltip,
  convertTx,
  toCNPY,
} from "@/components/util";

// convertCardData() converts the data from state into a display object for rendering
function convertCardData(state, v) {
  if (!v) return { None: "" };
  const value = cpyObj(v);
  if (value.transaction) {
    delete value.transaction;
    return value;
  }
  if (value.dexBatch) {
    return {
      Committee: value.dexBatch.Committee || value.dexBatch.committee,
      Orders: value.dexBatch.orders?.length || 0,
      PoolSize: toCNPY(value.dexBatch.pool_size || value.dexBatch.poolSize || 0),
      LockedHeight: value.dexBatch.locked_height || value.dexBatch.lockedHeight || "null",
    };
  }
  return value.block
    ? {
        height: value.block.blockHeader.height,
        hash: value.block.blockHeader.hash,
        proposer: value.block.blockHeader.proposerAddress,
      }
    : value.validator && !state.modalState.accOnly
      ? {
          address: value.validator.address,
          publicKey: value.validator.publicKey,
          netAddress: value.validator.netAddress,
          outputAddress: value.validator.output,
        }
      : value.account;
}

// convertPaginated() converts a paginated item into a display object for rendering
function convertPaginated(v) {
  if (v == null || v === 0) return [0];
  if ("block" in v) return convertBlock(v) || { None: "" };
  if ("transaction" in v) return { ...v, transaction: undefined };
  return v;
}

// convertTransactions() converts an array of transactions into a suitable display object
export function convertTransactions(txs) {
  for (let i = 0; i < txs.length; i++) {
    txs[i] = convertTx(txs[i]);
  }
  return txs;
}

// convertBlock() converts a block item into a display object for rendering
export function convertBlock(blk) {
  let { lastQuorumCertificate, nextValidatorRoot, stateRoot, transactionRoot, validatorRoot, vdf, ...value } =
    blk.block.blockHeader;
  return value;
}

// convertCertificateResults() converts a qc item into a display object for rendering
export function convertCertificateResults(qc) {
  return {
    certificate_height: qc.header.height,
    network_id: qc.header.networkID,
    chain_id: qc.header.chainId,
    block_hash: qc.blockHash,
    results_hash: qc.resultsHash,
  };
}

// convertTabData() converts the modal data into specific tab display object for rendering
function convertTabData(state, v, tab) {
  if ("block" in v) {
    switch (tab) {
      case 0:
        return convertBlock(v);
      case 1:
        return v.block.transactions ? convertTransactions(v.block.transactions) : 0;
      default:
        return v.block;
    }
  } else if ("transaction" in v) {
    switch (tab) {
      case 0:
        if ("qc" in v.transaction.msg) return convertCertificateResults(v.transaction.msg.qc);
        return v.transaction.msg;
      case 1:
        return { hash: v.txHash, time: v.transaction.time, sender: v.sender, type: v.messageType };
      default:
        return v;
    }
  } else if ("validator" in v && !state.modalState.accOnly) {
    let validator = cpyObj(v.validator);
    if (validator.committees && Array.isArray(validator.committees)) {
      validator.committees = validator.committees.join(",");
    }
    if (validator.stakedAmount) {
      validator.stakedAmount = toCNPY(validator.stakedAmount);
    }
    return validator;
  } else if ("account" in v) {
    let txs = v.sent_transactions.results.length > 0 ? v.sent_transactions.results : v.rec_transactions.results;
    switch (tab) {
      case 0:
        let account = cpyObj(v.account);
        account.amount = toCNPY(account.amount);
        return account;
      case 1:
        return convertTransactions(txs);
      default:
        return convertTransactions(txs);
    }
  } else if ("dexBatch" in v) {
    switch (tab) {
      case 0: // Orders
        return v.dexBatch.orders || [];
      case 1: // Deposits
        return v.dexBatch.deposits || [];
      case 2: // Withdrawals
        return v.dexBatch.withdraws || [];
      case 3: // Pool Points
        return v.dexBatch.poolPoints || v.dexBatch.pool_points || [];
      case 4: // Raw
      default:
        return v.dexBatch;
    }
  }
}

// getModalTitle() extracts the modal title from the object
function getModalTitle(state, v) {
  if ("transaction" in v) return "Transaction";
  if ("block" in v) return "Block";
  if ("dexBatch" in v) return "Dex Batch";
  if ("validator" in v && !state.modalState.accOnly) return "Validator";
  return "Account";
}

// getTabTitle() extracts the tab title from the object
function getTabTitle(state, data, tab) {
  if ("transaction" in data) {
    return tab === 0 ? "Message" : tab === 1 ? "Meta" : "Raw";
  }
  if ("block" in data) {
    return tab === 0 ? "Header" : tab === 1 ? "Transactions" : "Raw";
  }
  if ("dexBatch" in data) {
    switch (tab) {
      case 0: return "Orders";
      case 1: return "Deposits";
      case 2: return "Withdrawals";
      case 3: return "Pool Points";
      case 4: return "Raw";
      default: return "Raw";
    }
  }
  if ("validator" in data && !state.modalState.accOnly) {
    return tab === 0 ? "Validator" : tab === 1 ? "Account" : "Raw";
  }
  return tab === 0 ? "Account" : tab === 1 ? "Sent Transactions" : "Received Transactions";
}

// DetailModal() returns the main modal component for this file
export default function DetailModal({ state, setState }) {
  const data = state.modalState.data;
  const cards = convertCardData(state, data);
  
  // Local state for filtering and sorting
  const [addressFilter, setAddressFilter] = React.useState('');
  const [sortBy, setSortBy] = React.useState('none');
  const [sortDirection, setSortDirection] = React.useState('asc');

  // check if the data is empty or no results
  if (isEmpty(data)) return <></>;

  if (data === "no result found") {
    return (
      <ToastContainer position={"top-center"} className="search-toast">
        <Toast onClose={resetState} show delay={3000} autohide>
          <Toast.Header />
          <Toast.Body className="search-toast-body">no results found</Toast.Body>
        </Toast>
      </ToastContainer>
    );
  }

  // resetState() resets the modal state back to initial
  function resetState() {
    setState({ ...state, modalState: { show: false, query: "", page: 0, data: {}, accOnly: false } });
  }

  // renderTab() renders a tab based on the state data and tab number
  function renderTab(tab) {
    if ("block" in data) {
      return tab === 0 ? renderBasicTable(tab) : tab === 1 ? renderPageTable(tab) : renderJSONViewer(tab);
    }
    if ("transaction" in data) {
      return tab === 0 ? renderBasicTable(tab) : tab === 1 ? renderBasicTable(tab) : renderJSONViewer(tab);
    }
    if ("dexBatch" in data) {
      return tab === 4 ? renderJSONViewer(tab) : renderDexBatchList(tab);
    }
    if ("validator" in data && !state.modalState.accOnly) {
      return tab === 0 ? renderBasicTable(tab) : tab === 1 ? renderTableButton() : renderJSONViewer(tab);
    }
    return tab === 0 ? renderBasicTable(tab) : renderPageTable(tab);
  }

  // renderBasicTable() organizes the data into a table based on the tab number
  function renderBasicTable(tab) {
    const body = convertTabData(state, data, tab);
    return (
      <Table responsive>
        <tbody>
          {Object.keys(body).map((k, i) => (
            <tr key={i}>
              <td className="detail-table-title">{upperCaseAndRepUnderscore(k)}</td>
              <td className="detail-table-info">{convertIfTime(k, body[k])}</td>
            </tr>
          ))}
        </tbody>
      </Table>
    );
  }

  // renderPageTable() organizes the data into a paginated table based on the tab number
  function renderPageTable(tab) {
    let start = 0,
      end = 10,
      page = [0],
      d = data,
      ms = state.modalState,
      blk = d.block;
    if ("block" in d) {
      end = ms.page === 0 || ms.page === 1 ? 10 : ms.page * 10;
      start = end - 10;
      page = blk.transactions || page;
      d = { pageNumber: ms.Page, perPage: 10, totalPages: Math.ceil(blk.blockHeader.num_txs / 10), ...d };
    } else if ("account" in d) {
      page =
        tab === 1 ? convertTransactions(d.sent_transactions.results) : convertTransactions(d.rec_transactions.results);
      d = tab === 1 ? d.sent_transactions : d.rec_transactions;
    }
    return (
      <>
        <Table responsive>
          <tbody>
            <tr>
              {Object.keys(convertPaginated(convertTabData(state, data, 1)[0])).map((k, i) => (
                <td key={i} className="detail-table-row-title">
                  {upperCaseAndRepUnderscore(k)}
                </td>
              ))}
            </tr>
            {page.slice(start, end).map((item, key) => (
              <tr key={key}>
                {Object.keys(convertPaginated(item)).map((k, i) => (
                  <td key={i} className="detail-table-row-info">
                    {convertIfTime(k, item[k])}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </Table>
        {pagination(d, (p) =>
          API.getModalData(ms.query, p).then((r) => {
            setState({ ...state, modalState: { ...ms, show: true, query: ms.query, page: p, data: r } });
          }),
        )}
      </>
    );
  }

  // renderJSONViewer() renders a raw json display
  function renderJSONViewer(tab) {
    return <JsonViewer rootName={"result"} defaultInspectDepth={1} value={convertTabData(state, data, tab)} />;
  }

  // filterAndSortItems() filters and sorts items based on current filters
  function filterAndSortItems(items, tab) {
    if (!items || items.length === 0) return items;
    
    // Filter by address if filter is set
    let filteredItems = items;
    if (addressFilter) {
      filteredItems = items.filter(item => {
        // Check all fields for address-like values
        return Object.values(item).some(value => 
          value && value.toString().toLowerCase().includes(addressFilter.toLowerCase())
        );
      });
    }
    
    // Sort by amount/points if sortBy is set
    if (sortBy !== 'none') {
      filteredItems = [...filteredItems].sort((a, b) => {
        let aValue = 0;
        let bValue = 0;
        
        // Determine sort field based on tab and sortBy selection
        if (sortBy === 'amount') {
          // Look for amount-related fields
          const amountFields = ['amount', 'amountForSale', 'requestedAmount', 'points'];
          const aField = amountFields.find(field => field in a);
          const bField = amountFields.find(field => field in b);
          aValue = aField ? parseFloat(a[aField]) || 0 : 0;
          bValue = bField ? parseFloat(b[bField]) || 0 : 0;
        }
        
        return sortDirection === 'asc' ? aValue - bValue : bValue - aValue;
      });
    }
    
    return filteredItems;
  }

  // renderDexBatchList() renders a list view for DexBatch components
  function renderDexBatchList(tab) {
    const items = convertTabData(state, data, tab);
    
    if (!items || items.length === 0) {
      return <div className="text-center p-4">No items found</div>;
    }

    const filteredAndSortedItems = filterAndSortItems(items, tab);

    return (
      <div className="dex-batch-list">
        {/* Filter and Sort Controls */}
        <div className="mb-3 p-3 border rounded bg-light">
          <div className="row">
            <div className="col-md-6">
              <label className="form-label">Filter by Address:</label>
              <input
                type="text"
                className="form-control form-control-sm"
                placeholder="Enter address to filter..."
                value={addressFilter}
                onChange={(e) => setAddressFilter(e.target.value)}
              />
            </div>
            <div className="col-md-3">
              <label className="form-label">Sort by:</label>
              <select
                className="form-select form-select-sm"
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value)}
              >
                <option value="none">No sorting</option>
                <option value="amount">Amount/Points</option>
              </select>
            </div>
            <div className="col-md-3">
              <label className="form-label">Direction:</label>
              <select
                className="form-select form-select-sm"
                value={sortDirection}
                onChange={(e) => setSortDirection(e.target.value)}
                disabled={sortBy === 'none'}
              >
                <option value="asc">Ascending</option>
                <option value="desc">Descending</option>
              </select>
            </div>
          </div>
          <div className="mt-2 text-muted small">
            Showing {filteredAndSortedItems.length} of {items.length} items
          </div>
        </div>

        {/* Filtered Items List */}
        {filteredAndSortedItems.length === 0 ? (
          <div className="text-center p-4">No items match the current filter</div>
        ) : (
          filteredAndSortedItems.map((item, index) => (
            <div key={index} className="dex-batch-item mb-3 p-3 border rounded">
              <Table responsive size="sm">
                <tbody>
                  {Object.keys(item).map((key, i) => (
                    <tr key={i}>
                      <td className="detail-table-title" style={{ width: '30%' }}>
                        {upperCaseAndRepUnderscore(key)}
                      </td>
                      <td className="detail-table-info">
                        {key === 'address' || key === 'Address' ? (
                          <Truncate text={item[key]} />
                        ) : key.toLowerCase().includes('amount') || key.toLowerCase().includes('points') ? (
                          toCNPY(item[key] || 0)
                        ) : (
                          item[key]?.toString() || 'null'
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </Table>
            </div>
          ))
        )}
      </div>
    );
  }

  // renderTableButtons() renders a button to display the account
  function renderTableButton() {
    return (
      <Button
        className="open-acc-details-btn"
        variant="outline-secondary"
        onClick={() => setState({ ...state, modalState: { ...state.modalState, accOnly: true } })}
      >
        Open Account Details
      </Button>
    );
  }

  let toCNPYFields = ["amount", "stakedAmount"];

  // return the Modal
  return (
    <Modal size="xl" show={state.modalState.show} onHide={resetState}>
      <Modal.Header closeButton />
      <Modal.Body className="modal-body">
        {/* TITLE */}
        <h3 className="modal-header">
          <div className="modal-header-icon">
            <svg id="svg" version="1.1" width="400" height="400" viewBox="0, 0, 400,400">
              <g id="svgg">
                <path
                  id="path0"
                  d="M156.013 18.715 C 21.871 46.928,-30.448 226.543,66.017 327.677 C 136.809 401.895,253.592 404.648,327.818 333.848 C 462.974 204.931,340.320 -20.049,156.013 18.715 M215.200 96.800 C 217.840 99.440,220.000 106.280,220.000 112.000 C 220.000 130.024,197.388 139.788,184.800 127.200 C 182.160 124.560,180.000 117.720,180.000 112.000 C 180.000 106.280,182.160 99.440,184.800 96.800 C 187.440 94.160,194.280 92.000,200.000 92.000 C 205.720 92.000,212.560 94.160,215.200 96.800 M216.000 228.000 C 216.000 285.333,216.356 288.000,224.000 288.000 C 229.333 288.000,232.000 290.667,232.000 296.000 C 232.000 303.333,229.333 304.000,200.000 304.000 C 170.667 304.000,168.000 303.333,168.000 296.000 C 168.000 290.667,170.667 288.000,176.000 288.000 C 183.590 288.000,184.000 285.333,184.000 236.000 C 184.000 186.667,183.590 184.000,176.000 184.000 C 170.667 184.000,168.000 181.333,168.000 176.000 C 168.000 168.889,170.667 168.000,192.000 168.000 L 216.000 168.000 216.000 228.000 "
                  stroke="none"
                  fillRule="evenodd"
                ></path>
              </g>
            </svg>
          </div>
          {getModalTitle(state, data)} Details
        </h3>
        {/* CARDS */}
        <CardGroup className="modal-card-group">
          {Object.keys(cards).map((k, i) => {
            return withTooltip(
              <Card onClick={() => copy(state, setState, cards[k])} key={i} className="modal-cards">
                <Card.Body className="modal-card">
                  <h5 className="modal-card-title">{k}</h5>
                  <div className="modal-card-detail">
                    <Truncate text={String(toCNPYFields.includes(k) ? toCNPY(cards[k]) : cards[k])} />
                  </div>
                  <img className="copy-img" src="./copy.png" alt="copy" />
                </Card.Body>
              </Card>,
              cards[k],
              i,
              "top",
            );
          })}
        </CardGroup>
        {/* TABS */}
        <Tabs defaultActiveKey="0" id="modal-tab" className="mb-3" fill>
          {[...Array("dexBatch" in data ? 5 : 3)].map((_, i) => (
            <Tab key={i} tabClassName="rb-tab" eventKey={i} title={getTabTitle(state, data, i)}>
              {renderTab(i)}
            </Tab>
          ))}
        </Tabs>
      </Modal.Body>
    </Modal>
  );
}
