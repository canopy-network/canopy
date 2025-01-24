(self.webpackChunk_N_E = self.webpackChunk_N_E || []).push([
  [405],
  {
    5557: function (e, t, a) {
      (window.__NEXT_P = window.__NEXT_P || []).push([
        "/",
        function () {
          return a(5322);
        },
      ]);
    },
    5322: function (e, t, a) {
      "use strict";
      a.r(t),
        a.d(t, {
          default: function () {
            return ez;
          },
        });
      var n = a(5893),
        r = a(3353),
        s = a(5264),
        o = a(6159),
        i = a(5886),
        l = a(7294),
        d = a(8748),
        c = a(2280),
        u = a(9209),
        p = a(2854);
      function m(e, t, a, n) {
        let r = null,
          s = n && n.address ? n.net_address : "",
          o = !!n && !!n.address && n.delegate,
          i = !!n && !!n.address && n.compound,
          l = n && n.address ? n.output : "",
          d = null != a ? a.address : "",
          c = null != t ? t.publicKey : "",
          u = null != a ? a.address : "",
          p = n && n.address ? n.committees.join(",") : "";
        (d = "send" !== e && n && n.address ? n.address : d),
          (d = "stake" === e && n && n.address ? "WARNING: validator already staked" : d),
          ("edit-stake" === e || "stake" === e) && (r = n && n.address ? n.staked_amount : null);
        let m = {
          privateKey: {
            placeholder: "opt: private key hex to import",
            defaultValue: "",
            tooltip: "the raw private key to import if blank - will generate a new key",
            label: "private_key",
            inputText: "key",
            feedback: "please choose a private key to import",
            required: !1,
            type: "password",
            minLength: 64,
            maxLength: 128,
          },
          address: {
            placeholder: "the unique id of the account",
            defaultValue: d,
            tooltip: "the short public key id of the account",
            label: "sender",
            inputText: "address",
            feedback: "please choose an address to send the transaction from",
            required: !0,
            type: "text",
            minLength: 40,
            maxLength: 40,
          },
          pubKey: {
            placeholder: "public key of the node",
            defaultValue: c,
            tooltip: "the public key of the validator",
            label: "pubKey",
            inputText: "pubKey",
            feedback: "please choose a pubKey to send the transaction from",
            required: !0,
            type: "text",
            minLength: 96,
            maxLength: 96,
          },
          committees: {
            placeholder: "1, 22, 50",
            defaultValue: p,
            tooltip: "comma separated list of committee chain IDs to stake for",
            label: "committees",
            inputText: "committees",
            feedback: "please input atleast 1 committee",
            required: !0,
            type: "text",
            minLength: 1,
            maxLength: 200,
          },
          netAddr: {
            placeholder: "url of the node",
            defaultValue: s,
            tooltip: "the url of the validator for consensus and polling",
            label: "net_address",
            inputText: "net-addr",
            feedback: "please choose a net address for the validator",
            required: !0,
            type: "text",
            minLength: 5,
            maxLength: 50,
          },
          earlyWithdrawal: {
            placeholder: "early withdrawal rewards for 20% penalty",
            defaultValue: !i,
            tooltip: "validator NOT reinvesting their rewards to their stake, incurring a 20% penalty",
            label: "earlyWithdrawal",
            inputText: "withdrawal",
            feedback: "please choose if your validator to earlyWithdrawal or not",
            required: !0,
            type: "text",
            minLength: 4,
            maxLength: 5,
          },
          delegate: {
            placeholder: "validator delegation status",
            defaultValue: o,
            tooltip:
              "validator is passively delegating rather than actively validating. NOTE: THIS FIELD IS FIXED AND CANNOT BE UPDATED WITH EDIT-STAKE",
            label: "delegate",
            inputText: "delegate",
            feedback: "please choose if your validator is delegating or not",
            required: !0,
            type: "text",
            minLength: 4,
            maxLength: 5,
          },
          rec: {
            placeholder: "recipient of the tx",
            defaultValue: "",
            tooltip: "the recipient of the transaction",
            label: "recipient",
            inputText: "recipient",
            feedback: "please choose a recipient for the transaction",
            required: !0,
            type: "text",
            minLength: 40,
            maxLength: 40,
          },
          amount: {
            placeholder: "amount value for the tx",
            defaultValue: r,
            tooltip: "the amount of currency being sent / sold",
            label: "amount",
            inputText: "amount",
            feedback: "please choose an amount for the tx",
            required: !0,
            type: "number",
            minLength: 1,
            maxLength: 100,
          },
          receiveAmount: {
            placeholder: "amount of counter asset to receive",
            defaultValue: r,
            tooltip: "the amount of counter asset being received",
            label: "receiveAmount",
            inputText: "rec-amount",
            feedback: "please choose a receive amount for the tx",
            required: !0,
            type: "number",
            minLength: 1,
            maxLength: 100,
          },
          orderId: {
            placeholder: "the id of the existing order",
            tooltip: "the unique identifier of the order",
            label: "orderId",
            inputText: "order-id",
            feedback: "please input an order id",
            required: !0,
            type: "number",
            minLength: 1,
            maxLength: 100,
          },
          committeeId: {
            placeholder: "the id of the committee / counter asset",
            tooltip: "the unique identifier of the committee / counter asset",
            label: "committeeId",
            inputText: "commit-Id",
            feedback: "please input a committeeId id",
            required: !0,
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
            required: !0,
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
            required: !0,
            type: "text",
            minLength: 40,
            maxLength: 40,
          },
          output: {
            placeholder: "output of the node",
            defaultValue: l,
            tooltip: "the non-custodial address where rewards and stake is directed to",
            label: "output",
            inputText: "output",
            feedback: "please choose an output address for the validator",
            required: !0,
            type: "text",
            minLength: 40,
            maxLength: 40,
          },
          signer: {
            placeholder: "signer of the transaction",
            defaultValue: u,
            tooltip: "the signing address that authorizes the transaction",
            label: "signer",
            inputText: "signer",
            feedback: "please choose a signer address",
            required: !0,
            type: "text",
            minLength: 40,
            maxLength: 40,
          },
          paramSpace: {
            placeholder: "",
            defaultValue: "",
            tooltip: "the category 'space' of the parameter",
            label: "param_space",
            inputText: "param space",
            feedback: "please choose a space for the parameter change",
            required: !0,
            type: "select",
            minLength: 1,
            maxLength: 100,
          },
          paramKey: {
            placeholder: "",
            defaultValue: "",
            tooltip: "the identifier of the parameter",
            label: "param_key",
            inputText: "param key",
            feedback: "please choose a key for the parameter change",
            required: !0,
            type: "select",
            minLength: 1,
            maxLength: 100,
          },
          paramValue: {
            placeholder: "",
            defaultValue: "",
            tooltip: "the newly proposed value of the parameter",
            label: "param_value",
            inputText: "param val",
            feedback: "please choose a value for the parameter change",
            required: !0,
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
            required: !0,
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
            required: !0,
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
            required: !1,
            minLength: 0,
            maxLength: 200,
          },
          fee: {
            placeholder: "opt: transaction fee",
            defaultValue: "",
            tooltip: " a small amount of CNPY deducted from the account to process any transaction blank = default fee",
            label: "fee",
            inputText: "txn-fee",
            feedback: "please choose a valid number",
            required: !1,
            type: "number",
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
            required: !0,
            type: "password",
            minLength: 0,
            maxLength: 40,
          },
        };
        switch (e) {
          case "send":
            return [m.address, m.rec, m.amount, m.memo, m.fee, m.password];
          case "stake":
            return [
              m.address,
              m.pubKey,
              m.committees,
              m.netAddr,
              m.amount,
              m.delegate,
              m.earlyWithdrawal,
              m.output,
              m.signer,
              m.memo,
              m.fee,
              m.password,
            ];
          case "create_order":
            return [m.address, m.committeeId, m.amount, m.receiveAmount, m.receiveAddress, m.memo, m.fee, m.password];
          case "buy_order":
            return [m.address, m.buyersReceiveAddress, m.orderId, m.fee, m.password];
          case "edit_order":
            return [
              m.address,
              m.committeeId,
              m.orderId,
              m.amount,
              m.receiveAmount,
              m.receiveAddress,
              m.memo,
              m.fee,
              m.password,
            ];
          case "delete_order":
            return [m.address, m.committeeId, m.orderId, m.memo, m.fee, m.password];
          case "edit-stake":
            return [
              m.address,
              m.committees,
              m.netAddr,
              m.amount,
              m.earlyWithdrawal,
              m.output,
              m.signer,
              m.memo,
              m.fee,
              m.password,
            ];
          case "change-param":
            return [
              m.address,
              m.paramSpace,
              m.paramKey,
              m.paramValue,
              m.startBlock,
              m.endBlock,
              m.memo,
              m.fee,
              m.password,
            ];
          case "dao-transfer":
            return [m.address, m.amount, m.startBlock, m.endBlock, m.memo, m.fee, m.password];
          case "pause":
          case "unpause":
          case "unstake":
            return [m.address, m.signer, m.memo, m.fee, m.password];
          case "pass-and-addr":
            return [m.address, m.password];
          case "pass-and-pk":
            return [m.privateKey, m.password];
          case "pass-only":
            return [m.password];
          default:
            return [m.address, m.memo, m.fee, m.password];
        }
      }
      let h = {
        poll: {
          "PLACEHOLDER EXAMPLE": {
            proposalHash: "PLACEHOLDER EXAMPLE",
            proposalURL: "https://discord.com/channels/1310733928436600912/1323330593701761204",
            accounts: { approvedPercent: 38, rejectPercent: 62, votedPercent: 35 },
            validators: { approvedPercent: 76, rejectPercent: 24, votedPercent: 77 },
          },
        },
        pollJSON: { proposal: "canopy network is the best", endBlock: 100, URL: "https://discord.com/link-to-thread" },
        proposals: {
          "2cbb73b8abdacf233f4c9b081991f1692145624a95004f496a95d3cce4d492a4": {
            proposal: {
              parameter_space: "cons|fee|val|gov",
              parameter_key: "protocol_version",
              parameter_value: "example",
              start_height: 1,
              end_height: 1e6,
              signer: "4646464646464646464646464646464646464646464646464646464646464646",
            },
            approve: !1,
          },
        },
        params: {
          parameter_space: "consensus",
          parameter_key: "protocol_version",
          parameter_value: "1/150",
          start_height: 1,
          end_height: 100,
          signer: "303739303732333263...",
        },
        rawTx: {
          type: "change_parameter",
          msg: {
            parameter_space: "cons",
            parameter_key: "block_size",
            parameter_value: 1e3,
            start_height: 1,
            end_height: 100,
            signer: "1fe1e32edc41d688...",
          },
          signature: { public_key: "a88b9c0c7b77e7f8ac...", signature: "8f6d016d04e350..." },
          memo: "",
          fee: 1e4,
        },
      };
      function x(e) {
        return e.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
      }
      function f(e) {
        let t = !(arguments.length > 1) || void 0 === arguments[1] || arguments[1],
          a = arguments.length > 2 && void 0 !== arguments[2] ? arguments[2] : 1e15;
        return null == e
          ? "zero"
          : (t && (e /= 1e6), Number(e) < a)
            ? x(e)
            : Intl.NumberFormat("en", { notation: "compact", maximumSignificantDigits: 8 }).format(e);
      }
      function y(e, t, a) {
        let n = arguments.length > 3 && void 0 !== arguments[3] ? arguments[3] : "Copied!";
        navigator.clipboard.writeText(a), t({ ...e, toast: n });
      }
      function g(e, t) {
        return (0, n.jsx)(d.Z, {
          id: "toast",
          position: "bottom-end",
          children: (0, n.jsx)(c.Z, {
            bg: "dark",
            onClose: () => t({ ...e, toast: "" }),
            show: "" != e.toast,
            delay: 2e3,
            autohide: !0,
            children: (0, n.jsx)(c.Z.Body, { children: e.toast }),
          }),
        });
      }
      function b(e, t, a) {
        t.preventDefault();
        let n = {};
        for (let e = 0; t.target[e] && t.target[e].ariaLabel; e++) n[t.target[e].ariaLabel] = t.target[e].value;
        a(n);
      }
      function v(e, t, a) {
        let r = arguments.length > 3 && void 0 !== arguments[3] ? arguments[3] : "right";
        return (0, n.jsx)(
          u.Z,
          {
            placement: r,
            delay: { show: 250, hide: 400 },
            overlay: (0, n.jsx)(p.Z, { id: "button-tooltip", children: t }),
            children: e,
          },
          a,
        );
      }
      function j(e) {
        return !e || 0 === Object.keys(e).length;
      }
      let k = ["	", '"'],
        w = (e) => {
          let t = e.target.value;
          k.forEach((e) => {
            t = t.split(e).join("");
          }),
            (e.target.value = t);
        },
        N = (e) => {
          let t = e.target.value.replace(/,/g, "");
          /^\d*$/.test(t) && "0" !== t[0] && (e.target.value = x(t));
        },
        O = (e) => Number(parseInt(e.replace(/,/g, ""), 10)),
        T = [
          { icon: "./account.png", tip: "accounts" },
          { icon: "./gov.png", tip: "governance" },
          { icon: "./dashboard.png", tip: "monitor" },
        ],
        S = [
          { url: "https://discord.gg/pNcSJj7Wdh", icon: "./discord-filled.png" },
          { url: "https://x.com/CNPYNetwork", icon: "./twitter.png" },
        ];
      function I(e) {
        let { keystore: t, setActiveKey: a, keyIdx: l, setNavIdx: d } = e;
        return (0, n.jsx)(s.Z, {
          sticky: "top",
          "data-bs-theme": "light",
          id: "nav-bar",
          children: (0, n.jsxs)(r.Z, {
            id: "nav-bar-container",
            children: [
              (0, n.jsxs)(s.Z.Brand, {
                id: "nav-bar-brand",
                children: ["my ", (0, n.jsx)("span", { id: "nav-bar-brand-highlight", children: "canopy " }), "wallet"],
              }),
              (0, n.jsx)("div", {
                id: "nav-dropdown-container",
                children: (0, n.jsx)(o.Z, {
                  id: "nav-dropdown",
                  title: (0, n.jsxs)(n.Fragment, {
                    children: [
                      Object.keys(t)[l],
                      (0, n.jsx)("img", { alt: "key", id: "dropdown-image", src: "./key.png" }),
                    ],
                  }),
                  children: Object.keys(t).map((e, t) =>
                    (0, n.jsx)(o.Z.Item, { onClick: () => a(t), className: "nav-dropdown-item", children: e }, t),
                  ),
                }),
              }),
              (0, n.jsx)("div", {
                id: "nav-bar-select-container",
                children: (0, n.jsx)(i.Z, {
                  id: "nav-bar-select",
                  justify: !0,
                  variant: "tabs",
                  defaultActiveKey: "0",
                  children: T.map((e, t) =>
                    (0, n.jsx)(
                      i.Z.Item,
                      {
                        onClick: () => d(t),
                        children: v(
                          (0, n.jsx)(i.Z.Link, {
                            eventKey: t.toString(),
                            children: (0, n.jsx)("img", { className: "navbar-image-link", src: e.icon, alt: e.tip }),
                          }),
                          e.tip,
                          t,
                          "bottom",
                        ),
                      },
                      t,
                    ),
                  ),
                }),
              }),
              (0, n.jsx)("a", {
                href: S[0].url,
                children: (0, n.jsx)("div", {
                  id: "nav-social-icon-discord",
                  style: { backgroundImage: "url(" + S[0].icon + ")" },
                  className: "nav-social-icon",
                }),
              }),
              (0, n.jsx)("a", {
                href: S[1].url,
                children: (0, n.jsx)("div", {
                  style: { backgroundImage: "url(" + S[1].icon + ")" },
                  className: "nav-social-icon",
                }),
              }),
            ],
          }),
        });
      }
      let P = "http://127.0.0.1:50002",
        L = "http://127.0.0.1:50003";
      window.__CONFIG__
        ? ((P = window.__CONFIG__.rpcURL),
          (L = window.__CONFIG__.adminRPCURL),
          window.__CONFIG__.baseChainRPCURL,
          Number(window.__CONFIG__.chainId))
        : console.log("config undefined");
      let _ = "/v1/admin/log",
        E = "/v1/admin/consensus-info?id=1",
        A = "/v1/admin/peer-info";
      async function C(e, t) {
        let a = await fetch(e + t, { method: "GET" }).catch((e) => {
          console.log(e);
        });
        return null == a ? {} : a.json();
      }
      async function Z(e, t) {
        let a = await fetch(e + t, { method: "GET" }).catch((e) => {
          console.log(e);
        });
        return null == a ? {} : a.text();
      }
      async function R(e, t, a) {
        let n = await fetch(e + t, { method: "POST", body: a }).catch((e) => {
          console.log(e);
        });
        return null == n ? {} : n.json();
      }
      function K(e, t) {
        return JSON.stringify({ height: e, address: t });
      }
      function D(e, t) {
        return JSON.stringify({ pageNumber: e, address: t, perPage: 5 });
      }
      function V(e, t) {
        return JSON.stringify({ approve: t, proposal: e });
      }
      function M(e, t, a, n) {
        return JSON.stringify({ address: e, pollJSON: t, pollApprove: n, password: a, submit: !0 });
      }
      function q(e, t) {
        return JSON.stringify({ address: e, password: t, submit: !0 });
      }
      function J(e, t, a, n, r, s, o, i, l, d, c, u, p) {
        return JSON.stringify({
          address: e,
          pubKey: t,
          netAddress: n,
          committees: a,
          amount: r,
          delegate: s,
          earlyWithdrawal: o,
          output: i,
          signer: l,
          memo: d,
          fee: c,
          submit: u,
          password: p,
        });
      }
      function F(e, t, a, n, r, s, o, i, l, d) {
        return JSON.stringify({
          address: e,
          committees: t.toString(),
          orderId: a,
          amount: n,
          receiveAmount: r,
          receiveAddress: s,
          memo: o,
          fee: i,
          submit: l,
          password: d,
        });
      }
      function B(e, t, a, n, r, s, o, i, l, d, c) {
        return JSON.stringify({
          address: e,
          amount: t,
          paramSpace: a,
          paramKey: n,
          paramValue: r,
          startBlock: s,
          endBlock: o,
          memo: i,
          fee: l,
          submit: d,
          password: c,
        });
      }
      async function U() {
        return C(L, "/v1/admin/keystore");
      }
      async function G(e, t) {
        return R(L, "/v1/admin/keystore-get", q(e, t));
      }
      async function W(e) {
        return R(L, "/v1/admin/keystore-new-key", q("", e));
      }
      async function H(e, t) {
        return R(L, "/v1/admin/keystore-import-raw", JSON.stringify({ privateKey: e, password: t }));
      }
      async function z() {
        return Z(L, _);
      }
      async function Y(e, t) {
        return R(P, "/v1/query/account", K(e, t));
      }
      async function X() {
        return C(P, "/v1/gov/poll");
      }
      async function $() {
        return C(P, "/v1/gov/proposals");
      }
      async function Q(e, t) {
        return R(L, "/v1/gov/add-vote", V(JSON.parse(e), t));
      }
      async function ee(e) {
        return R(L, "/v1/gov/del-vote", V(JSON.parse(e)));
      }
      async function et(e, t, a) {
        return R(L, "/v1/admin/tx-start-poll", M(e, JSON.parse(t), a));
      }
      async function ea(e, t, a, n) {
        return R(L, "/v1/admin/tx-vote-poll", M(e, JSON.parse(t), n, a));
      }
      async function en(e, t, a) {
        let n = {};
        if (
          ((n.account = await Y(e, t)),
          (n.sent_transactions = await R(P, "/v1/query/txs-by-sender", D(a, t))),
          (n.rec_transactions = await R(P, "/v1/query/txs-by-rec", D(a, t))),
          (n.combined = []),
          null != n.sent_transactions.results && null != n.rec_transactions.results)
        )
          n.combined = n.combined.concat(n.rec_transactions.results, n.sent_transactions.results);
        else if (null != n.sent_transactions.results) n.combined = n.sent_transactions.results;
        else {
          if (null == n.rec_transactions.results) return n;
          n.combined = n.rec_transactions.results;
        }
        return (
          n.combined.sort(function (e, t) {
            return t.height === e.height ? t.index - e.index : t.height - e.height;
          }),
          n
        );
      }
      async function er(e, t) {
        return R(P, "/v1/query/validator", K(e, t));
      }
      async function es() {
        return C(L, "/v1/admin/resource-usage");
      }
      async function eo(e, t, a, n, r, s, o) {
        return R(L, "/v1/admin/tx-send", J(e, "", "", "", a, !1, !1, t, "", n, Number(r), o, s));
      }
      async function ei(e, t, a, n, r, s, o, i, l, d, c, u, p) {
        return R(
          L,
          "/v1/admin/tx-stake",
          J(e, t, a, n, r, "true" === s.toLowerCase(), "true" === o.toLowerCase(), i, l, d, Number(c), p, u),
        );
      }
      async function el(e, t, a, n, r, s, o, i, l, d, c) {
        return R(
          L,
          "/v1/admin/tx-edit-stake",
          J(e, "", t, a, n, !1, "true" === r.toLowerCase(), s, o, i, Number(l), c, d),
        );
      }
      async function ed(e, t, a, n, r, s) {
        return R(L, "/v1/admin/tx-unstake", J(e, "", "", "", 0, !1, !1, "", t, a, Number(n), s, r));
      }
      async function ec(e, t, a, n, r, s) {
        return R(L, "/v1/admin/tx-pause", J(e, "", "", "", 0, !1, !1, "", t, a, Number(n), s, r));
      }
      async function eu(e, t, a, n, r, s) {
        return R(L, "/v1/admin/tx-unpause", J(e, "", "", "", 0, !1, !1, "", t, a, Number(n), s, r));
      }
      async function ep(e, t, a, n, r, s, o, i, l, d) {
        return R(L, "/v1/admin/tx-change-param", B(e, 0, t, a, n, Number(r), Number(s), o, Number(i), d, l));
      }
      async function em(e, t, a, n, r, s, o, i) {
        return R(L, "/v1/admin/tx-dao-transfer", B(e, Number(t), "", "", "", Number(a), Number(n), r, Number(s), i, o));
      }
      async function eh(e, t, a, n, r, s, o, i, l) {
        return R(L, "/v1/admin/tx-create-order", F(e, t, 0, Number(a), Number(n), r, s, Number(o), l, i));
      }
      async function ex(e, t, a, n, r, s) {
        return R(
          L,
          "/v1/admin/tx-buy-order",
          JSON.stringify({ address: e, receiveAddress: t, orderId: Number(a), fee: Number(n), submit: s, password: r }),
        );
      }
      async function ef(e, t, a, n, r, s, o, i, l, d) {
        return R(L, "/v1/admin/tx-edit-order", F(e, t, Number(a), Number(n), Number(r), s, o, Number(i), d, l));
      }
      async function ey(e, t, a, n, r, s, o) {
        return R(L, "/v1/admin/tx-delete-order", F(e, t, Number(a), 0, 0, "", n, Number(r), o, s));
      }
      async function eg(e) {
        return R(P, "/v1/tx", e);
      }
      async function eb(e) {
        return R(P, "/v1/query/params", K(e, ""));
      }
      async function ev() {
        return C(L, E);
      }
      async function ej() {
        return C(L, A);
      }
      var ek = a(4373),
        ew = a(5377),
        eN = a(6529),
        eO = a(1417),
        eT = a(4283),
        eS = a(6374),
        eI = a(2448),
        eP = a(641),
        eL = a(5401),
        e_ = a(4568),
        eE = a(8888);
      let eA = [
        { title: "SEND", name: "send", src: "arrow-up" },
        { title: "STAKE", name: "stake", src: "stake" },
        { title: "EDIT", name: "edit-stake", src: "edit-stake" },
        { title: "UNSTAKE", name: "unstake", src: "unstake" },
        { title: "PAUSE", name: "pause", src: "pause" },
        { title: "PLAY", name: "unpause", src: "unpause" },
        { title: "SWAP", name: "create_order", src: "swap" },
        { title: "LOCK", name: "buy_order", src: "buy" },
        { title: "REPRICE", name: "edit_order", src: "edit_order" },
        { title: "VOID", name: "delete_order", src: "delete_order" },
      ];
      function eC(e) {
        let { keygroup: t, account: a, validator: r } = e,
          [s, o] = (0, l.useState)({
            showModal: !1,
            txType: "send",
            txResult: {},
            showSubmit: !0,
            showPKModal: !1,
            showPKImportModal: !1,
            showNewModal: !1,
            pk: {},
            toast: "",
            showSpinner: !1,
          }),
          i = a.account;
        function d() {
          o({
            ...s,
            pk: {},
            txResult: {},
            showSubmit: !0,
            showModal: !1,
            showPKModal: !1,
            showNewModal: !1,
            showPKImportModal: !1,
          });
        }
        function c(e) {
          b(s, e, (e) => {
            e.private_key
              ? H(e.private_key, e.password).then((e) => o({ ...s, showSpinner: !1 }))
              : W(e.password).then((e) => o({ ...s, showSpinner: !1 }));
          });
        }
        function u(e) {
          let t = arguments.length > 1 && void 0 !== arguments[1] ? arguments[1] : "outline-secondary",
            a = arguments.length > 2 && void 0 !== arguments[2] ? arguments[2] : "pk-button";
          return (0, n.jsx)(eN.Z, { id: a, variant: t, type: "submit", children: e });
        }
        function p() {
          let e = arguments.length > 0 && void 0 !== arguments[0] ? arguments[0] : d;
          return (0, n.jsx)(eN.Z, { variant: "secondary", onClick: e, children: "Close" });
        }
        function h(e, t) {
          return v(
            (0, n.jsxs)("td", {
              onClick: () => y(s, o, e),
              children: [
                (0, n.jsx)("img", { className: "account-summary-info-content-image", src: "./copy.png" }),
                (0, n.jsx)("div", {
                  className: "account-summary-info-table-column",
                  children: (0, n.jsx)(ew.Z, { text: e }),
                }),
              ],
            }),
            e,
            t,
            "top",
          );
        }
        function k(e, t, a, r, o, i, l, c, h) {
          return (0, n.jsx)(eS.Z, {
            show: e,
            size: "lg",
            onHide: c,
            children: (0, n.jsxs)(eT.Z, {
              onSubmit: r,
              children: [
                (0, n.jsx)(eS.Z.Header, { children: (0, n.jsx)(eS.Z.Title, { children: t }) }),
                (0, n.jsxs)(eS.Z.Body, {
                  className: "modal-body",
                  children: [
                    m(a, o, i, l).map((e, t) =>
                      (0, n.jsxs)(
                        eO.Z,
                        {
                          className: "mb-3",
                          size: "lg",
                          children: [
                            v(
                              (0, n.jsx)(eO.Z.Text, { className: "input-text", children: e.inputText }),
                              e.tooltip,
                              t,
                              "auto",
                            ),
                            (0, n.jsx)(eT.Z.Control, {
                              className: "input-text-field",
                              onChange: "number" === e.type ? N : w,
                              type: "number" == e.type ? "text" : e.type,
                              defaultValue: "number" === e.type ? x(e.defaultValue || "") : e.defaultValue,
                              placeholder: e.placeholder,
                              required: e.required,
                              min: 0,
                              minLength: e.minLength,
                              maxLength: e.maxLength,
                              "aria-label": e.label,
                            }),
                          ],
                        },
                        t,
                      ),
                    ),
                    (function () {
                      let { pk: e, txResult: t } = s,
                        a = j(e),
                        r = j(t);
                      return a && r
                        ? (0, n.jsx)(n.Fragment, {})
                        : (0, n.jsx)(ek.ZP, {
                            value: a ? { result: t } : { result: e },
                            shortenTextAfterLength: 100,
                            displayDataTypes: !1,
                          });
                    })(),
                    (0, n.jsx)(eI.Z, { style: { display: s.showSpinner ? "block" : "none", margin: "0 auto" } }),
                  ],
                }),
                (0, n.jsx)(eS.Z.Footer, {
                  children: (function (e) {
                    switch (e) {
                      case "import-or-generate":
                        return u("Import or Generate Key");
                      case "import-pk":
                        return (0, n.jsxs)(n.Fragment, { children: [u("Import Key", "outline-danger"), p(d)] });
                      case "new-pk":
                        return (0, n.jsxs)(n.Fragment, { children: [u("Generate New Key"), p(d)] });
                      case "reveal-pk":
                        return (0, n.jsxs)(n.Fragment, { children: [u("Get Private Key", "outline-danger"), p(d)] });
                      default:
                        if (0 === Object.keys(s.txResult).length)
                          return (0, n.jsxs)(n.Fragment, { children: [u("Generate Transaction"), p()] });
                        {
                          let e = s.showSubmit ? u("Submit Transaction", "outline-danger") : (0, n.jsx)(n.Fragment, {});
                          return (0, n.jsxs)(n.Fragment, { children: [e, p()] });
                        }
                    }
                  })(h),
                }),
              ],
            }),
          });
        }
        return t && 0 !== Object.keys(t).length && a.account
          ? (0, n.jsx)(n.Fragment, {
              children: (0, n.jsxs)("div", {
                className: "content-container",
                children: [
                  (0, n.jsx)("span", { id: "balance", children: f(i.amount) }),
                  (0, n.jsx)("span", { style: { fontWeight: "bold", color: "#32908f" }, children: " CNPY" }),
                  (0, n.jsx)("br", {}),
                  (0, n.jsx)("hr", {
                    style: { border: "1px dashed black", borderRadius: "5px", width: "60%", margin: "0 auto" },
                  }),
                  (0, n.jsx)("br", {}),
                  k(
                    s.showModal,
                    s.txType,
                    s.txType,
                    function (e) {
                      b(s, e, (e) => {
                        let t = 0 !== Object.keys(s.txResult).length,
                          a = {
                            send: () => eo(e.sender, e.recipient, O(e.amount), e.memo, e.fee, e.password, t),
                            stake: () =>
                              ei(
                                e.sender,
                                e.pubKey,
                                e.committees,
                                e.net_address,
                                O(e.amount),
                                e.delegate,
                                e.earlyWithdrawal,
                                e.output,
                                e.signer,
                                e.memo,
                                e.fee,
                                e.password,
                                t,
                              ),
                            "edit-stake": () =>
                              el(
                                e.sender,
                                e.committees,
                                e.net_address,
                                O(e.amount),
                                e.earlyWithdrawal,
                                e.output,
                                e.signer,
                                e.memo,
                                e.fee,
                                e.password,
                                t,
                              ),
                            unstake: () => ed(e.sender, e.signer, e.memo, e.fee, e.password, t),
                            pause: () => ec(e.sender, e.signer, e.memo, e.fee, e.password, t),
                            unpause: () => eu(e.sender, e.signer, e.memo, e.fee, e.password, t),
                            create_order: () =>
                              eh(
                                e.sender,
                                e.committeeId,
                                O(e.amount),
                                O(e.receiveAmount),
                                e.receiveAddress,
                                e.memo,
                                e.fee,
                                e.password,
                                t,
                              ),
                            buy_order: () => ex(e.sender, e.receiveAddress, O(e.orderId), e.fee, e.password, t),
                            edit_order: () =>
                              ef(
                                e.sender,
                                e.committeeId,
                                O(e.orderId),
                                O(e.amount),
                                O(e.receiveAmount),
                                e.receiveAddress,
                                e.memo,
                                e.fee,
                                e.password,
                                t,
                              ),
                            delete_order: () => ey(e.sender, e.committeeId, e.orderId, e.memo, e.fee, e.password, t),
                          }[s.txType];
                        a &&
                          a().then((e) => {
                            o({ ...s, showSubmit: !t, txResult: e });
                          });
                      });
                    },
                    t,
                    i,
                    r,
                    d,
                  ),
                  eA.map(function (e, t) {
                    return (0, n.jsxs)(
                      "div",
                      {
                        className: "send-receive-button-container",
                        children: [
                          (0, n.jsx)("img", {
                            className: "send-receive-button",
                            onClick: () => {
                              var t;
                              return (t = e.name), void o({ ...s, showModal: !0, txType: t });
                            },
                            src: "./".concat(e.src, ".png"),
                            alt: e.title,
                          }),
                          (0, n.jsx)("span", { style: { fontSize: "10px" }, children: e.title }),
                        ],
                      },
                      t,
                    );
                  }),
                  (0, n.jsx)(eE.Z, {
                    className: "account-summary-container",
                    children: [
                      {
                        title: "Account Type",
                        info: 0 === Object.keys(r).length || r.address === r.output ? "CUSTODIAL" : "NON-CUSTODIAL",
                      },
                      {
                        title: "Stake Amount",
                        info: null == r.staked_amount ? "0.00" : f(r.staked_amount),
                        after: " cnpy",
                      },
                      {
                        title: "Staked Status",
                        info: r.address ? (r.unstaking_time ? "UNSTAKING" : "STAKED") : "UNSTAKED",
                      },
                    ].map(function (e, t) {
                      return (0, n.jsx)(
                        eP.Z,
                        {
                          children: (0, n.jsxs)(eL.Z, {
                            className: "account-summary-container-card",
                            children: [
                              (0, n.jsx)(eL.Z.Header, { style: { fontWeight: "100" }, children: e.title }),
                              (0, n.jsx)(eL.Z.Body, {
                                style: { padding: "10px" },
                                children: (0, n.jsxs)(eL.Z.Title, {
                                  style: { fontWeight: "bold", fontSize: "14px" },
                                  children: [
                                    e.info,
                                    (0, n.jsx)("span", {
                                      style: { fontSize: "10px", color: "#32908f" },
                                      children: e.after,
                                    }),
                                  ],
                                }),
                              }),
                            ],
                          }),
                        },
                        t,
                      );
                    }),
                  }),
                  (0, n.jsx)("br", {}),
                  (0, n.jsx)("br", {}),
                  [
                    { title: "Address", info: i.address },
                    { title: "Public Key", info: t.publicKey },
                  ].map(function (e, t) {
                    return (0, n.jsxs)(
                      "div",
                      {
                        className: "account-summary-info",
                        onClick: () => y(s, o, e.info),
                        children: [
                          (0, n.jsx)("span", { className: "account-summary-info-title", children: e.title }),
                          (0, n.jsxs)("div", {
                            className: "account-summary-info-content-container",
                            children: [
                              (0, n.jsx)("div", {
                                className: "account-summary-info-content",
                                children: (0, n.jsx)(ew.Z, { text: e.info }),
                              }),
                              (0, n.jsx)("img", {
                                className: "account-summary-info-content-image",
                                style: { top: "-20px" },
                                src: "./copy.png",
                              }),
                            ],
                          }),
                        ],
                      },
                      t,
                    );
                  }),
                  (0, n.jsx)("br", {}),
                  0 === a.combined.length
                    ? null
                    : (0, n.jsxs)("div", {
                        className: "recent-transactions-table",
                        children: [
                          (0, n.jsx)("span", {
                            style: { textAlign: "center", fontWeight: "100", fontSize: "14px", color: "grey" },
                            children: "RECENT TRANSACTIONS",
                          }),
                          (0, n.jsxs)(e_.Z, {
                            className: "table-fixed",
                            bordered: !0,
                            hover: !0,
                            style: { marginTop: "10px" },
                            children: [
                              (0, n.jsx)("thead", {
                                children: (0, n.jsx)("tr", {
                                  children: ["Height", "Amount", "Recipient", "Type", "Hash"].map((e, t) =>
                                    (0, n.jsx)("th", { children: e }, t),
                                  ),
                                }),
                              }),
                              (0, n.jsx)("tbody", {
                                children: a.combined.slice(0, 5).map((e, t) => {
                                  var a, r, s;
                                  return (0, n.jsxs)(
                                    "tr",
                                    {
                                      children: [
                                        (0, n.jsx)("td", { children: e.height }),
                                        (0, n.jsx)("td", {
                                          children:
                                            null !==
                                              (r =
                                                null !== (a = e.transaction.msg.amount) && void 0 !== a
                                                  ? a
                                                  : e.transaction.msg.AmountForSale) && void 0 !== r
                                              ? r
                                              : "N/A",
                                        }),
                                        h(null !== (s = e.recipient) && void 0 !== s ? s : e.sender, t),
                                        (0, n.jsx)("td", { children: e.message_type }),
                                        h(e.tx_hash, t),
                                      ],
                                    },
                                    t,
                                  );
                                }),
                              }),
                            ],
                          }),
                        ],
                      }),
                  g(s, o),
                  k(
                    s.showPKModal,
                    "Private Key",
                    "pass-and-addr",
                    function (e) {
                      b(s, e, (e) =>
                        G(e.sender, e.password).then((e) => {
                          o({ ...s, showSubmit: 0 === Object.keys(s.txResult).length, pk: e });
                        }),
                      );
                    },
                    t,
                    i,
                    null,
                    d,
                    "reveal-pk",
                  ),
                  k(
                    s.showPKImportModal,
                    "Private Key",
                    "pass-and-pk",
                    () => {
                      c(), d();
                    },
                    t,
                    i,
                    null,
                    d,
                    "import-pk",
                  ),
                  k(
                    s.showNewModal,
                    "Private Key",
                    "pass-only",
                    function (e) {
                      b(s, e, (e) =>
                        W(e.password).then((e) => {
                          o({ ...s, showSubmit: 0 === Object.keys(s.txResult).length, pk: e });
                        }),
                      );
                    },
                    null,
                    null,
                    null,
                    d,
                    "new-pk",
                  ),
                  (0, n.jsx)(eN.Z, {
                    id: "pk-button",
                    variant: "outline-secondary",
                    onClick: () => o({ ...s, showNewModal: !0 }),
                    children: "New Private Key",
                  }),
                  (0, n.jsx)(eN.Z, {
                    id: "import-pk-button",
                    variant: "outline-secondary",
                    onClick: () => o({ ...s, showPKImportModal: !0 }),
                    children: "Import Private Key",
                  }),
                  (0, n.jsx)(eN.Z, {
                    id: "reveal-pk-button",
                    variant: "outline-danger",
                    onClick: () => o({ ...s, showPKModal: !0 }),
                    children: "Reveal Private Key",
                  }),
                ],
              }),
            })
          : k(!0, "UPLOAD PRIVATE OR CREATE KEY", "pass-and-pk", c, null, null, null, null, "import-or-generate");
      }
      var eZ = a(5152),
        eR = a.n(eZ),
        eK = a(7123),
        eD = a(6596),
        eV = a(6205),
        eM = a(9709),
        eq = a(6910),
        eJ = a(3447);
      let eF = eR()(
        () =>
          a
            .e(391)
            .then(a.t.bind(a, 8391, 23))
            .then((e) => e.LazyLog),
        { loadableGenerated: { webpack: () => [8391] }, ssr: !1 },
      );
      function eB() {
        let [e, t] = (0, l.useState)({
          logs: "retrieving logs...",
          pauseLogs: !1,
          resource: [],
          consensusInfo: {},
          peerInfo: {},
        });
        function a() {
          let a = [ev(), ej(), es()];
          e.pauseLogs || a.push(z()),
            Promise.all(a).then((e) => {
              let [a, n, r, s] = e;
              t((e) => {
                let t = e.resource.length >= 30 ? [...e.resource.slice(1), r] : [...e.resource, r];
                return {
                  ...e,
                  consensusInfo: a,
                  peerInfo: n,
                  resource: t,
                  ...(e.pauseLogs ? {} : { logs: s.toString() }),
                };
              });
            });
        }
        if (
          ((0, l.useEffect)(() => {
            let e = setInterval(a, 1e3);
            return () => clearInterval(e);
          }, []),
          !e.consensusInfo.view || !e.peerInfo.id)
        )
          return a(), (0, n.jsx)(eI.Z, { id: "spinner" });
        let s = Number(e.peerInfo.numInbound),
          o = Number(e.peerInfo.numOutbound),
          i = e.consensusInfo.view,
          d = (function (e, t) {
            let [a, n] = e > t ? [e, t] : [t, e];
            for (let r = 1; r < 1e6; r++) {
              let s = a / (n / r);
              if (0.1 > Math.abs(s - Math.round(s)))
                return e > t ? "".concat(Math.round(s), ":").concat(r) : "".concat(r, ":").concat(Math.round(s));
            }
          })((s = s || 0), (o = o || 0)),
          c = [
            {
              slides: [
                {
                  title: e.consensusInfo.syncing ? "SYNCING" : "SYNCED",
                  dT: "H: " + f(i.height, !1) + ", R: " + i.round + ", P: " + i.phase,
                  d1: "PROP: " + ("" === e.consensusInfo.proposer ? "UNDECIDED" : e.consensusInfo.proposer),
                  d2: "BLK: " + ("" === e.consensusInfo.blockHash ? "WAITING" : e.consensusInfo.blockHash),
                  d3: e.consensusInfo.status,
                },
                {
                  title:
                    "ROUND PROGRESS: " +
                    ({
                      ELECTION: 0,
                      "ELECTION-VOTE": 1,
                      PROPOSE: 2,
                      "PROPOSE-VOTE": 3,
                      PRECOMMIT: 4,
                      "PRECOMMIT-VOTE": 5,
                      COMMIT: 6,
                      "COMMIT-PROCESS": 7,
                    }[e.consensusInfo.view.phase] /
                      8) *
                      100 +
                    "%",
                  dT: "ADDRESS: " + e.consensusInfo.address,
                  d1: "",
                  d2: "",
                  d3: "",
                },
              ],
              btnSlides: [
                { url: L + E, title: "QUORUM" },
                { url: L + "/v1/admin/config", title: "CONFIG" },
                { url: L + _, title: "LOGGER" },
              ],
            },
            {
              slides: [
                {
                  title: "TOTAL PEERS: " + (null == e.peerInfo.numPeers ? "0" : e.peerInfo.numPeers),
                  dT: "INBOUND: " + s + ", OUTBOUND: " + o,
                  d1: "ID: " + e.peerInfo.id.public_key,
                  d2:
                    "NET ADDR: " + (e.peerInfo.id.net_address ? e.peerInfo.id.net_address : "External Address Not Set"),
                  d3: "I / O RATIO " + (d || "0:0"),
                },
              ],
              btnSlides: [
                { url: L + "/v1/admin/peer-book", title: "PEER BOOK" },
                { url: L + A, title: "PEER INFO" },
              ],
            },
          ];
        return (0, n.jsxs)("div", {
          className: "content-container",
          id: "dashboard-container",
          children: [
            (0, n.jsx)(r.Z, {
              id: "dashboard-inner",
              fluid: !0,
              children: (0, n.jsx)(eE.Z, {
                children: c.map((e, t) => {
                  var a;
                  return (0, n.jsx)(
                    eP.Z,
                    {
                      children: (0, n.jsxs)(eK.Z, {
                        slide: !1,
                        interval: null,
                        className: "carousel",
                        children: [
                          e.slides.map((e, t) =>
                            (0, n.jsx)(
                              eK.Z.Item,
                              {
                                children: (0, n.jsx)(eL.Z, {
                                  className: "carousel-item-container",
                                  children: (0, n.jsxs)(eL.Z.Body, {
                                    children: [
                                      (0, n.jsx)(eL.Z.Title, {
                                        className: "carousel-item-title",
                                        children: (0, n.jsx)("span", { className: "text-white", children: e.title }),
                                      }),
                                      (0, n.jsx)("p", {
                                        id: "carousel-item-detail-title",
                                        className: "carousel-item-detail",
                                        children: (0, n.jsx)(ew.Z, { text: e.dT }),
                                      }),
                                      (0, n.jsx)("p", {
                                        className: "carousel-item-detail",
                                        children: (0, n.jsx)(ew.Z, { text: e.d1 }),
                                      }),
                                      (0, n.jsx)("p", {
                                        className: "carousel-item-detail",
                                        children: (0, n.jsx)(ew.Z, { text: e.d2 }),
                                      }),
                                      (0, n.jsx)("p", { children: e.d3 }),
                                    ],
                                  }),
                                }),
                              },
                              t,
                            ),
                          ),
                          ((a = e.btnSlides),
                          (0, n.jsx)(eK.Z.Item, {
                            children: (0, n.jsx)(eL.Z, {
                              className: "carousel-item-container",
                              children: (0, n.jsxs)(eL.Z.Body, {
                                children: [
                                  (0, n.jsx)(eL.Z.Title, { children: "EXPLORE RAW JSON" }),
                                  (0, n.jsx)("div", {
                                    children: a.map((e, t) =>
                                      (0, n.jsx)(
                                        eN.Z,
                                        {
                                          className: "carousel-btn",
                                          variant: "outline-secondary",
                                          onClick: () => window.open(e.url, "_blank"),
                                          children: e.title,
                                        },
                                        t,
                                      ),
                                    ),
                                  }),
                                ],
                              }),
                            }),
                          })),
                        ],
                      }),
                    },
                    t,
                  );
                }),
              }),
            }),
            (0, n.jsx)("hr", { id: "dashboard-hr" }),
            (0, n.jsx)("div", {
              onClick: () => t({ ...e, pauseLogs: !e.pauseLogs }),
              className: "logs-button-container",
              children: (0, n.jsx)("img", {
                className: "logs-button",
                alt: "play-pause-btn",
                src: e.pauseLogs ? "./unpause_filled.png" : "./pause_filled.png",
              }),
            }),
            (0, n.jsx)(eF, { enableSearch: !0, id: "lazy-log", text: e.logs.replace("\n", "") }),
            (0, n.jsx)(r.Z, {
              id: "charts-container",
              children: [
                [
                  {
                    yax: "PROCESS",
                    n1: "CPU %",
                    d1: "process.usedCPUPercent",
                    n2: "RAM %",
                    d2: "process.usedMemoryPercent",
                  },
                  { yax: "SYSTEM", n1: "CPU %", d1: "system.usedCPUPercent", n2: "RAM %", d2: "system.usedRAMPercent" },
                ],
                [
                  { yax: "DISK", n1: "Disk %", d1: "system.usedDiskPercent", n2: "" },
                  {
                    yax: "IN OUT",
                    removeTick: !0,
                    n1: "Received",
                    d1: "system.ReceivedBytesIO",
                    n2: "Written",
                    d2: "system.WrittenBytesIO",
                  },
                ],
                [
                  { yax: "THREADS", n1: "Thread Count", d1: "process.threadCount", n2: "" },
                  { yax: "FILES", n1: "File Descriptors", d1: "process.fdCount", n2: "" },
                ],
              ].map((t, a) =>
                (0, n.jsx)(
                  eE.Z,
                  {
                    children: [void 0, void 0].map((a, r) => {
                      let s =
                        "" === t[r].n2
                          ? (0, n.jsx)(n.Fragment, {})
                          : (0, n.jsx)(eD.u, {
                              name: t[r].n2,
                              type: "monotone",
                              dataKey: t[r].d2,
                              stroke: "#848484",
                              fillOpacity: 1,
                              fill: "url(#cpu)",
                            });
                      return (0, n.jsx)(eP.Z, {
                        children: (0, n.jsxs)(eV.T, {
                          className: "area-chart",
                          width: 600,
                          height: 250,
                          data: e.resource,
                          margin: { top: 40, right: 40 },
                          children: [
                            (0, n.jsx)(eM.B, {
                              tick: !t[r].removeTick,
                              tickCount: 1,
                              label: { value: t[r].yax, angle: -90 },
                            }),
                            (0, n.jsx)(eD.u, {
                              name: t[r].n1,
                              type: "monotone",
                              dataKey: t[r].d1,
                              stroke: "#eeeeee",
                              fillOpacity: 1,
                              fill: "url(#ram)",
                            }),
                            s,
                            (0, n.jsx)(eq.u, { contentStyle: { backgroundColor: "#222222" } }),
                            (0, n.jsx)(eJ.D, {}),
                          ],
                        }),
                      });
                    }),
                  },
                  a,
                ),
              ),
            }),
            (0, n.jsx)("div", { style: { height: "50px", width: "100%" } }),
          ],
        });
      }
      var eU = a(6495),
        eG = a(3148),
        eW = a(2311);
      function eH(e) {
        let { keygroup: t, account: a, validator: s } = e,
          [o, i] = (0, l.useState)({
            txResult: {},
            rawTx: {},
            showPropModal: !1,
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
        function d() {
          Promise.all([X(), $(), eb(0)]).then((e) => {
            i((t) => ({ ...t, apiResults: { poll: e[0], proposals: e[1], params: e[2] } }));
          });
        }
        if (
          ((0, l.useEffect)(() => {
            let e = setInterval(() => {
              d();
            }, 4e3);
            return () => clearInterval(e);
          }, []),
          j(o.apiResults))
        )
          return d(), (0, n.jsx)(eI.Z, { id: "spinner" });
        function c(e, t) {
          return Q(e, t).then((e) => i({ ...o, voteOnProposalAccord: "1", toast: "Voted!" }));
        }
        function u(e, t, a, n) {
          return ea(e, t, a, n).then((e) => i({ ...o, voteOnPollAccord: "1", toast: "Voted!" }));
        }
        function p() {
          i({ ...o, paramSpace: "", txResult: {}, showPropModal: !1 });
        }
        function x(e) {
          i({ ...o, txPropType: e, showPropModal: !0, paramSpace: "", txResult: {} });
        }
        function f(e, t) {
          return (0, n.jsxs)(n.Fragment, {
            children: [
              (0, n.jsx)("img", { className: "governance-header-image", alt: "vote", src: t }),
              (0, n.jsx)("span", { id: "propose-title", children: e }),
              (0, n.jsx)("span", { id: "propose-subtitle", children: "  on CANOPY" }),
              (0, n.jsx)("br", {}),
              (0, n.jsx)("br", {}),
              (0, n.jsx)("hr", { className: "gov-header-hr" }),
              (0, n.jsx)("br", {}),
              (0, n.jsx)("br", {}),
            ],
          });
        }
        function k(e, t, a, r, s) {
          let l = arguments.length > 5 && void 0 !== arguments[5] ? arguments[5] : h.params,
            d = (e, t) => i({ ...o, [e]: t, ...(e === a && { voteJSON: t }) });
          return (0, n.jsx)(eW.Z, {
            className: "accord",
            activeKey: o[t],
            onSelect: (e) => d(t, e),
            children: (0, n.jsxs)(eW.Z.Item, {
              className: "accord-item",
              eventKey: "0",
              children: [
                (0, n.jsx)(eW.Z.Header, { children: e }),
                (0, n.jsxs)(eW.Z.Body, {
                  children: [
                    (0, n.jsx)(eT.Z.Control, {
                      className: "accord-body-container",
                      defaultValue: JSON.stringify(l, null, 2),
                      as: "textarea",
                      onChange: (e) => d(a, e.target.value),
                    }),
                    s &&
                      (0, n.jsxs)(eO.Z, {
                        className: "accord-pass-container",
                        size: "lg",
                        children: [
                          (0, n.jsx)(eO.Z.Text, { children: "Password" }),
                          (0, n.jsx)(eT.Z.Control, {
                            type: "password",
                            onChange: (e) => d("pwd", e.target.value),
                            required: !0,
                          }),
                        ],
                      }),
                    r.map((e, t) =>
                      (0, n.jsx)(
                        eN.Z,
                        { className: "propose-button", onClick: e.onClick, variant: "outline-dark", children: e.title },
                        t,
                      ),
                    ),
                  ],
                }),
              ],
            }),
          });
        }
        return (
          j(o.apiResults.poll)
            ? (o.apiResults.poll = h.poll)
            : (o.apiResults.poll["PLACEHOLDER EXAMPLE"] = h.poll["PLACEHOLDER EXAMPLE"]),
          j(o.apiResults.proposals) && (o.apiResults.proposals = h.proposals),
          (0, n.jsx)(n.Fragment, {
            children: (0, n.jsxs)("div", {
              className: "content-container",
              children: [
                f("poll", "./poll.png"),
                (0, n.jsx)(eK.Z, {
                  className: "poll-carousel",
                  interval: null,
                  "data-bs-theme": "dark",
                  children: Array.from(Object.entries(o.apiResults.poll)).map((e, t) => {
                    let [a, s] = e;
                    return (0, n.jsxs)(
                      eK.Z.Item,
                      {
                        children: [
                          (0, n.jsx)("h6", { className: "poll-prop-hash", children: s.proposalHash }),
                          (0, n.jsx)("a", { href: s.proposalURL, className: "poll-prop-url", children: s.proposalURL }),
                          (0, n.jsxs)(r.Z, {
                            className: "poll-carousel-container",
                            fluid: !0,
                            children: [
                              (0, n.jsx)(eU.$Q, {
                                data: {
                                  labels: [
                                    s.accounts.votedPercent + "% Accounts Reporting",
                                    s.validators.votedPercent + "% Validators Reporting",
                                  ],
                                  datasets: [
                                    {
                                      label: "% Voted YES",
                                      data: [s.accounts.approvedPercent, s.validators.approvedPercent],
                                      backgroundColor: "#7749c0",
                                    },
                                    {
                                      label: "% Voted NO",
                                      data: [s.accounts.rejectPercent, s.validators.rejectPercent],
                                      backgroundColor: "#000",
                                    },
                                  ],
                                },
                                options: {
                                  responsive: !0,
                                  plugins: { tooltip: { enabled: !0 } },
                                  scales: { y: { beginAtZero: !0, max: 100 } },
                                },
                              }),
                              (0, n.jsx)("br", {}),
                            ],
                          }),
                        ],
                      },
                      t,
                    );
                  }),
                }),
                (0, n.jsx)("br", {}),
                k(
                  "START OR VOTE ON POLL",
                  "voteOnPollAccord",
                  "voteJSON",
                  [
                    {
                      title: "START NEW",
                      onClick: () =>
                        et(a.account.address, o.voteJSON, o.pwd).then((e) =>
                          i({ ...o, voteOnPollAccord: "1", toast: "Started Poll!" }),
                        ),
                    },
                    { title: "APPROVE", onClick: () => u(a.account.address, o.voteJSON, !0, o.pwd) },
                    { title: "REJECT", onClick: () => u(a.account.address, o.voteJSON, !1, o.pwd) },
                  ],
                  !0,
                  h.pollJSON,
                ),
                f("propose", "./proposal.png"),
                (0, n.jsxs)(e_.Z, {
                  className: "vote-table",
                  bordered: !0,
                  responsive: !0,
                  hover: !0,
                  children: [
                    (0, n.jsx)("thead", {
                      children: (0, n.jsxs)("tr", {
                        children: [
                          (0, n.jsx)("th", { children: "VOTE" }),
                          (0, n.jsx)("th", { children: "PROPOSAL ID" }),
                          (0, n.jsx)("th", { children: "ENDS" }),
                        ],
                      }),
                    }),
                    (0, n.jsx)("tbody", {
                      children: Array.from(
                        Object.entries(o.apiResults.proposals).map((e, t) => {
                          let [a, r] = e;
                          return (0, n.jsxs)(
                            "tr",
                            {
                              children: [
                                (0, n.jsx)("td", { children: r.approve ? "YES" : "NO" }),
                                (0, n.jsx)("td", {
                                  children: (0, n.jsx)("div", {
                                    className: "vote-table-col",
                                    children: (0, n.jsx)(ew.Z, { text: "#" + a }),
                                  }),
                                }),
                                (0, n.jsx)("td", { children: r.proposal.end_height }),
                              ],
                            },
                            t,
                          );
                        }),
                      ),
                    }),
                  ],
                }),
                k(
                  "VOTE ON PROPOSAL",
                  "voteOnProposalAccord",
                  "voteJSON",
                  [
                    { title: "APPROVE", onClick: () => c(o.voteJSON, !0) },
                    { title: "REJECT", onClick: () => c(o.voteJSON, !1) },
                    {
                      title: "DELETE",
                      onClick: () =>
                        ee(o.voteJSON).then((e) => i({ ...o, voteOnProposalAccord: "1", toast: "Deleted!" })),
                    },
                  ],
                  !1,
                ),
                g(o, i),
                (0, n.jsx)("br", {}),
                k(
                  "SUBMIT PROPOSAL",
                  "propAccord",
                  "rawTx",
                  [
                    {
                      title: "SUBMIT",
                      onClick: () =>
                        eg(o.rawTx).then((e) => {
                          y(o, i, e, "tx hash copied to keyboard!");
                        }),
                    },
                  ],
                  !1,
                  h.rawTx,
                ),
                (0, n.jsx)(eN.Z, {
                  className: "propose-button",
                  onClick: () => x(0),
                  variant: "outline-dark",
                  children: "New Protocol Change",
                }),
                (0, n.jsx)(eN.Z, {
                  className: "propose-button",
                  onClick: () => x(1),
                  variant: "outline-dark",
                  children: "New Treasury Subsidy",
                }),
                (0, n.jsx)("br", {}),
                (0, n.jsx)("br", {}),
                (0, n.jsx)(eS.Z, {
                  show: o.showPropModal,
                  size: "lg",
                  onHide: p,
                  children: (0, n.jsxs)(eT.Z, {
                    onSubmit: function (e) {
                      b(o, e, (e) => {
                        0 === o.txPropType
                          ? (function (e, t, a, n, r, s, l, d, c) {
                              ep(e, t, a, n, r, s, l, d, c, !1).then((e) => {
                                i({ ...o, txResult: e });
                              });
                            })(
                              e.sender,
                              e.param_space,
                              e.param_key,
                              e.param_value,
                              e.start_block,
                              e.end_block,
                              e.memo,
                              e.fee,
                              e.password,
                            )
                          : (function (e, t, a, n, r, s, l) {
                              em(e, t, a, n, r, s, l, !1).then((e) => {
                                i({ ...o, txResult: e });
                              });
                            })(e.sender, e.amount, e.start_block, e.end_block, e.memo, e.fee, e.password);
                      });
                    },
                    children: [
                      (0, n.jsx)(eS.Z.Header, {
                        closeButton: !0,
                        children: (0, n.jsx)(eS.Z.Title, {
                          children: 0 === o.txPropType ? "Change Parameter" : "Treasury Subsidy",
                        }),
                      }),
                      (0, n.jsxs)(eS.Z.Body, {
                        style: { overflowWrap: "break-word" },
                        children: [
                          m(0 === o.txPropType ? "change-param" : "dao-transfer", t, a.account, s).map((e, t) => {
                            let a = j(o.txResult) ? "" : "none";
                            if ("select" === e.type)
                              switch (e.label) {
                                case "param_key":
                                  let r = [],
                                    s = o.apiResults.params[o.paramSpace];
                                  return (
                                    s && (r = Object.keys(s)),
                                    (0, n.jsx)(
                                      eO.Z,
                                      {
                                        style: { display: a },
                                        className: "mb-3",
                                        size: "lg",
                                        children: (0, n.jsxs)(eT.Z.Select, {
                                          size: "lg",
                                          "aria-label": e.label,
                                          children: [
                                            (0, n.jsx)("option", { children: "param key" }),
                                            r.map((e, t) => (0, n.jsx)("option", { value: e, children: e }, t)),
                                          ],
                                        }),
                                      },
                                      t,
                                    )
                                  );
                                case "param_space":
                                  return (0, n.jsx)(
                                    eO.Z,
                                    {
                                      style: { display: a },
                                      className: "mb-3",
                                      size: "lg",
                                      children: (0, n.jsxs)(eT.Z.Select, {
                                        size: "lg",
                                        onChange: (e) => i({ ...o, paramSpace: e.target.value }),
                                        "aria-label": e.label,
                                        children: [
                                          (0, n.jsx)("option", { children: "param space" }),
                                          (0, n.jsx)("option", { value: "Consensus", children: "consensus" }),
                                          (0, n.jsx)("option", { value: "Validator", children: "validator" }),
                                          (0, n.jsx)("option", { value: "Governance", children: "governance" }),
                                          (0, n.jsx)("option", { value: "Fee", children: "fee" }),
                                        ],
                                      }),
                                    },
                                    t,
                                  );
                              }
                            return (0, n.jsxs)(
                              eO.Z,
                              {
                                style: { display: a },
                                className: "mb-3",
                                size: "lg",
                                children: [
                                  v(
                                    (0, n.jsx)(eO.Z.Text, { className: "param-input", children: e.inputText }),
                                    e.tooltip,
                                    t,
                                  ),
                                  (0, n.jsx)(eT.Z.Control, {
                                    placeholder: e.placeholder,
                                    required: e.required,
                                    defaultValue: e.defaultValue,
                                    type: e.type,
                                    min: 0,
                                    minLength: e.minLength,
                                    maxLength: e.maxLength,
                                    "aria-label": e.label,
                                  }),
                                ],
                              },
                              t,
                            );
                          }),
                          j(o.txResult)
                            ? (0, n.jsx)(n.Fragment, {})
                            : (0, n.jsx)(ek.ZP, {
                                value: o.txResult,
                                shortenTextAfterLength: 100,
                                displayDataTypes: !1,
                              }),
                        ],
                      }),
                      (0, n.jsx)(eS.Z.Footer, {
                        children: (0, n.jsxs)(n.Fragment, {
                          children: [
                            (0, n.jsx)(eN.Z, {
                              style: { display: j(o.txResult) ? "" : "none" },
                              id: "import-pk-button",
                              variant: "outline-secondary",
                              type: "submit",
                              children: "Generate New Proposal",
                            }),
                            (0, n.jsx)(eN.Z, { variant: "secondary", onClick: p, children: "Close" }),
                          ],
                        }),
                      }),
                    ],
                  }),
                }),
              ],
            }),
          })
        );
      }
      function ez() {
        let [e, t] = (0, l.useState)({ navIdx: 0, keystore: null, keyIdx: 0, account: {}, validator: {} });
        function a() {
          let a = arguments.length > 0 && void 0 !== arguments[0] ? arguments[0] : e.keyIdx;
          U().then((n) => {
            Promise.all([en(0, Object.keys(n)[a], 0), er(0, Object.keys(n)[a])]).then((r) => {
              t({ ...e, keyIdx: a, keystore: n, account: r[0], validator: r[1] });
            });
          });
        }
        return ((0, l.useEffect)(() => {
          let e = setInterval(() => {
            a();
          }, 4e3);
          return () => clearInterval(e);
        }),
        null === e.keystore)
          ? (a(), (0, n.jsx)(eI.Z, { id: "spinner" }))
          : (0, n.jsx)(n.Fragment, {
              children: (0, n.jsxs)("div", {
                id: "container",
                children: [
                  (0, n.jsx)(I, { ...e, setActiveKey: a, setNavIdx: (a) => t({ ...e, navIdx: a }) }),
                  (0, n.jsx)("div", {
                    id: "pageContent",
                    children:
                      0 === e.navIdx
                        ? (0, n.jsx)(eC, { keygroup: Object.values(e.keystore)[e.keyIdx], ...e })
                        : 1 === e.navIdx
                          ? (0, n.jsx)(eH, { keygroup: Object.values(e.keystore)[e.keyIdx], ...e })
                          : (0, n.jsx)(eB, {}),
                  }),
                ],
              }),
            });
      }
      eG.kL.register(eG.ZL, eG.uw, eG.f$, eG.u, eG.De);
    },
  },
  function (e) {
    e.O(0, [196, 473, 888, 774, 179], function () {
      return e((e.s = 5557));
    }),
      (_N_E = e.O());
  },
]);
