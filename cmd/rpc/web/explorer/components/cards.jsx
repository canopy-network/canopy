import {Card, Col, Row} from "react-bootstrap";
import Truncate from 'react-truncate-inside';
import {addDate, convertBytes, convertNumber, convertTime} from "@/components/util";

// TODO: change this images to #82DCA8 so they are the same as the letters 
const cardImages = ["./block-filled.png", "./chart-up.png", "./transaction-filled.png", "./lock-filled.png"]
const cardTitles = ["Latest Block", "Supply", "Transactions", "Validators"]

// getCardHeader() returns the header information for the card
function getCardHeader(props, idx) {
    const blks = props.blocks
    switch (idx) {
        case 0:
            return convertNumber(blks.results[0].block_header.height)
        case 1:
            return convertNumber(props.supply.total)
        case 2:
            if (blks.results[0].block_header.num_txs == null) {
                return "+0"
            }
            return "+" + convertNumber(blks.results[0].block_header.num_txs)
        case 3:
            let totalStake = 0
            props.canopyCommittee.results.forEach(function (validator) {
                totalStake += Number(validator.staked_amount)
            })
            return <>{convertNumber(totalStake)}<span style={{fontSize: "14px"}}>{" stake"}</span></>
    }
}

// getCardSubHeader() returns the sub header of the card (right below the header)
function getCardSubHeader(props, idx) {
    const v = props.blocks
    switch (idx) {
        case 0:
            return convertTime(v.results[0].block_header.time)
        case 1:
            return convertNumber(Number(props.supply.total) - Number(props.supply.staked), 1000) + " liquid"
        case 2:
            return "blk size: " + convertBytes(v.results[0].meta.size)
        case 3:
            return props.canopyCommittee.results.length + " unique vals"
    }
}

// getCardRightAligned() returns the data for the right aligned note
function getCardRightAligned(props, idx) {
    const v = props.blocks
    switch (idx) {
        case 0:
            return v.results[0].meta.took
        case 1:
            return convertNumber(props.supply.staked, 1000) + " staked"
        case 2:
            return "block #" + v.results[0].block_header.height
        case 3:
            return "stake threshold " + convertNumber(props.params.Validator.validator_stake_percent_for_subsidized_committee, 1000) + "%"
    }
}

// getCardNote() returns the data for the small text above the footer
function getCardNote(props, idx) {
    const v = props.blocks
    switch (idx) {
        case 0:
            return <div style={{height: "25px", paddingTop: "10px"}}><Truncate text={v.results[0].block_header.hash}/>
            </div>
        case 1:
            return "+" + Number(50) + "/blk"
        case 2:
            return "TOTAL " + convertNumber(v.results[0].block_header.total_txs)
        case 3:
            return "MaxStake: " + convertNumber(props.canopyCommittee.results[0].staked_amount, 1000)
        default:
            return "?"
    }
}

// getCardFooter() returns the data for the footer of the card
function getCardFooter(props, idx) {
    const v = props.blocks
    switch (idx) {
        case 0:
            return "Next block: " + addDate(v.results[0].block_header.time, v.results[0].meta.took)
        case 1:
            let s = "DAO pool supply: "
            if (props.pool != null) {
                return s + convertNumber(props.pool.amount, 1000)
            }
            return s
        case 2:
            let totalFee = 0, txs = v.results[0].transactions;
            if (txs == null || txs.length === 0) {
                return "Average fee in last blk: 0"
            }
            txs.forEach(function (tx) {
                totalFee += Number(tx.transaction.fee)
            })
            return "Average fee in last blk: " + convertNumber(totalFee / txs.length, 1000000)
        case 3:
            let totalStake = 0
            props.canopyCommittee.results.forEach(function (validator) {
                totalStake += Number(validator.staked_amount)
            })
            return (totalStake / props.supply.staked * 100).toFixed(1) + "% in canopy committee"
    }
}

// getCardOnClick() returns the callback function when a certain card is clicked
function getCardOnClick(props, index) {
    if (index === 0) {
        return () => props.openModal(0)
    } else {
        if (index === 1) {
            return () => props.selectTable(7, 0)
        } else if (index === 2) {
            return () => props.selectTable(1, 0)
        }
        return () => props.selectTable(index + 1, 0)
    }
}

// Cards() returns the main component
export default function Cards(props) {
    const cardData = props.state.cardData
    return (<Row sm={1} md={2} lg={4} className="g-4">
        {Array.from({length: 4}, (_, idx) => {
            return <Col key={idx}>
                <Card className="text-center">
                    <Card.Body className="card-body" onClick={getCardOnClick(props, idx)}>
                        <div className="card-image" style={{backgroundImage: "url(" + cardImages[idx] + ")"}}/>
                        <Card.Title className="card-title">{cardTitles[idx]}</Card.Title>
                        <h5>{getCardHeader(cardData, idx)}</h5>
                        <Card.Text>
                            <span>{getCardSubHeader(cardData, idx)}</span>
                            <span className="card-info-3">{getCardRightAligned(cardData, idx)}</span>
                            <br/>
                            <span className="card-info-4">{getCardNote(cardData, idx)}</span>
                        </Card.Text>
                        <Card.Footer className="card-footer">{getCardFooter(cardData, idx)}</Card.Footer>
                    </Card.Body>
                </Card>
            </Col>
        })}
    </Row>);
}