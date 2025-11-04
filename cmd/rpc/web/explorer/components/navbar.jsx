import { convertIfNumber } from "@/components/util";
import { useState } from "react";
import { Form } from "react-bootstrap";
import Container from "react-bootstrap/Container";
import Navbar from "react-bootstrap/Navbar";

export default function Navigation({ openModal }) {
  let [query, setQuery] = useState("");
  let q = "";
  let urls = { discord: "https://discord.gg/pNcSJj7Wdh", x: "https://x.com/CNPYNetwork" };

  return (
    <>
      <Navbar sticky="top" data-bs-theme="light" className="nav-bar">
        <Container>
          <Navbar.Brand className="nav-bar-brand">
            <img src="./scanopy.png" alt="Scanopy Logo" className="nav-bar-logo" />
          </Navbar.Brand>
          <div className="nav-bar-center">
            <Form
              onSubmit={(e) => {
                e.preventDefault();
                openModal(convertIfNumber(query), 0);
              }}
            >
              <Form.Control
                type="search"
                className="main-input nav-bar-search me-2"
                placeholder="search by address, hash, or height"
                style={{ backgroundImage: 'url("./search.png")' }}
                onChange={(e) => {
                  setQuery(e.target.value);
                }}
              />
            </Form>
          </div>
          <a href={urls.discord}>
            <div 
              id="nav-social-icon1" 
              className="nav-social-icon justify-content-end"
              style={{ backgroundImage: 'url("./discord-filled.png")' }}
            />
          </a>
          <a href={urls.x}>
            <div 
              id="nav-social-icon2" 
              className="nav-social-icon justify-content-end"
              style={{ backgroundImage: 'url("./twitter.png")' }}
            />
          </a>
        </Container>
      </Navbar>
    </>
  );
}
