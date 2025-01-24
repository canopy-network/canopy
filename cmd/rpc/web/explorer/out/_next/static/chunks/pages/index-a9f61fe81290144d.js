(self.webpackChunk_N_E = self.webpackChunk_N_E || []).push([
  [405],
  {
    5557: function (e, t, a) {
      (window.__NEXT_P = window.__NEXT_P || []).push([
        "/",
        function () {
          return a(1);
        },
      ]);
    },
    1: function (e, t, a) {
      "use strict";
      a.r(t),
        a.d(t, {
          default: function () {
            return ep;
          },
        });
      var n = a(5893),
        s = a(3353),
        r = a(7913),
        i = a(5955),
        l = a(7294),
        c = a(196),
        o = a(2854),
        d = a(3630);
      function u(e) {
        return e.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
      }
      function h(e) {
        let t = arguments.length > 1 && void 0 !== arguments[1] ? arguments[1] : 1e6;
        return Number(e) < t
          ? u(e)
          : Intl.NumberFormat("en", { notation: "compact", maximumSignificantDigits: 3 }).format(e);
      }
      function m(e) {
        return new Date(Math.floor(e / 1e3)).toLocaleTimeString();
      }
      function p(e, t) {
        return e.includes("time") ? m(t) : "boolean" == typeof t ? String(t) : t;
      }
      function g(e, t, a) {
        let s = arguments.length > 3 && void 0 !== arguments[3] ? arguments[3] : "right";
        return (0, n.jsx)(
          c.Z,
          {
            placement: s,
            delay: { show: 250, hide: 400 },
            overlay: (0, n.jsx)(o.Z, { id: "button-tooltip", children: t }),
            children: e,
          },
          a,
        );
      }
      function b(e) {
        return !isNaN(parseFloat(e)) && !isNaN(e - 0);
      }
      function f(e) {
        let t,
          a = e.split("_");
        for (t = 0; t < a.length; t++) a[t] = a[t].charAt(0).toUpperCase() + a[t].slice(1);
        return a.join(" ");
      }
      function x(e) {
        return Object.assign({}, e);
      }
      function y(e, t) {
        let a = [];
        if ("perPage" in e) {
          let s = e.pageNumber - 2;
          s <= 0 && (s = 1);
          for (let r = s; r <= Math.min(Math.ceil(e.totalPages), s + 5); r++)
            a.push((0, n.jsx)(d.Z.Item, { onClick: () => t(r), active: r === e.pageNumber, children: r }, r));
        }
        return (0, n.jsxs)(d.Z, { className: "pagination", children: [a, (0, n.jsx)(d.Z.Ellipsis, {})] });
      }
      function v(e) {
        return (
          null == e.recipient && (e.recipient = e.sender),
          ("index" in e && 0 !== e.index) || (e.index = 0),
          (e = JSON.parse(
            JSON.stringify(
              e,
              ["sender", "recipient", "message_type", "height", "index", "tx_hash", "fee", "sequence"],
              4,
            ),
          ))
        );
      }
      function j(e) {
        let { openModal: t } = e,
          a = "";
        return (0, n.jsx)(n.Fragment, {
          children: (0, n.jsx)(r.Z, {
            sticky: "top",
            "data-bs-theme": "light",
            className: "nav-bar",
            children: (0, n.jsxs)(s.Z, {
              children: [
                (0, n.jsxs)(r.Z.Brand, {
                  className: "nav-bar-brand",
                  children: [
                    (0, n.jsx)("span", { className: "nav-bar-brand-highlight", children: "scan" }),
                    "opy explorer",
                  ],
                }),
                (0, n.jsx)("div", {
                  className: "nav-bar-center",
                  children: (0, n.jsx)(i.Z, {
                    onSubmit: () => {
                      var e;
                      return t(isNaN((e = a)) || isNaN(parseFloat(e)) ? e : Number(e), 0);
                    },
                    children: (0, n.jsx)(i.Z.Control, {
                      type: "search",
                      className: "main-input nav-bar-search me-2",
                      placeholder: "search by address, hash, or height",
                      onChange: (e) => {
                        a = e.target.value;
                      },
                    }),
                  }),
                }),
                (0, n.jsx)("a", {
                  href: "https://discord.gg/pNcSJj7Wdh",
                  children: (0, n.jsx)("div", {
                    id: "nav-social-icon1",
                    className: "nav-social-icon justify-content-end",
                  }),
                }),
                (0, n.jsx)("a", {
                  href: "https://x.com/CNPYNetwork",
                  children: (0, n.jsx)("div", {
                    id: "nav-social-icon2",
                    className: "nav-social-icon justify-content-end",
                  }),
                }),
              ],
            }),
          }),
        });
      }
      Date.prototype.addMS = function (e) {
        return this.setTime(this.getTime() + e), this;
      };
      var k = a(314);
      let N = [
        { src: "./block.png", label: "Blocks" },
        { src: "./transaction.png", label: "Transactions" },
        { src: "./pending.png", label: "Pending" },
        { src: "./account.png", label: "Accounts" },
        { src: "./validator.png", label: "Validators" },
        { src: "./gov.png", label: "Governance" },
        { src: "./swaps.png", label: "Swaps" },
        { src: "./supply.png", label: "Supply" },
      ];
      function _(e) {
        let { selectTable: t } = e;
        return (0, n.jsxs)(k.Eq, {
          className: "d-lg-block sidebar",
          children: [
            (0, n.jsxs)("div", {
              className: "sidebar-list",
              children: [
                N.map((e, a) =>
                  g(
                    (0, n.jsx)("div", {
                      onClick: () => t(a, 0),
                      className: "sidebar-icon-container",
                      children: (0, n.jsx)("div", {
                        className: "sidebar-icon",
                        style: { backgroundImage: "url(".concat(e.src, ")") },
                      }),
                    }),
                    e.label,
                    a,
                  ),
                ),
                (0, n.jsx)("div", {
                  className: "sidebar-icon-container",
                  children: (0, n.jsx)("a", {
                    href: "https://canopy-network.gitbook.io/docs",
                    children: g(
                      (0, n.jsx)("div", {
                        className: "sidebar-icon",
                        style: { backgroundImage: "url(./explore.png)" },
                      }),
                      "Explore",
                    ),
                  }),
                }),
              ],
            }),
            (0, n.jsx)("a", {
              href: "https://github.com/canopy-network",
              children: (0, n.jsx)("div", { id: "sidebar-social", style: { backgroundImage: "url(./github.png)" } }),
            }),
          ],
        });
      }
      var w = a(4568),
        S = a(5377);
      function C(e) {
        let t = Object.assign({}, e);
        return delete t.transaction, v(t);
      }
      function O(e) {
        let {
          last_quorum_certificate: t,
          next_validator_root: a,
          state_root: n,
          transaction_root: s,
          validator_root: r,
          last_block_hash: i,
          network_id: l,
          total_vdf_iterations: c,
          vdf: o,
          ...d
        } = x(e.block_header);
        return (
          (d.num_txs = "num_txs" in e.block_header ? e.block_header.num_txs : "0"),
          (d.total_txs = "total_txs" in e.block_header ? e.block_header.total_txs : "0"),
          JSON.parse(JSON.stringify(d, ["height", "hash", "time", "num_txs", "total_txs", "proposer_address"], 4))
        );
      }
      function Z(e) {
        var t;
        let a = e.RequestedAmount / e.AmountForSale;
        return {
          Id: null !== (t = e.Id) && void 0 !== t ? t : 0,
          Chain: e.Committee,
          AmountForSale: e.AmountForSale,
          Rate: a.toFixed(2),
          RequestedAmount: e.RequestedAmount,
          SellerReceiveAddress: e.SellerReceiveAddress,
          SellersSendAddress: e.SellersSendAddress,
          BuyerSendAddress: e.BuyerSendAddress,
          Status: "BuyerReceiveAddress" in e ? "Reserved" : "Open",
          BuyerReceiveAddress: e.BuyerReceiveAddress,
          BuyerChainDeadline: e.BuyerChainDeadline,
        };
      }
      function q(e) {
        var t, a;
        let n = [{ Results: "null" }];
        if ("Consensus" in e)
          return (function (e) {
            if (!e.Consensus) return ["0"];
            let t = x(e);
            return ["Consensus", "Validator", "Fee", "Governance"].flatMap((e) =>
              Object.entries(t[e] || {}).map((t) => {
                let [a, n] = t;
                return { ParamName: a, ParamValue: n, ParamSpace: e };
              }),
            );
          })(e);
        if ("committee_staked" in e)
          return e.committee_staked.map((t) =>
            (function (e, t) {
              let a = (e.amount / t) * 100;
              return { Chain: 1, stake_cut: "".concat(a, "%"), total_restake: e.amount };
            })(t, e.staked),
          );
        if (!e.hasOwnProperty("type"))
          return (
            (null === (a = e[0]) || void 0 === a
              ? void 0
              : null === (t = a.orders) || void 0 === t
                ? void 0
                : t.map(Z)) || n
          );
        if (null === e.results) return n;
        let s = e.results.map(
          {
            "tx-results-page": C,
            "pending-results-page": C,
            "block-results-page": O,
            accounts: (e) => e,
            validators: (e) => e,
          }[e.type] || (() => []),
        );
        return 0 === s.length ? n : s;
      }
      function T(e) {
        var t, a;
        let { filterText: s, sortColumn: r, sortDirection: i, category: l, committee: c, tableData: o } = e.state,
          d =
            ((t = q(o)),
            (a = s
              ? t.filter((e) =>
                  Object.values(e).some((e) =>
                    null == e ? void 0 : e.toString().toLowerCase().includes(s.toLowerCase()),
                  ),
                )
              : t),
            r
              ? [...a].sort((e, t) => {
                  let a = e[r],
                    n = t[r];
                  return a < n ? ("asc" === i ? -1 : 1) : a > n ? ("asc" === i ? 1 : -1) : 0;
                })
              : a);
        return (0, n.jsxs)("div", {
          className: "data-table",
          children: [
            (0, n.jsxs)("div", {
              className: "data-table-content",
              children: [
                6 === l &&
                  (0, n.jsx)("input", {
                    type: "number",
                    value: c,
                    min: "1",
                    onChange: (t) => t.target.value && e.selectTable(6, 0, Number(t.target.value)),
                    className: "chain-table mb-3",
                  }),
                (0, n.jsx)("input", {
                  type: "text",
                  value: s,
                  onChange: (t) => e.setState({ ...e.state, filterText: t.target.value }),
                  className: "search-table mb-3",
                }),
                (0, n.jsx)("h5", {
                  className: "data-table-head",
                  children:
                    "tx-results-page" === o.type
                      ? "Transactions"
                      : "pending-results-page" === o.type
                        ? "Pending"
                        : "block-results-page" === o.type
                          ? "Blocks"
                          : "accounts" === o.type
                            ? "Accounts"
                            : "validators" === o.type
                              ? "Validators"
                              : "Consensus" in o
                                ? "Governance"
                                : "committee_staked" in o
                                  ? "Committees"
                                  : "Sell Orders",
                }),
              ],
            }),
            (0, n.jsxs)(w.Z, {
              responsive: !0,
              bordered: !0,
              hover: !0,
              size: "sm",
              className: "table",
              children: [
                (0, n.jsx)("thead", {
                  children: (0, n.jsx)("tr", {
                    children: Object.keys(q(o)[0]).map((t, a) =>
                      (0, n.jsxs)(
                        "th",
                        {
                          className: "table-head",
                          onClick: () => {
                            let a = r === t && "asc" === i ? "desc" : "asc";
                            e.setState({ ...e.state, sortColumn: t, sortDirection: a });
                          },
                          style: { cursor: "pointer" },
                          children: [f(t), r === t && ("asc" === i ? " ↑" : " ↓")],
                        },
                        a,
                      ),
                    ),
                  }),
                }),
                (0, n.jsx)("tbody", {
                  children: d.map((t, a) =>
                    (0, n.jsx)(
                      "tr",
                      {
                        children: Object.keys(t).map((a, s) =>
                          (0, n.jsx)(
                            "td",
                            {
                              className: "table-col",
                              children: (function (e, t, a) {
                                if ("public_key" === e) return (0, n.jsx)(S.Z, { text: t });
                                if ((!b(t) && /[0-9A-Fa-f]{6}/g.test(t)) || "height" === e) {
                                  let e = b(t) ? t : (0, n.jsx)(S.Z, { text: t });
                                  return (0, n.jsx)("a", {
                                    href: "#",
                                    onClick: () => a(t),
                                    style: { cursor: "pointer" },
                                    children: e,
                                  });
                                }
                                return e.includes("time") ? m(t) : b(t) ? u(t) : p(e, t);
                              })(a, t[a], e.openModal),
                            },
                            s,
                          ),
                        ),
                      },
                      a,
                    ),
                  ),
                }),
              ],
            }),
            y(o, (t) => e.selectTable(e.state.category, t)),
          ],
        });
      }
      var A = a(1672),
        B = a(8748),
        P = a(2280),
        D = a(6529),
        F = a(8695),
        M = a(4462),
        I = a(5401),
        R = a(928),
        E = a(8041);
      let L = "http://127.0.0.1:50002";
      window.__CONFIG__
        ? ((L = window.__CONFIG__.rpcURL), window.__CONFIG__.baseChainRPCURL, Number(window.__CONFIG__.chainId))
        : console.log("config undefined");
      let J = "/v1/query/validators";
      async function G(e, t) {
        let a = await fetch(L + t, { method: "POST", body: e }).catch((e) => {
          console.log(e);
        });
        return null == a ? {} : a.json();
      }
      function V(e) {
        return JSON.stringify({ height: e });
      }
      function z(e) {
        return JSON.stringify({ hash: e });
      }
      function H(e, t) {
        return JSON.stringify({ pageNumber: e, perPage: 10, address: t });
      }
      function K(e, t) {
        return JSON.stringify({ height: e, address: t });
      }
      function U(e, t) {
        return JSON.stringify({ height: e, id: t });
      }
      function W(e, t) {
        return JSON.stringify({ pageNumber: e, perPage: 10, height: t });
      }
      function X(e, t) {
        return G(W(e, 0), "/v1/query/blocks");
      }
      async function Y(e, t, a) {
        let n = {};
        return (
          (n.account = await G(K(e, t), "/v1/query/account")),
          (n.sent_transactions = await G(H(a, t), "/v1/query/txs-by-sender")),
          (n.rec_transactions = await G(H(a, t), "/v1/query/txs-by-rec")),
          n
        );
      }
      function Q(e, t) {
        return G(V(e), "/v1/query/params");
      }
      function $(e, t) {
        return G(V(e), "/v1/query/supply");
      }
      async function ee(e, t) {
        var a, n;
        let s = "no result found";
        if ("string" == typeof e) {
          if (64 === e.length) {
            let t = await G(z(e), "/v1/query/block-by-hash");
            if (null == t ? void 0 : null === (n = t.block_header) || void 0 === n ? void 0 : n.hash)
              return { block: t };
            let a = await G(z(e), "/v1/query/tx-by-hash");
            return (null == a ? void 0 : a.sender) ? a : s;
          }
          if (40 === e.length) {
            let [a, n] = await Promise.all([G(K(0, e), "/v1/query/validator"), Y(0, e, t)]);
            return n.account.address || a.address ? (n.account.address ? { ...n, validator: a } : { validator: a }) : s;
          }
          return s;
        }
        let r = await G(V(e), "/v1/query/block-by-height");
        return (null == r ? void 0 : null === (a = r.block_header) || void 0 === a ? void 0 : a.hash)
          ? { block: r }
          : s;
      }
      async function et() {
        let e = {};
        return (
          (e.blocks = await X(1, 0)),
          (e.canopyCommittee = await G(JSON.stringify({ height: 0, pageNumber: 1, perPage: 10, committee: 1 }), J)),
          (e.supply = await $(0, 0)),
          (e.pool = await G(U(0, 4294967296), "/v1/query/pool")),
          (e.params = await Q(0, 0)),
          e
        );
      }
      async function ea(e, t, a) {
        switch (t) {
          case 0:
            return await X(e, 0);
          case 1:
            return await G(W(e, 0), "/v1/query/txs-by-height");
          case 2:
            return await G(H(e, ""), "/v1/query/pending");
          case 3:
            return await G(W(e, 0), "/v1/query/accounts");
          case 4:
            return await G(W(e, 0), J);
          case 5:
            return await Q(e, 0);
          case 6:
            return await G(U(0, a), "/v1/query/orders");
          case 7:
            return await $(0);
        }
      }
      function en(e) {
        return null == e || 0 === e
          ? [0]
          : "block" in e
            ? er(e) || { None: "" }
            : "transaction" in e
              ? { ...e, transaction: void 0 }
              : e;
      }
      function es(e) {
        for (let t = 0; t < e.length; t++) e[t] = v(e[t]);
        return e;
      }
      function er(e) {
        let {
          last_quorum_certificate: t,
          next_validator_root: a,
          state_root: n,
          transaction_root: s,
          validator_root: r,
          last_block_hash: i,
          network_id: l,
          vdf: c,
          ...o
        } = e.block.block_header;
        return o;
      }
      function ei(e, t, a) {
        if ("block" in t)
          switch (a) {
            case 0:
              return er(t);
            case 1:
              return t.block.transactions ? es(t.block.transactions) : 0;
            default:
              return t.block;
          }
        else if ("transaction" in t)
          switch (a) {
            case 0:
              if ("qc" in t.transaction.msg) {
                var n;
                return {
                  certificate_height: (n = t.transaction.msg.qc).header.height,
                  network_id: n.header.networkID,
                  committee_id: n.header.committeeID,
                  block_hash: n.blockHash,
                  results_hash: n.resultsHash,
                };
              }
              return t.transaction.msg;
            case 1:
              return { hash: t.tx_hash, time: t.transaction.time, sender: t.sender, type: t.message_type };
            default:
              return t;
          }
        else if ("validator" in t && !e.modalState.accOnly) return x(t.validator);
        else if ("account" in t)
          switch (a) {
            case 0:
              return x(t.account);
            case 1:
              return es(t.sent_transactions.results);
            default:
              return es(t.rec_transactions.results);
          }
      }
      function el(e) {
        let { state: t, setState: a } = e,
          s = t.modalState.data,
          r = (function (e, t) {
            if (!t) return { None: "" };
            let a = x(t);
            return a.transaction
              ? (delete a.transaction, a)
              : a.block
                ? er(a)
                : a.validator && !e.modalState.accOnly
                  ? a.validator
                  : a.account;
          })(t, s);
        if (0 === Object.keys(s).length) return (0, n.jsx)(n.Fragment, {});
        if ("no result found" === s)
          return (0, n.jsx)(B.Z, {
            position: "top-center",
            className: "search-toast",
            children: (0, n.jsxs)(P.Z, {
              onClose: i,
              show: !0,
              delay: 3e3,
              autohide: !0,
              children: [
                (0, n.jsx)(P.Z.Header, {}),
                (0, n.jsx)(P.Z.Body, { className: "search-toast-body", children: "no results found" }),
              ],
            }),
          });
        function i() {
          a({ ...t, modalState: { show: !1, query: "", page: 0, data: {}, accOnly: !1 } });
        }
        function l(e) {
          let a = ei(t, s, e);
          return (0, n.jsx)(w.Z, {
            responsive: !0,
            children: (0, n.jsx)("tbody", {
              children: Object.keys(a).map((e, t) =>
                (0, n.jsxs)(
                  "tr",
                  {
                    children: [
                      (0, n.jsx)("td", { className: "detail-table-title", children: f(e) }),
                      (0, n.jsx)("td", { className: "detail-table-info", children: p(e, a[e]) }),
                    ],
                  },
                  t,
                ),
              ),
            }),
          });
        }
        function c(e) {
          let r = 0,
            i = 10,
            l = [0],
            c = s,
            o = t.modalState,
            d = c.block;
          return (
            "block" in c
              ? ((r = (i = 0 === o.page || 1 === o.page ? 10 : 10 * o.page) - 10),
                (l = d.transactions || l),
                (c = { pageNumber: o.Page, perPage: 10, totalPages: Math.ceil(d.block_header.num_txs / 10), ...c }))
              : "account" in c &&
                ((l = 1 === e ? es(c.sent_transactions.results) : es(c.rec_transactions.results)),
                (c = 1 === e ? c.sent_transactions : c.rec_transactions)),
            (0, n.jsxs)(n.Fragment, {
              children: [
                (0, n.jsx)(w.Z, {
                  responsive: !0,
                  children: (0, n.jsxs)("tbody", {
                    children: [
                      (0, n.jsx)("tr", {
                        children: Object.keys(en(ei(t, s, 1)[0])).map((e, t) =>
                          (0, n.jsx)("td", { className: "detail-table-row-title", children: f(e) }, t),
                        ),
                      }),
                      l
                        .slice(r, i)
                        .map((e, t) =>
                          (0, n.jsx)(
                            "tr",
                            {
                              children: Object.keys(en(e)).map((t, a) =>
                                (0, n.jsx)("td", { className: "detail-table-row-info", children: p(t, e[t]) }, a),
                              ),
                            },
                            t,
                          ),
                        ),
                    ],
                  }),
                }),
                y(c, (e) =>
                  ee(o.query, e).then((n) => {
                    a({ ...t, modalState: { ...o, show: !0, query: o.query, page: e, data: n } });
                  }),
                ),
              ],
            })
          );
        }
        function o() {
          return (0, n.jsx)(A.he, { rootName: "result", defaultInspectDepth: 1, value: ei(t, s, 2) });
        }
        return (0, n.jsxs)(F.Z, {
          size: "xl",
          show: t.modalState.show,
          onHide: i,
          children: [
            (0, n.jsx)(F.Z.Header, { closeButton: !0 }),
            (0, n.jsxs)(F.Z.Body, {
              className: "modal-body",
              children: [
                (0, n.jsxs)("h3", {
                  className: "modal-header",
                  children: [
                    (0, n.jsx)("div", { className: "modal-header-icon" }),
                    "transaction" in s
                      ? "Transaction"
                      : "block" in s
                        ? "Block"
                        : "validator" in s && !t.modalState.accOnly
                          ? "Validator"
                          : "Account",
                    " Details",
                  ],
                }),
                (0, n.jsx)(M.Z, {
                  className: "modal-card-group",
                  children: Object.keys(r).map((e, s) =>
                    g(
                      (0, n.jsx)(
                        I.Z,
                        {
                          onClick: () => {
                            var n;
                            return (n = r[e]), void (navigator.clipboard.writeText(n), a({ ...t, showToast: !0 }));
                          },
                          className: "modal-cards",
                          children: (0, n.jsxs)(I.Z.Body, {
                            className: "modal-card",
                            children: [
                              (0, n.jsx)("h5", { className: "modal-card-title", children: e }),
                              (0, n.jsx)("div", {
                                className: "modal-card-detail",
                                children: (0, n.jsx)(S.Z, { text: String(r[e]) }),
                              }),
                              (0, n.jsx)("img", { className: "copy-img", src: "./copy.png", alt: "copy" }),
                            ],
                          }),
                        },
                        s,
                      ),
                      r[e],
                      s,
                      "top",
                    ),
                  ),
                }),
                (0, n.jsx)(R.Z, {
                  defaultActiveKey: "0",
                  id: "fill-tab-example",
                  className: "mb-3",
                  fill: !0,
                  children: [void 0, void 0, void 0].map((e, r) =>
                    (0, n.jsx)(
                      E.Z,
                      {
                        tabClassName: "rb-tab",
                        eventKey: r,
                        title:
                          "transaction" in s
                            ? 0 === r
                              ? "Message"
                              : 1 === r
                                ? "Meta"
                                : "Raw"
                            : "block" in s
                              ? 0 === r
                                ? "Header"
                                : 1 === r
                                  ? "Transactions"
                                  : "Raw"
                              : "validator" in s && !t.modalState.accOnly
                                ? 0 === r
                                  ? "Validator"
                                  : 1 === r
                                    ? "Account"
                                    : "Raw"
                                : 0 === r
                                  ? "Account"
                                  : 1 === r
                                    ? "Sent Transactions"
                                    : "Received Transactions",
                        children:
                          "block" in s
                            ? 0 === r
                              ? l(r)
                              : 1 === r
                                ? c(r)
                                : o()
                            : "transaction" in s
                              ? 0 === r
                                ? l(r)
                                : 1 === r
                                  ? l(r)
                                  : o()
                              : "validator" in s && !t.modalState.accOnly
                                ? 0 === r
                                  ? l(r)
                                  : 1 === r
                                    ? (0, n.jsx)(D.Z, {
                                        className: "open-acc-details-btn",
                                        variant: "outline-secondary",
                                        onClick: () => a({ ...t, modalState: { ...t.modalState, accOnly: !0 } }),
                                        children: "Open Account Details",
                                      })
                                    : o()
                                : 0 === r
                                  ? l(r)
                                  : c(r),
                      },
                      r,
                    ),
                  ),
                }),
              ],
            }),
          ],
        });
      }
      var ec = a(8888),
        eo = a(641);
      let ed = ["./block-filled.png", "./chart-up.png", "./transaction-filled.png", "./lock-filled.png"],
        eu = ["Latest Block", "Supply", "Transactions", "Validators"];
      function eh(e) {
        let t = e.state.cardData;
        return (0, n.jsx)(ec.Z, {
          sm: 1,
          md: 2,
          lg: 4,
          className: "g-4",
          children: Array.from({ length: 4 }, (a, s) =>
            (0, n.jsx)(
              eo.Z,
              {
                children: (0, n.jsx)(I.Z, {
                  className: "text-center",
                  children: (0, n.jsxs)(I.Z.Body, {
                    className: "card-body",
                    onClick:
                      0 === s
                        ? () => e.openModal(0)
                        : 1 === s
                          ? () => e.selectTable(7, 0)
                          : 2 === s
                            ? () => e.selectTable(1, 0)
                            : () => e.selectTable(s + 1, 0),
                    children: [
                      (0, n.jsx)("div", { className: "card-image", style: { backgroundImage: "url(" + ed[s] + ")" } }),
                      (0, n.jsx)(I.Z.Title, { className: "card-title", children: eu[s] }),
                      (0, n.jsx)("h5", {
                        children: (function (e, t) {
                          let a = e.blocks;
                          if (0 === a.results.length) return "Loading";
                          switch (t) {
                            case 0:
                              return h(a.results[0].block_header.height);
                            case 1:
                              return h(e.supply.total);
                            case 2:
                              if (null == a.results[0].block_header.num_txs) return "+0";
                              return "+" + h(a.results[0].block_header.num_txs);
                            case 3:
                              let s = 0;
                              if (!e.canopyCommittee.results) return 0;
                              return (
                                e.canopyCommittee.results.forEach(function (e) {
                                  s += Number(e.staked_amount);
                                }),
                                (0, n.jsxs)(n.Fragment, {
                                  children: [
                                    h(s),
                                    (0, n.jsx)("span", { style: { fontSize: "14px" }, children: " stake" }),
                                  ],
                                })
                              );
                          }
                        })(t, s),
                      }),
                      (0, n.jsxs)(I.Z.Text, {
                        children: [
                          (0, n.jsx)("span", {
                            children: (function (e, t) {
                              let a = e.blocks;
                              if (0 === a.results.length) return "Loading";
                              switch (t) {
                                case 0:
                                  return m(a.results[0].block_header.time);
                                case 1:
                                  return h(Number(e.supply.total) - Number(e.supply.staked), 1e3) + " liquid";
                                case 2:
                                  return (
                                    "blk size: " +
                                    (function (e) {
                                      let t = arguments.length > 1 && void 0 !== arguments[1] ? arguments[1] : 2;
                                      if (!+e) return "0 Bytes";
                                      let a = Math.floor(Math.log(e) / Math.log(1024));
                                      return ""
                                        .concat(parseFloat((e / Math.pow(1024, a)).toFixed(0 > t ? 0 : t)), " ")
                                        .concat(["B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"][a]);
                                    })(a.results[0].meta.size)
                                  );
                                case 3:
                                  if (!e.canopyCommittee.results) return "0 unique vals";
                                  return e.canopyCommittee.results.length + " unique vals";
                              }
                            })(t, s),
                          }),
                          (0, n.jsx)("span", {
                            className: "card-info-3",
                            children: (function (e, t) {
                              let a = e.blocks;
                              if (0 === a.results.length) return "Loading";
                              switch (t) {
                                case 0:
                                  return a.results[0].meta.took;
                                case 1:
                                  return h(e.supply.staked, 1e3) + " staked";
                                case 2:
                                  return "block #" + a.results[0].block_header.height;
                                case 3:
                                  return (
                                    "stake threshold " +
                                    h(e.params.Validator.validator_stake_percent_for_subsidized_committee, 1e3) +
                                    "%"
                                  );
                              }
                            })(t, s),
                          }),
                          (0, n.jsx)("br", {}),
                          (0, n.jsx)("span", {
                            className: "card-info-4",
                            children: (function (e, t) {
                              let a = e.blocks;
                              if (0 === a.results.length) return "Loading";
                              switch (t) {
                                case 0:
                                  return (0, n.jsx)("div", {
                                    style: { height: "25px", paddingTop: "10px" },
                                    children: (0, n.jsx)(S.Z, { text: a.results[0].block_header.hash }),
                                  });
                                case 1:
                                  return "+" + Number(50) + "/blk";
                                case 2:
                                  return "TOTAL " + h(a.results[0].block_header.total_txs);
                                case 3:
                                  if (!e.canopyCommittee.results) return "MaxStake: 0";
                                  return "MaxStake: " + h(e.canopyCommittee.results[0].staked_amount, 1e3);
                                default:
                                  return "?";
                              }
                            })(t, s),
                          }),
                        ],
                      }),
                      (0, n.jsx)(I.Z.Footer, {
                        className: "card-footer",
                        children: (function (e, t) {
                          let a = e.blocks;
                          if (0 === a.results.length) return "Loading";
                          switch (t) {
                            case 0:
                              var n, s;
                              return (
                                "Next block: " +
                                ((n = a.results[0].block_header.time),
                                (s = a.results[0].meta.took),
                                new Date(Math.floor(n / 1e3)).addMS(s).toLocaleTimeString())
                              );
                            case 1:
                              let r = "DAO pool supply: ";
                              if (null != e.pool) return r + h(e.pool.amount, 1e3);
                              return r;
                            case 2:
                              let i = 0,
                                l = a.results[0].transactions;
                              if (null == l || 0 === l.length) return "Average fee in last blk: 0";
                              return (
                                l.forEach(function (e) {
                                  i += Number(e.transaction.fee);
                                }),
                                "Average fee in last blk: " + h(i / l.length, 1e6)
                              );
                            case 3:
                              let c = 0;
                              if (!e.canopyCommittee.results) return "0% in validator set";
                              return (
                                e.canopyCommittee.results.forEach(function (e) {
                                  c += Number(e.staked_amount);
                                }),
                                ((c / e.supply.staked) * 100).toFixed(1) + "% in validator set"
                              );
                          }
                        })(t, s),
                      }),
                    ],
                  }),
                }),
              },
              s,
            ),
          ),
        });
      }
      var em = a(2448);
      function ep() {
        let [e, t] = (0, l.useState)({
          loading: !0,
          cardData: {},
          category: 0,
          tablePage: 0,
          tableData: {},
          showToast: !1,
          sortColumn: null,
          sortDirection: "asc",
          filterText: "",
          committee: 1,
          modalState: { show: !1, page: 0, query: "", data: {}, accOnly: !1 },
        });
        function a(a) {
          Promise.all([ea(e.tablePage, e.category, e.committee), et()]).then((n) =>
            a
              ? t({ ...e, loading: !1, tableData: n[0], cardData: n[1] })
              : t({ ...e, tableData: n[0], cardData: n[1] }),
          );
        }
        async function s(a) {
          t({ ...e, modalState: { show: !0, query: a, page: 0, accOnly: !1, data: await ee(a, 0) } });
        }
        async function r(a, n, s) {
          null == s && (s = e.committee),
            t({ ...e, committee: s, category: a, tablePage: n, tableData: await ea(n, a, s) });
        }
        if (
          ((0, l.useEffect)(() => {
            let e = setInterval(() => {
              a(!1);
            }, 4e3);
            return () => clearInterval(e);
          }),
          e.loading || !e.cardData.blocks)
        )
          return (
            a(!0),
            (0, n.jsxs)(n.Fragment, {
              children: [
                (0, n.jsx)(em.Z, { id: "spinner" }),
                (0, n.jsx)("br", {}),
                " ",
                (0, n.jsx)("br", {}),
                (0, n.jsx)("center", { children: (0, n.jsx)("h3", { children: "Waiting to explore!" }) }),
                (0, n.jsx)("center", { children: (0, n.jsx)("h7", { children: "... loading or no blocks yet ..." }) }),
              ],
            })
          );
        {
          let a = { state: e, setState: t, openModal: s, selectTable: r };
          return (0, n.jsx)(n.Fragment, {
            children: (0, n.jsxs)("div", {
              id: "container",
              children: [
                (0, n.jsx)(j, { ...a }),
                (0, n.jsx)(_, { ...a }),
                (0, n.jsxs)("div", {
                  id: "pageContent",
                  children: [
                    (0, n.jsx)(eh, { ...a }),
                    (0, n.jsx)(T, { ...a }),
                    (0, n.jsx)(el, { ...a }),
                    (0, n.jsx)(B.Z, {
                      id: "toast",
                      position: "bottom-end",
                      children: (0, n.jsx)(P.Z, {
                        bg: "dark",
                        onClose: () => t({ ...e, showToast: !1 }),
                        show: e.showToast,
                        delay: 2e3,
                        autohide: !0,
                        children: (0, n.jsx)(P.Z.Body, { children: "Copied!" }),
                      }),
                    }),
                  ],
                }),
              ],
            }),
          });
        }
      }
    },
  },
  function (e) {
    e.O(0, [265, 888, 774, 179], function () {
      return e((e.s = 5557));
    }),
      (_N_E = e.O());
  },
]);
