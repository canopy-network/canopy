import { useEffect, useState } from "react";
import dynamic from "next/dynamic";
import Truncate from "react-truncate-inside";
import { getRatio, formatNumber } from "@/components/util";
import Container from "react-bootstrap/Container";
import { Button, Card, Carousel, Col, Row, Spinner } from "react-bootstrap";
import { YAxis, Tooltip, Legend, AreaChart, Area } from "recharts";
import {
  getAdminRPCURL,
  configPath,
  ConsensusInfo,
  consensusInfoPath,
  Logs,
  logsPath,
  peerBookPath,
  PeerInfo,
  peerInfoPath,
  Resource,
} from "@/components/api";

const LazyLog = dynamic(() => import("react-lazylog").then((mod) => mod.LazyLog), { ssr: false });

// Dashboard() is the main component of this file
export default function Dashboard() {
  const [state, setState] = useState({
    logs: "retrieving logs...",
    pauseLogs: false,
    resource: [],
    consensusInfo: {},
    peerInfo: {},
  });
  
  // queryAPI() executes the page api calls
  function queryAPI() {
    const promises = [ConsensusInfo(), PeerInfo(), Resource()];
    if (!state.pauseLogs) promises.push(Logs());

    Promise.allSettled(promises).then(([consensusInfo, peerInfo, resource, logs]) => {
      consensusInfo = consensusInfo.status === "fulfilled" ? consensusInfo.value : {};
      peerInfo = peerInfo.status === "fulfilled" ? peerInfo.value : {};
      resource = resource.status === "fulfilled" ? resource.value : {};
      logs = logs.status === "fulfilled" ? logs.value : {};

      setState((prevState) => {
        const updatedResource =
          prevState.resource.length >= 30
            ? [...prevState.resource.slice(1), resource]
            : [...prevState.resource, resource];

        return {
          ...prevState,
          consensusInfo,
          peerInfo,
          resource: updatedResource,
          ...(prevState.pauseLogs ? {} : { logs: logs.toString() }),
        };
      });
    });
  }

  // getRoundProgress() converts the round to a percentage to represent progress
  function getRoundProgress(consensusInfo) {
    const progressMap = {
      ELECTION: 0,
      "ELECTION-VOTE": 1,
      PROPOSE: 2,
      "PROPOSE-VOTE": 3,
      PRECOMMIT: 4,
      "PRECOMMIT-VOTE": 5,
      COMMIT: 6,
      "COMMIT-PROCESS": 7,
    };
    return (progressMap[consensusInfo.view.phase] / 8) * 100;
  }

  // execute every second
  useEffect(() => {
    const interval = setInterval(queryAPI, 1000);
    return () => clearInterval(interval);
  }, []);
  // if loading
  if (!state.consensusInfo.view || !state.peerInfo.id) {
    queryAPI();
    return <Spinner id="spinner" />;
  }
  let inPeer = Number(state.peerInfo.numInbound),
    ouPeer = Number(state.peerInfo.numOutbound),
    v = state.consensusInfo.view;
  inPeer = inPeer ? inPeer : 0;
  ouPeer = ouPeer ? ouPeer : 0;
  const ioRatio = getRatio(inPeer, ouPeer),
    carouselItems = [
      {
        slides: [
          {
            title: state.consensusInfo.syncing ? "SYNCING" : "SYNCED",
            dT: "H: " + formatNumber(v.height, false) + ", R: " + v.round + ", P: " + v.phase,
            d1: "PROP: " + (state.consensusInfo.proposer === "" ? "UNDECIDED" : state.consensusInfo.proposer),
            d2: "BLK: " + (state.consensusInfo.blockHash === "" ? "WAITING" : state.consensusInfo.blockHash),
            d3: state.consensusInfo.status,
          },
          {
            title: "ROUND PROGRESS: " + getRoundProgress(state.consensusInfo) + "%",
            dT: "ADDRESS: " + state.consensusInfo.address,
            d1: "",
            d2: "",
            d3: "",
          },
        ],
        btnSlides: [
          { url: getAdminRPCURL() + consensusInfoPath, title: "QUORUM" },
          { url: getAdminRPCURL() + configPath, title: "CONFIG" },
          { url: getAdminRPCURL() + logsPath, title: "LOGGER" },
        ],
      },
      {
        slides: [
          {
            title: "TOTAL PEERS: " + (state.peerInfo.numPeers == null ? "0" : state.peerInfo.numPeers),
            dT: "INBOUND: " + inPeer + ", OUTBOUND: " + ouPeer,
            d1: "ID: " + state.peerInfo.id.public_key,
            d2:
              "NET ADDR: " +
              (state.peerInfo.id.net_address ? state.peerInfo.id.net_address : "External Address Not Set"),
            d3: "I / O RATIO " + (ioRatio ? ioRatio : "0:0"),
          },
        ],
        btnSlides: [
          { url: getAdminRPCURL() + peerBookPath, title: "PEER BOOK" },
          { url: getAdminRPCURL() + peerInfoPath, title: "PEER INFO" },
        ],
      },
    ];

  // renderButtonCarouselItem() generates the button for the carousel
  function renderButtonCarouselItem(props) {
    return (
      <Carousel.Item>
        <Card className="carousel-item-container">
          <Card.Body>
            <Card.Title>EXPLORE RAW JSON</Card.Title>
            <div>
              {props.map((item, index) => (
                <Button
                  key={index}
                  className="carousel-btn"
                  variant="outline-secondary"
                  onClick={() => window.open(item.url, "_blank")}
                >
                  {item.title}
                </Button>
              ))}
            </div>
          </Card.Body>
        </Card>
      </Carousel.Item>
    );
  }
  
  // return the dashboard rendering
  return (
    <div className="content-container" id="dashboard-container">
      <Container id="dashboard-inner" fluid>
        <Row>
          {carouselItems.map((k, i) => (
            <Col key={i}>
              <Carousel slide={false} interval={null} className="carousel">
                {k.slides.map((k, i) => (
                  <Carousel.Item key={i}>
                    <Card className="carousel-item-container">
                      <Card.Body>
                        <Card.Title className="carousel-item-title">
                          <span>{k.title}</span>
                        </Card.Title>
                        <p id="carousel-item-detail-title" className="carousel-item-detail">
                          {<Truncate text={k.dT} />}
                        </p>
                        <p className="carousel-item-detail">
                          <Truncate text={k.d1} />
                        </p>
                        <p className="carousel-item-detail">
                          <Truncate text={k.d2} />
                        </p>
                        <p>{k.d3}</p>
                      </Card.Body>
                    </Card>
                  </Carousel.Item>
                ))}
                {renderButtonCarouselItem(k.btnSlides)}
              </Carousel>
            </Col>
          ))}
        </Row>
      </Container>
      <hr id="dashboard-hr" />
      <div onClick={() => setState({ ...state, pauseLogs: !state.pauseLogs })} className="logs-button-container">
        <img
          className="logs-button"
          alt="play-pause-btn"
          src={state.pauseLogs ? "./unpause.png" : "./pause.png"}
        />
      </div>
      <LazyLog enableSearch={true} id="lazy-log" text={state.logs.replace("\n", "")} />
      <Container id="charts-container">
        {[
          [
            { yax: "PROCESS", n1: "CPU %", d1: "process.usedCPUPercent", n2: "RAM %", d2: "process.usedMemoryPercent" },
            { yax: "SYSTEM", n1: "CPU %", d1: "system.usedCPUPercent", n2: "RAM %", d2: "system.usedRAMPercent" },
          ],
          [
            { yax: "DISK", n1: "Disk %", d1: "system.usedDiskPercent", n2: "" },
            {
              yax: "IN OUT",
              removeTick: true,
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
        ].map((k, i) => (
          <Row key={i}>
            {[...Array(2)].map((_, i) => {
              let line2 =
                k[i].n2 === "" ? (
                  <></>
                ) : (
                  <Area
                    name={k[i].n2}
                    type="monotone"
                    dataKey={k[i].d2}
                    stroke="#848484"
                    fillOpacity={1}
                    fill="url(#cpu)"
                  />
                );
              return (
                <Col>
                  <AreaChart
                    className="area-chart"
                    width={600}
                    height={250}
                    data={state.resource}
                    margin={{ top: 40, right: 40 }}
                  >
                    <YAxis tick={!k[i].removeTick} tickCount={1} label={{ value: k[i].yax, angle: -90 }} />
                    <Area
                      name={k[i].n1}
                      type="monotone"
                      dataKey={k[i].d1}
                      stroke="#eeeeee"
                      fillOpacity={1}
                      fill="url(#ram)"
                    />
                    {line2}
                    <Tooltip contentStyle={{ backgroundColor: "#222222" }} />
                    <Legend />
                  </AreaChart>
                </Col>
              );
            })}
          </Row>
        ))}
      </Container>
      <div style={{ height: "50px", width: "100%" }} />
    </div>
  );
}
