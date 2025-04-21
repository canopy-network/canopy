import Navigation from "@/components/navbar";
import Sidebar from "@/components/sidebar";
import DTable from "@/components/table";
import DetailModal from "@/components/modal";
import Cards from "@/components/cards";
import { useEffect, useState } from "react";
import { Spinner, Toast, ToastContainer } from "react-bootstrap";
import { getCardData, getTableData, getModalData, Config } from "@/components/api";

export default function Home() {
  const [state, setState] = useState({
    loading: true,
    cardData: {},
    category: 0,
    tablePage: 0,
    tableData: {},
    showToast: false,
    consensusDuration: 0,

    sortColumn: null,
    sortDirection: "asc",
    filterText: "",
    committee: 1,

    modalState: {
      show: false,
      page: 0,
      query: "",
      data: {},
      accOnly: false,
    },
  });

  function getCardAndTableData(setLoading) {
    Promise.allSettled([getTableData(state.tablePage, state.category, state.committee), getCardData(), Config()]).then(
      (values) => {
        let settledValues = [];
        for (const v of values) {
          if (v.status === "rejected") {
            settledValues.push({});
            continue;
          }
          settledValues.push(v.value);
        }

        const consensusDuration =
          settledValues[2].electionTimeoutMS +
          settledValues[2].electionVoteTimeoutMS +
          settledValues[2].proposeTimeoutMS +
          settledValues[2].proposeVoteTimeoutMS +
          settledValues[2].precommitTimeoutMS +
          settledValues[2].precommitVoteTimeoutMS +
          settledValues[2].commitTimeoutMS;

        if (setLoading) {
          return setState({
            ...state,
            loading: false,
            tableData: settledValues[0],
            cardData: settledValues[1],
            consensusDuration: consensusDuration,
          });
        }
        return setState({
          ...state,
          tableData: settledValues[0],
          cardData: settledValues[1],
          consensusDuration: consensusDuration,
        });
      },
    );
  }

  async function openModal(query) {
    let data = await getModalData(query, 0);
    setState({
      ...state,
      modalState: {
        show: true,
        query: query,
        page: 0,
        accOnly: !Boolean(data?.validator),
        data,
      },
    });
  }

  async function selectTable(category, page, committee) {
    if (committee == null) {
      committee = state.committee;
    }
    setState({
      ...state,
      committee: committee,
      category: category,
      tablePage: page,
      tableData: await getTableData(page, category, committee),
    });
  }

  useEffect(() => {
    const interval = setInterval(() => {
      getCardAndTableData(false);
    }, 4000);
    return () => clearInterval(interval);
  });
  if (state.loading || !state.cardData.blocks) {
    getCardAndTableData(true);
    return (
      <>
        <Spinner id="spinner" />
        <br /> <br />
        <center>
          <h3>Waiting to explore!</h3>
        </center>
        <center>
          <h7>... loading or no blocks yet ...</h7>
        </center>
      </>
    );
  } else {
    const props = { state, setState, openModal, selectTable };
    const onToastClose = () => setState({ ...state, showToast: false });
    return (
      <>
        <div id="container">
          <Navigation {...props} />
          <Sidebar {...props} />
          <div id="pageContent">
            <Cards {...props} />
            <DTable {...props} />
            <DetailModal {...props} />
            <ToastContainer id="toast" position={"bottom-end"}>
              <Toast bg={"dark"} onClose={onToastClose} show={state.showToast} delay={2000} autohide>
                <Toast.Body>Copied!</Toast.Body>
              </Toast>
            </ToastContainer>
          </div>
        </div>
      </>
    );
  }
}
