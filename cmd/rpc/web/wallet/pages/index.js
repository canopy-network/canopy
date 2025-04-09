import Navigation from "@/components/navbar";
import { AccountWithTxs, Height, Keystore, Params, Validator, CommitteesData } from "@/components/api";
import { createContext, use, useEffect, useState } from "react";
import Accounts from "@/components/account";
import Dashboard from "@/components/dashboard";
import Governance from "@/components/governance";
import Footer from "@/components/footer";
import { Spinner } from "react-bootstrap";

export const KeystoreContext = createContext();

export default function Home() {
  const [state, setState] = useState({
    navIdx: 0,
    keystore: null,
    keyIdx: 0,
    account: {},
    validator: {},
    height: 0,
    keys: [],
    params: {},
  });
  const setNavIdx = (i) => setState({ ...state, navIdx: i });

  function queryAPI(i = state.keyIdx) {
    Keystore().then((ks) => {
      if (!ks.addressMap || Object.keys(ks.addressMap).length === 0) {
        console.warn("mergedKS is empty. No data to query.");
        setState({ ...state, keystore: {}, account: {}, validator: {} }); // Handle empty case
        return;
      }

      const mergedKS = Object.entries(ks.addressMap).reduce((acc, [address, details]) => {
        const key = details.keyNickname || address;
        acc[key] = details;
        return acc;
      }, {});

      Promise.allSettled([
        AccountWithTxs(0, mergedKS[Object.keys(mergedKS)[i]].keyAddress, Object.keys(mergedKS)[i], 0),
        Validator(0, mergedKS[Object.keys(mergedKS)[i]].keyAddress, Object.keys(mergedKS)[i]),
        Height(),
        Params(0),
        CommitteesData(0),
      ]).then((results) => {
        let settledResults = [];
        for (const result of results) {
          if (result.status === "rejected") {
            settledResults.push({});
            continue;
          }
          settledResults.push(result.value);
        }

        let knownCommittees =
          Object.keys(settledResults[4]).length > 0 ? settledResults[4].list.map((committee) => committee.chainID) : [];
        let validator = Object.keys(settledResults[1]).length > 0 ? { ...settledResults[1], knownCommittees } : {};

        setState({
          ...state,
          keys: Object.keys(mergedKS),
          keyIdx: i,
          keystore: mergedKS,
          account: settledResults[0],
          validator: validator,
          height: settledResults[2],
          params: settledResults[3],
        });
      });
    });
  }

  // Initial API call on component mount
  useEffect(() => {
    queryAPI();
  }, []); // Empty dependency array ensures this runs only once on mount

  useEffect(() => {
    const i = setInterval(() => {
      queryAPI();
    }, 4000);
    return () => clearInterval(i);
  });

  if (state.keystore === null) {
    return <Spinner id="spinner" />;
  }
  return (
    <KeystoreContext.Provider value={state.keystore}>
      <div id="container" class="content-light">
        <Navigation {...state} setActiveKey={queryAPI} setNavIdx={setNavIdx} />
        <div id="pageContent">
          {state.navIdx == 0 && (
            <Accounts keygroup={Object.values(state.keystore)[state.keyIdx]} setActiveKey={queryAPI} {...state} />
          )}
          {state.navIdx == 1 && <Governance keygroup={Object.values(state.keystore)[state.keyIdx]} {...state} />}
          {state.navIdx == 2 && <Dashboard />}
        </div>
      </div>
      <Footer />
    </KeystoreContext.Provider>
  );
}
