import React from "react";
import { MDBCollapse } from "mdb-react-ui-kit";
import { withTooltip } from "@/components/util";

const sidebarIcons = [
  { src: "./block.png", label: "Blocks" },
  { src: "./transaction.png", label: "Transactions" },
  { src: "./pending.png", label: "Pending" },
  { src: "./account.png", label: "Accounts" },
  { src: "./validator.png", label: "Validators" },
  { src: "./gov.png", label: "Governance" },
  { src: "./swaps.png", label: "Swaps" },
  { src: "./supply.png", label: "Supply" },
  { src: "./batch.png", label: "Dex Batches" },
];

const urls = {
  docs: "https://canopy-network.gitbook.io/docs",
  github: "https://github.com/canopy-network",
};

export default function Sidebar({ selectTable }) {
  return (
    <MDBCollapse className="d-lg-block sidebar">
      <div className="sidebar-list">
        {sidebarIcons.map((icon, i) =>
          withTooltip(
            <div onClick={() => selectTable(i, 0)} className="sidebar-icon-container">
              <div className="sidebar-icon" style={{ backgroundImage: `url(${icon.src})` }} />
            </div>,
            icon.label,
            i,
          ),
        )}
        <div className="sidebar-icon-container">
          <a href={urls.docs}>
            {withTooltip(<div className="sidebar-icon" style={{ backgroundImage: "url(./explore.png)" }} />, "Explore")}
          </a>
        </div>
      </div>
      <a href={urls.github}>
        <div id="sidebar-social" style={{ backgroundImage: "url(./github.png)" }} />
      </a>
    </MDBCollapse>
  );
}
