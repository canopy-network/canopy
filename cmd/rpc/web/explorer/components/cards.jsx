import { addDate, convertBytes, convertNumber, convertTime } from "@/components/util";
import { Card, Col, Row } from "react-bootstrap";
import Truncate from "react-truncate-inside";

const cardImages = [
  <svg id="svg" width="400" height="400" viewBox="0, 0, 400,400">
    <g id="svgg">
      <path
        id="path0"
        d="M168.944 38.203 C 157.364 45.791,146.927 52.000,145.752 52.000 C 144.577 52.000,133.767 58.300,121.730 66.000 C 109.693 73.700,99.070 80.000,98.123 80.000 C 97.176 80.000,86.300 86.085,73.954 93.522 L 51.508 107.043 64.754 115.057 C 105.074 139.450,176.847 180.000,179.703 180.000 C 181.528 180.000,185.261 182.700,188.000 186.000 C 190.739 189.300,195.320 192.000,198.180 192.000 C 202.984 192.000,338.339 115.094,344.929 108.620 C 348.962 104.657,208.907 23.951,198.433 24.203 C 193.795 24.314,180.525 30.614,168.944 38.203 M48.000 205.490 L 48.000 291.237 120.000 333.344 L 192.000 375.452 192.000 288.606 L 192.000 201.761 139.000 171.227 C 109.850 154.434,77.450 135.980,67.000 130.218 L 48.000 119.743 48.000 205.490 M337.935 127.230 C 331.300 130.953,299.350 149.241,266.935 167.869 L 208.000 201.738 208.000 288.595 L 208.000 375.452 280.000 333.344 L 352.000 291.237 352.000 205.618 C 352.000 158.528,351.550 120.103,351.000 120.230 C 350.450 120.356,344.571 123.506,337.935 127.230 "
        stroke="none"
        fillRule="evenodd"
      ></path>
    </g>
  </svg>,
  <svg id="svg" width="400" height="400" viewBox="0, 0, 400,400">
    <g id="svgg">
      <path
        id="path0"
        d="M264.190 118.273 C 256.926 137.203,263.939 147.851,285.088 150.000 L 305.592 152.083 287.171 167.542 C 277.040 176.044,257.648 193.207,244.080 205.682 L 219.410 228.365 181.580 195.511 C 129.537 150.315,130.947 150.132,71.494 209.754 C 27.805 253.568,24.472 258.043,25.728 271.213 C 28.460 299.846,49.581 292.685,94.274 247.971 L 132.298 209.930 170.737 242.465 C 211.958 277.354,224.046 281.644,239.437 266.847 C 260.592 246.508,332.693 183.333,334.750 183.333 C 336.012 183.333,337.615 192.240,338.313 203.125 L 339.583 222.917 356.250 222.917 L 372.917 222.917 372.917 168.750 L 372.917 114.583 319.709 113.417 C 276.377 112.466,266.072 113.368,264.190 118.273 "
        stroke="none"
        fillRule="evenodd"
      ></path>
    </g>
  </svg>,
  <svg id="svg" width="400" height="400" viewBox="0, 0, 400,400">
    <g id="svgg">
      <path
        id="path0"
        d="M143.663 40.611 C 29.079 81.483,-5.008 232.939,81.007 319.006 C 145.191 383.228,254.783 383.228,319.006 319.006 C 443.119 194.892,308.838 -18.308,143.663 40.611 M264.583 145.833 L 301.910 183.333 200.955 183.333 L 100.000 183.333 100.000 166.667 L 100.000 150.000 162.500 150.000 L 225.000 150.000 225.000 129.167 C 225.000 103.480,220.570 101.615,264.583 145.833 M300.000 233.333 L 300.000 250.000 237.718 250.000 L 175.435 250.000 174.176 270.958 L 172.917 291.915 135.504 254.291 L 98.090 216.667 199.045 216.667 L 300.000 216.667 300.000 233.333 "
        stroke="none"
        fillRule="evenodd"
      ></path>
    </g>
  </svg>,
  <svg id="svg" width="400" height="400" viewBox="0, 0, 400,400">
    <g id="svgg">
      <path
        id="path0"
        d="M154.729 43.946 C 123.793 63.565,112.000 86.533,112.000 127.166 C 112.000 154.556,110.494 162.880,103.853 172.207 C 58.547 235.833,77.836 322.306,144.799 355.768 C 254.190 410.431,365.685 289.797,303.000 184.599 C 289.024 161.144,288.000 157.249,288.000 127.515 C 288.000 50.139,216.078 5.040,154.729 43.946 M237.491 61.726 C 261.220 78.464,266.240 87.955,270.287 123.729 L 273.047 148.117 252.472 138.585 C 222.187 124.553,177.723 124.309,148.499 138.013 L 126.998 148.096 129.338 124.162 C 132.096 95.939,139.134 80.806,156.165 66.475 C 178.853 47.384,214.231 45.318,237.491 61.726 M215.200 232.800 C 221.809 239.409,221.232 257.998,214.237 263.803 C 211.068 266.433,207.918 273.388,207.237 279.259 C 205.410 295.019,192.000 296.361,192.000 280.783 C 192.000 274.291,189.300 266.739,186.000 264.000 C 171.850 252.256,181.283 228.000,200.000 228.000 C 205.720 228.000,212.560 230.160,215.200 232.800 "
        stroke="none"
        fillRule="evenodd"
      ></path>
    </g>
  </svg>,
];
const cardTitles = ["Latest Block", "Supply", "Transactions", "Validators"];

// getCardHeader() returns the header information for the card
function getCardHeader(props, idx) {
  const blks = props.blocks;
  if (blks.results.length === 0) {
    return "Loading";
  }
  switch (idx) {
    case 0:
      return convertNumber(blks.results[0].blockHeader.height);
    case 1:
      return convertNumber(props.supply.total, 1000, true);
    case 2:
      if (blks.results[0].blockHeader.numTxs == null) {
        return "+0";
      }
      return "+" + convertNumber(blks.results[0].blockHeader.numTxs);
    case 3:
      let totalStake = 0;
      if (!props.canopyCommittee.results) {
        return 0;
      }
      props.canopyCommittee.results.forEach(function (validator) {
        totalStake += Number(validator.stakedAmount);
      });
      return (
        <>
          {convertNumber(totalStake, 1000, true)}
          <span style={{ fontSize: "14px" }}>{" stake"}</span>
        </>
      );
  }
}

// getCardSubHeader() returns the sub header of the card (right below the header)
function getCardSubHeader(props, consensusDuration, idx) {
  const v = props.blocks;
  if (v.results.length === 0) {
    return "Loading";
  }
  switch (idx) {
    case 0:
      return convertTime(v.results[0].blockHeader.time - consensusDuration);
    case 1:
      return convertNumber(Number(props.supply.total) - Number(props.supply.staked), 1000, true) + " liquid";
    case 2:
      return "blk size: " + convertBytes(v.results[0].meta.size);
    case 3:
      if (!props.canopyCommittee.results) {
        return 0 + " unique vals";
      }
      return props.canopyCommittee.results.length + " unique vals";
  }
}

// getCardRightAligned() returns the data for the right aligned note
function getCardRightAligned(props, idx) {
  const v = props.blocks;
  if (v.results.length === 0) {
    return "Loading";
  }
  switch (idx) {
    case 0:
      return v.results[0].meta.took;
    case 1:
      return convertNumber(props.supply.staked, 1000, true) + " staked";
    case 2:
      return "block #" + v.results[0].blockHeader.height;
    case 3:
      return "stake threshold " + convertNumber(props.params.validator.stakePercentForSubsidizedCommittee, 1000) + "%";
  }
}

// getCardNote() returns the data for the small text above the footer
function getCardNote(props, idx) {
  const v = props.blocks;
  if (v.results.length === 0) {
    return "Loading";
  }
  switch (idx) {
    case 0:
      return <Truncate className="d-inline" text={v.results[0].blockHeader.hash} />;
    case 1:
      return "+" + Number(props.ecoParams.MintPerBlock/1000000) + "/blk";
    case 2:
      return "TOTAL " + convertNumber(v.results[0].blockHeader.totalTxs);
    case 3:
      if (!props.canopyCommittee.results) {
        return "MaxStake: " + 0;
      }
      return "MaxStake: " + convertNumber(props.canopyCommittee.results[0].stakedAmount, 1000, true);
    default:
      return "?";
  }
}

// getCardFooter() returns the data for the footer of the card
function getCardFooter(props, consensusDuration, idx) {
  const v = props.blocks;
  if (v.results.length === 0) {
    return "Loading";
  }
  switch (idx) {
    case 0:
      return "Next block: " + addDate(v.results[0].blockHeader.time, consensusDuration);
    case 1:
      let s = "DAO pool supply: ";
      if (props.pool != null) {
        return s + convertNumber(props.pool.amount, 1000, true);
      }
      return s;
    case 2:
      let totalFee = 0,
        txs = v.results[0].transactions;
      if (txs == null || txs.length === 0) {
        return "Average fee in last blk: 0";
      }
      txs.forEach(function (tx) {
        let fee = Number(tx.transaction.fee);
        totalFee += !isNaN(fee) ? fee : 0;
      });
      let txWithFee = txs.filter((tx) => tx.transaction.fee != null);

      return `Average fee in last blk: ${totalFee > 0 ? convertNumber(totalFee / txWithFee.length, 1000000) : 0}`;
    case 3:
      let totalStake = 0;
      if (!props.canopyCommittee.results) {
        return 0 + "% in validator set";
      }
      props.canopyCommittee.results.forEach(function (validator) {
        totalStake += Number(validator.stakedAmount);
      });
      return ((totalStake / props.supply.staked) * 100).toFixed(1) + "% in validator set";
  }
}

// getCardOnClick() returns the callback function when a certain card is clicked
function getCardOnClick(props, index) {
  if (index === 0) {
    return () => props.openModal(0);
  } else {
    if (index === 1) {
      return () => props.selectTable(7, 0);
    } else if (index === 2) {
      return () => props.selectTable(1, 0);
    }
    return () => props.selectTable(index + 1, 0);
  }
}

// Cards() returns the main component
export default function Cards(props) {
  const cardData = props.state.cardData;
  const consensusDuration = props.state.consensusDuration;
  return (
    <Row sm={1} md={2} lg={4} className="g-4">
      {Array.from({ length: 4 }, (_, idx) => {
        return (
          <Col key={idx}>
            <Card className="text-center">
              <Card.Body className="card-body" onClick={getCardOnClick(props, idx)}>
                <div className="card-image">{cardImages[idx]}</div>
                <Card.Title className="card-title">{cardTitles[idx]}</Card.Title>
                <h5>{getCardHeader(cardData, idx)}</h5>
                <div className="d-flex justify-content-between mb-1">
                  <Card.Text className="card-info-2 mb-2">
                    {getCardSubHeader(cardData, consensusDuration, idx)}
                  </Card.Text>
                  <Card.Text className="card-info-3">{getCardRightAligned(cardData, idx)}</Card.Text>
                </div>
                <div className="card-info-4 mb-3">{getCardNote(cardData, idx)}</div>
                <Card.Footer className="card-footer">{getCardFooter(cardData, consensusDuration, idx)}</Card.Footer>
              </Card.Body>
            </Card>
          </Col>
        );
      })}
    </Row>
  );
}
