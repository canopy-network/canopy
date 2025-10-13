<img src="https://github.com/user-attachments/assets/b8d6f342-c18b-492e-b87f-06755f775c5f" alt="Canopy Logo" width="500"/>

_Official golang implementation of the Canopy Network Protocol_

[![GoDoc](https://img.shields.io/badge/godoc-reference-white.svg)](https://godoc.org/github.com/canopy-network/canopy)
[![Getting Started](https://img.shields.io/badge/getting%20started-guide-white)](https://canopynetwork.org)
[![Go Version](https://img.shields.io/badge/golang-v1.21-white.svg)](https://golang.org)
[![Next.js Version](https://img.shields.io/badge/next%20js-v14.2.3-white.svg)](https://nextjs.org/)


# Overview

[![License](https://img.shields.io/badge/License-MIT-white.svg)](https://opensource.org/licenses/MIT)
[![Testing](https://img.shields.io/badge/testing-docker%20compose-white)](https://docs.docker.com/compose/)
[![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos-white.svg)](https://github.com/canopy-network/canopy/releases)
[![Status](https://img.shields.io/badge/status-alphanet-white)](https://docs.docker.com/compose/)

### ⫸ **Welcome to the Network that Powers the Peer-to-Peer Launchpad for New Chains**

Built on a recursive architecture, chains bootstrap each other into independence —  
forming an `unstoppable` web of utility and security. 

**Here you'll find:**

➪ A recursive framework to build blockchains.

➪ The seed chain that started the recursive cycle.

For more information on the Canopy Network Protocol visit [https://canopynetwork.org](https://canopynetwork.org)

## Network Status

⪢ Canopy is in `Betanet` 🚀 ➝ learn more about the [road-to-mainnet](https://www.canopynetwork.org/learn-more/road-to-mainnet)

## Protocol Documentation

➪ Check out the Canopy Network wiki:  [https://canopy-network.gitbook.io/docs](https://canopy-network.gitbook.io/docs)

## Repository Documentation

Welcome to the Canopy Network reference implementation. This repository can be well understood reading about the core modules:

- [Controller](controller/README.md): Coordinates communication between all the major parts of the Canopy blockchain, like a central hub or "bus" that connects the system together.
- [Finite State Machine (FSM)](fsm/README.md): Defines the logic for how transactions change the blockchain's state — it decides what’s valid and how state transitions happen from one block to the next.
- [Byzantine Fault Tolerant (BFT) Consensus](bft/README.md): A consensus mechanism that allows the network to agree on new blocks even if some nodes are unreliable or malicious.
- [Peer-to-Peer Networking](p2p/README.md): A secure and encrypted communication system that lets nodes talk directly to each other without needing a central server.
- [Persistence](store/README.md): Manages the blockchain’s storage — it saves the current state (ledger), indexes past transactions, and ensures fast and reliable data verification.

## How to Run It

➪ To run the Canopy binary, use the following commands:

```bash
make build/canopy-full
canopy start
```

## How to Run It with 🐳 Docker

➪ To run a Canopy `Localnet` in a *containerized* environment, use the following commands:
```bash
make docker/build
make docker/up-fast
make docker/logs

or simply

make docker/up && make docker/logs
```

## Running Tests

➪ To run Canopy unit tests, use the Go testing tools:

```bash
make test
```

## How to Contribute

➪ Canopy is an open-source project, and we welcome contributions from the community. Here's how to get involved:

1. **Fork** the repository and clone it locally.
2. **Code** your improvements or fixes.
3. **Submit a Pull Request** (PR) for review.

➣ Please follow these [guidelines](CONTRIBUTING.md) to maintain high-quality contributions:

### High Impact or Architectural Changes

➪ Before making large changes, discuss them with the Canopy team on [Discord](https://discord.gg/pNcSJj7Wdh) to ensure alignment.

### Coding Style

- Code must adhere to official Go formatting (use [`gofmt`](https://golang.org/cmd/gofmt)).
- (Optional) Use [EditorConfig](https://editorconfig.org) for consistent formatting.
- All code should follow Go documentation/commentary guidelines.
- PRs should be opened against the `development` branch.

[![Pre-Release](https://img.shields.io/github/release-pre/canopy-network/canopy.svg)](https://github.com/canopy-network/canopy/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/canopy-network/canopy)](https://goreportcard.com/report/github.com/canopy-network/canopy)
[![Contributors](https://img.shields.io/github/contributors/canopy-network/canopy.svg)](https://github.com/canopy-network/canopy/pulse)
[![Last Commit](https://img.shields.io/github/last-commit/canopy-network/canopy.svg)](https://github.com/canopy-network/canopy/pulse)

## Contact

[![Twitter](https://img.shields.io/twitter/url/http/shields.io.svg?style=social)](https://x.com/CNPYNetwork)
[![Discord](https://img.shields.io/badge/discord-online-blue.svg)](https://discord.gg/pNcSJj7Wdh)

> Code written by hand with ❤️ by the team, documentation with the help of AI for some of the ESL team
