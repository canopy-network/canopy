import Container from 'react-bootstrap/Container';
import Navbar from 'react-bootstrap/Navbar';
import {Form} from "react-bootstrap";
import {convertIfNumber} from "@/components/util";

function Navigation({openModal}) {
    let q = "", urls = {discord: "https://discord.com", x: "https://x.com"};
    return <>
        <Navbar sticky="top" data-bs-theme="light" className="nav-bar">
            <Container>
                <Navbar.Brand className="nav-bar-brand">
                    <span className="nav-bar-brand-highlight">scan</span>opy explorer
                </Navbar.Brand>
                <div className="nav-bar-center">
                    <Form onSubmit={() => openModal(convertIfNumber(q), 0)}>
                        <Form.Control type="search" className="nav-bar-search me-2"
                                      placeholder="search by address, hash, or height"
                                      onChange={(e) => {
                                          q = e.target.value
                                      }}
                        />
                    </Form>
                </div>
                <a href={urls.discord}>
                    <div id="nav-social-icon1" className="nav-social-icon justify-content-end"/>
                </a>
                <a href={urls.x}>
                    <div id="nav-social-icon2" className="nav-social-icon justify-content-end"/>
                </a>
            </Container>
        </Navbar>
    </>;
}

export default Navigation;