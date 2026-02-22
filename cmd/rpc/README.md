# JSON-RPC API Methods

<hr/>

## RPC Routes Reference
- /v1/
- /v1/tx
- /v1/query/height
- /v1/query/account
- /v1/query/accounts
- /v1/query/pool
- /v1/query/pools
- /v1/query/validator
- /v1/query/validators
- /v1/query/committee
- /v1/query/committee-data
- /v1/query/committees-data
- /v1/query/subsidized-committees
- /v1/query/retired-committees
- /v1/query/non-signers
- /v1/query/params
- /v1/query/supply
- /v1/query/fee-params
- /v1/query/gov-params
- /v1/query/con-params
- /v1/query/val-params
- /v1/query/eco-params
- /v1/query/state
- /v1/query/state-diff
- /v1/query/state-diff
- /v1/query/cert-by-height
- /v1/query/block-by-height
- /v1/query/blocks
- /v1/query/block-by-hash
- /v1/query/txs-by-height
- /v1/query/txs-by-sender
- /v1/query/txs-by-rec
- /v1/query/tx-by-hash
- /v1/query/events-by-height
- /v1/query/events-by-address
- /v1/query/events-by-chain
- /v1/query/order
- /v1/query/orders
- /v1/query/dex-batch
- /v1/query/next-dex-batch
- /v1/query/dex-price
- /v1/query/last-proposers
- /v1/query/valid-double-signer
- /v1/query/double-signers
- /v1/query/minimum-evidence-height
- /v1/query/lottery
- /v1/query/pending
- /v1/query/failed-txs
- /v1/gov/proposals
- /v1/gov/poll
- /v1/query/root-chain-info
- /v1/query/validator-set
- /v1/query/checkpoint
- /v1/subscribe-rc-info
- /debug/blocked
- /debug/heap
- /debug/cpu
- /debug/routine
- /v1/eth
- /v1/admin/keystore
- /v1/admin/keystore-new-key
- /v1/admin/keystore-import
- /v1/admin/keystore-import-raw
- /v1/admin/keystore-delete
- /v1/admin/keystore-get
- /v1/admin/tx-send
- /v1/admin/tx-stake
- /v1/admin/tx-edit-stake
- /v1/admin/tx-unstake
- /v1/admin/tx-pause
- /v1/admin/tx-unpause
- /v1/admin/tx-change-param
- /v1/admin/tx-dao-transfer
- /v1/admin/tx-create-order
- /v1/admin/tx-edit-order
- /v1/admin/tx-delete-order
- /v1/admin/tx-dex-limit-order
- /v1/admin/tx-dex-liquidity-deposit
- /v1/admin/tx-dex-liquidity-withdraw
- /v1/admin/tx-lock-order
- /v1/admin/tx-close-order
- /v1/admin/subsidy
- /v1/admin/tx-start-poll
- /v1/admin/tx-vote-poll
- /v1/admin/resource-usage
- /v1/admin/peer-info
- /v1/admin/consensus-info
- /v1/admin/peer-book
- /v1/admin/config
- /v1/admin/log
- /v1/gov/add-vote
- /v1/gov/del-vote

# Standard

## Version

**Route**: `/v1/`

**Description**: responds with the software version

**HTTP Method**: `GET`

**Request**: `null`

**Response**: `string` - the protocol version

**Example**:

```
$ curl -H "Content-Type: application/json" -X GET --data '{}' localhost:50002/v1/

> "0.0.0-alpha"
```

## Tx

**Route**: `/v1/tx`

**Description**: submits a transaction

**HTTP Method**: `POST`

**Request**:

- **type**: `string` - the message name (`"send"`, `"stake"`, `"edit-stake"`, ...)
- **msg**: `Message` - the payload of the transaction (`MessageSend`, `MessageStake`, ...)
- **signature**: `Signature` - the authorization (public key and signature) of the transaction
- **time**: `uint64` - the creation time of the transaction for replay protection (unix micro)
- **createdHeight**: `uint64` - the height the transaction was created `+/- 4320` accepted
- **fee**: `uint64`  the network fee for sending the transaction (minimum is parameterized)
- **memo**: `string` - an embedded message in the transaction (optional)
- **networkID**: `uint64` - the unique identifier of the network (`1` for mainnet, `2` for testnet, ...)
- **chainID**: `uint64` - the unique identifier of the committee (`1` for canopy, `2` for canary, ...)

**Response**: `hex string` - the hash of the transaction

**Example**:

```
$ curl -X POST localhost:50002/v1/tx \
  -H "Content-Type: application/json" \
  -d '{
    "type": "send",
    "msg": {
      "fromAddress": "b8bc466953be5f6f31954108f683d2b02b8b7453",
      "toAddress": "08c18a0e3ef3727b42eab9eef51494f8f7f83bd0",
      "amount": 1000000
    },
    "signature": {
      "publicKey": "a5b97c05cb26c9bc118b3e2258f03101a9a427317dccd80abd4f3a82a42afc6c27ae5f63a04aaef180558b0176282f1c78a51af474036316b1c2ae826bf8c23a",
      "signature": "89ce883843a4ebfc763e0fd377718e659063bdfee492f3fff1cfe055b9cbc1f67da580e3ad638c32d520c3db4baf94a0c4f605cb73b0ee021a330a23a93ffe99"
    },
    "time": 1747865418150488,
    "createdHeight": 1,
    "fee": 10000,
    "memo": "hello world",
    "networkID": 1,
    "chainID": 1
  }'

> "25c7216b7523fdfdb60b989b38c4b9d83a546a63029d56f2ce6f2be6bd255aa4"
```

## Height

**Route:** `/v1/query/height`

**Description**: responds with the next block version

**HTTP Method**: `POST`

**Request**: `null`

**Response**:

- **height**: `uint64` - the number of the next block

**Example:**:

```
$ curl -H "Content-Type: application/json" -X POST --data '{}' localhost:50002/v1/query/height

> 9157
```

## Account

**Route:** `/v1/query/account`

**Description**: responds with an account for the specified address

**HTTP Method**: `POST`

**Request**:
- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **address**: `hex string` - the 20 byte identifier of the account

**Response**:

- **address**: `hex string` - the 20 byte identifier
- **amount**: `uint64` - the balance of funds the account has

**Example**:

```
$ curl -X POST localhost:50002/v1/query/account \
  -H "Content-Type: application/json" \
  -d '{
        "address": "0971d5d96f1533479ab1a6472fe0260df6ae732d",
        "height": 1000
      }'

> {
    "address": "0971d5d96f1533479ab1a6472fe0260df6ae732d",
    "amount": 99990000
  }
```

## Accounts

**Route:** `/v1/query/accounts`

**Description**: responds with a list of accounts based on the page parameters

**HTTP Method**: `POST`

**Request**:
- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **perPage**: `int` - the number of elements per page (the default is 10 and max is 5,000)
- **pageNumber**: `int` - the number of the page (the default is 1)

**Response**:
- **perPage**: `int` - the number of elements per page
- **pageNumber**: `int` - the number of the page
- **results**: `array` - the list of result objects
  - **address**: - `hex string` the 20 byte unique identifier of the account
  - **amount**: - `uint64` the balance of funds the account has in micro denomination
- **type**: `string` - the type of results
- **count**: `int` - length of results
- **totalPages**: `int` - number of pages
- **totalCount**: `int` - total number of items that exist in all pages

**Example**:

```
$ curl -X POST localhost:50002/v1/query/accounts \
  -H "Content-Type: application/json" \
  -d '{
        "perPage": 10,
        "page": 1,
        "height": 1000
      }'

> {
    "pageNumber": 1,
    "perPage": 10,
    "results": [
      {
        "address": "180c9d067ce4275612896dc7ce01390329e7f101",
        "amount": 969901
      },
      {
        "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
        "amount": 18239045002
      },
      {
        "address": "551f21e333012027b81701a35023efc88b864975",
        "amount": 130
      }
    ],
    "type": "accounts",
    "count": 3,
    "totalPages": 1,
    "totalCount": 3
  }
```

## Pool

**Route:** `/v1/query/pool`

**Description**: responds with a Pool structure for a specific ID

**HTTP Method**: `POST`

**Request**:
- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **id**: `uint64` - the unique identifier

**Response**:

- **id**: `string` - the unique identifier
- **amount**:`uint64` - the balance of funds the pool has in micro denomination
- **poolPoints**: `array` - a list of pool points
  - **address**: `hex-string` - the recipient address of the points
  - **points**: `uint64` the amount of points owned
- **totalPoolPoints**: `uint64` - the sum of all pool points
**Example**:

```
$ curl -X POST localhost:50002/v1/query/pool \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000
        "id": 1,
      }'

> {
  "id": "1",
  "amount": 969,
  "poolPoints": [
    {
      "address": "551f21e333012027b81701a35023efc88b864975",
      "points": 100
    }
  ],
  "totalPoolPoints": 100
}
```

## Pools

**Route:** `/v1/query/pools`

**Description**: responds with a list of pools based on the page parameters

**HTTP Method**: `POST`

**Request**:
- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **perPage**: `int` - the number of elements per page (the default is 10 and max is 5,000)
- **pageNumber**: `int` - the number of the page (the default is 1)

**Response**:
- **perPage**: `int` - the number of elements per page
- **pageNumber**: `int` - the number of the page
- **results**: `array` - the list of result objects
  - **id**: - `string` the unique identifier of the pool
  - **amount**: - `uint64` the balance of funds the pool has in micro denomination
  - **poolPoints**: `array` - a list of pool points
    - **address**: `hex-string` - the recipient address of the points
    - **points**: `uint64` the amount of points owned
- **totalPoolPoints**: `uint64` - the sum of all pool points
- **type**: `string` - the type of results
- **count**: `int` - length of results
- **totalPages**: `int` - number of pages
- **totalCount**: `int` - total number of items that exist in all pages

**Example**:

```
$ curl -X POST localhost:50002/v1/query/pools \
  -H "Content-Type: application/json" \
  -d '{
        "perPage": 10,
        "page": 1,
        "height": 1000
      }'

> {
    "pageNumber": 1,
    "perPage": 10,
    "results": [
      {
        "id": "1",
        "amount": 969,
        "poolPoints": [
          {
            "address": "551f21e333012027b81701a35023efc88b864975",
            "points": 100
          }
        ],
        "totalPoolPoints": 100
      }
    ],
    "type": "pools",
    "count": 1,
    "totalPages": 1,
    "totalCount": 1
  }
```

## Validator

**Route:** `/v1/query/validator`

**Description**: responds with a Validator structure for a specific address

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **address**: `hex string` - the 20 byte identifier

**Response**:

- **address**: `hex-string` - the 20 byte identifier
- **publicKey**: `hex string` - the unique public identifier of the validator that is used to validate digital signatures
- **stakedAmount**: `uint64` - the locked balance of funds the address has in micro denomination
- **committees**: `[]uint64` - list of chain ids the validator is staked on behalf
- **netAddress**: `url` - the public peer-to-peer address of the validator
- **maxPausedHeight**: `uint64` - the height the validator will be automatically begin unstaking if not unpaused (0 is not paused)
- **unstakingHeight**: `uint64` - the height the validator's locked funds are returned (0 is not unstaking)
- **output**: `hex string` - the 20 byte unique identifier of the account where rewards and locked funds are distributed
- **delegate**: `bool` - is the validator a delegate only
- **compound**: `bool` - is the validator automatically compounding their rewards

**Example**:

```
$ curl -X POST localhost:50002/v1/query/validator \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000
        "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
      }'

> {
      "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
      "publicKey": "b2947db37385bb43c46244cef15f2451a446cea011fc1a2e1d52b1cecc7a50a8924e0e062555793bbd55a91a685017ee",
      "committees": [
        1
      ],
      "netAddress": "tcp://localhost",
      "stakedAmount": 2000,
      "maxPausedHeight": 0,
      "unstakingHeight": 0,
      "output": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
      "delegate": false,
      "compound": false
  }
```

## Validators

**Route:** `/v1/query/validators`

**Description**: responds with a page of filtered validators

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **perPage**: `int` - the number of elements per page (the default is 10 and max is 5,000)
- **pageNumber**: `int` - the number of the page (the default is 1)
- **unstaking**: `int` - only validators are currently unstaking (optional: `0=Off`, `1=MustBe`, `2=Exclude`)
- **paused**: `int` - only validators are currently paused (optional: `0=Off`, `1=MustBe`, `2=Exclude`)
- **delegate**: `int` only validators are set as delegates (optional: `0=Off`, `1=MustBe`, `2=Exclude`)
- **committee**: `uint64` - validators are staked for this chain id (optional: `0=Any`)


**Response**:
- **perPage**: `int` - the number of elements per page
- **pageNumber**: `int` - the number of the page
- **results**: `array` - the list of result objects
  - **address**: `uint64` - the 20 byte identifier
  - **publicKey**: `hex string` - the unique public identifier of the validator that is used to validate digital signatures
  - **stakedAmount**: `uint64` - the locked balance of funds the address has in micro denomination
  - **committees**: `[]uint64` - list of chain ids the validator is staked on behalf
  - **netAddress**: `url` - the public peer-to-peer address of the validator
  - **maxPausedHeight**: `uint64` - the height the validator will be automatically begin unstaking if not unpaused (0 is not paused)
  - **unstakingHeight**: `uint64` - the height the validator's locked funds are returned (0 is not unstaking)
  - **output**: `hex string` - the 20 byte unique identifier of the account where rewards and locked funds are distributed
  - **delegate**: `bool` - is the validator a delegate only
  - **compound**: `bool` - is the validator automatically compounding their rewards
- **type**: `string` - the type of results
- **count**: `int` - length of results
- **totalPages**: `int` - number of pages
- **totalCount**: `int` - total number of items that exist in all pages

**Example**:

```
$ curl -X POST localhost:50002/v1/query/validators \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000
      }'

> {
    "pageNumber": 1,
    "perPage": 10,
    "results": [
      {
        "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
        "publicKey": "b2947db37385bb43c46244cef15f2451a446cea011fc1a2e1d52b1cecc7a50a8924e0e062555793bbd55a91a685017ee",
        "committees": [
          1
        ],
        "netAddress": "tcp://localhost",
        "stakedAmount": 2000,
        "maxPausedHeight": 0,
        "unstakingHeight": 0,
        "output": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
        "delegate": false,
        "compound": false
      },
      {
        "address": "551f21e333012027b81701a35023efc88b864975",
        "publicKey": "904d1f2b856eaab5f44c16d1fd2d77b469b2d080f31fdb78354d021c1ff47ebb44af30a2ce5f389f6bc6d07946aa86d421a05c31c72927678b540d76bf112649",
        "committees": [
          1,
          2,
          3
        ],
        "netAddress": "",
        "stakedAmount": 23344689106,
        "maxPausedHeight": 0,
        "unstakingHeight": 0,
        "output": "551f21e333012027b81701a35023efc88b864975",
        "delegate": true,
        "compound": true
      }
    ],
    "type": "validators",
    "count": 2,
    "totalPages": 1,
    "totalCount": 2
  }
```

## Validator-Set

**Route:** `/v1/query/validator-set`

**Description**: responds with the exact list of non-delegate validators for a specific committee id (non-paginated, currently in consensus)

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **id**: `uint64` – the unique identifier of the committee

**Response**:
- **validatorSet** `array` - the list of result objects
  - **publicKey**: `hex string` - the unique public identifier of the validator that is used to validate digital signatures
  - **votingPower**: `uint64` - the locked balance of funds the address has in micro denomination
  - **netAddress**: `url` - the public peer-to-peer address of the validator

**Example**:

```
$ curl -X POST localhost:50002/v1/query/validator-set \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000,
        "id": 1
      }'

> {
  "validatorSet": [
    {
      "publicKey": "b2947db37385bb43c46244cef15f2451a446cea011fc1a2e1d52b1cecc7a50a8924e0e062555793bbd55a91a685017ee",
      "votingPower": 2000,
      "netAddress": "tcp://localhost"
    }
  ]
}
```

## Committee

**Route:** `/v1/query/committee`

**Description**: responds with a page of non-delegate validators for a specific committee id

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **committee**: `uint64` – the unique identifier of the committee
- **perPage**: `int` - the number of elements per page (the default is 10 and max is 5,000)
- **pageNumber**: `int` - the number of the page (the default is 1)

**Response**:
- **perPage**: `int` - the number of elements per page
- **pageNumber**: `int` - the number of the page
- **results**: `array` - the list of result objects
  - **address**: `uint64` - the 20 byte identifier
  - **publicKey**: `hex string` - the unique public identifier of the validator that is used to validate digital signatures
  - **stakedAmount**: `uint64` - the locked balance of funds the address has in micro denomination
  - **committees**: `[]uint64` - list of chain ids the validator is staked on behalf
  - **netAddress**: `url` - the public peer-to-peer address of the validator
  - **maxPausedHeight**: `uint64` - the height the validator will be automatically begin unstaking if not unpaused (0 is not paused)
  - **unstakingHeight**: `uint64` - the height the validator's locked funds are returned (0 is not unstaking)
  - **output**: `hex string` - the 20 byte unique identifier of the account where rewards and locked funds are distributed
  - **delegate**: `bool` - is the validator a delegate only
  - **compound**: `bool` - is the validator automatically compounding their rewards
- **type**: `string` - the type of results
- **count**: `int` - length of results
- **totalPages**: `int` - number of pages
- **totalCount**: `int` - total number of items that exist in all pages

**Example**:

```
$ curl -X POST localhost:50002/v1/query/committee \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000,
        "committee": 1
      }'

> {
    "pageNumber": 1,
    "perPage": 10,
    "results": [
      {
        "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
        "publicKey": "b2947db37385bb43c46244cef15f2451a446cea011fc1a2e1d52b1cecc7a50a8924e0e062555793bbd55a91a685017ee",
        "committees": [
          1
        ],
        "netAddress": "tcp://localhost",
        "stakedAmount": 2000,
        "maxPausedHeight": 0,
        "unstakingHeight": 0,
        "output": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
        "delegate": false,
        "compound": false
      }
    ],
    "type": "validators",
    "count": 1,
    "totalPages": 1,
    "totalCount": 1
  }
```

## Committee-Data

**Route:** `/v1/query/committee-data`

**Description**: responds with information about the current status of the Committee

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **id**: `uint64` – the unique identifier of the committee

**Response**:
- **chainID**: `uint64` - the unique identifier of the committee
- **lastRootHeightUpdated**: `uint64` - the most recent root-chain height the committee reported in their certificate results transaction
- **lastChainHeightUpdated**: `uint64` - the most recent chain height the committee reported in their certificate results transaction
- **paymentPercents**: `array` - a list of recipients and the percentage of rewards they will receive, distributed at the end of the block
  - **address**: `hex string` - the address where the tokens will be received
  - **percents**: `uint64` - the dilutable share of the committee treasury pool
  - **chainId**: `uint64` - the committee pool from which the payment is distributed
- **numberOfSamples**: `uint64` - the total count of processed Certificate Result Transactions, used to normalize reward percentages accurately

**Example**:

```
$ curl -X POST localhost:50002/v1/query/committee-data \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000,
        "id": 1
      }'

> {
  "chainID": 1,
  "lastRootHeightUpdated": 998,
  "lastChainHeightUpdated": 998,
  "paymentPercents": [
    {
      "address": "551f21e333012027b81701a35023efc88b864975",
      "percents": 10,
      "chainId": 1
    },
    {
      "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
      "percents": 90,
      "chainId": 1
    }
  ],
  "numberOfSamples": 1
}
```

## Committees-Data

**Route:** `/v1/query/committees-data`

**Description**: responds with the master list of subsidized committee data

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)

**Response**:
- **list**: `array` - a list of result objects
  - **chainID**: `uint64` - the unique identifier of the committee
  - **lastRootHeightUpdated**: `uint64` - the most recent root-chain height the committee reported in their certificate results transaction
  - **lastChainHeightUpdated**: `uint64` - the most recent chain height the committee reported in their certificate results transaction
  - **paymentPercents**: `array` - a list of recipients and the percentage of rewards they will receive, distributed at the end of the block
    - **address**: `hex string` - the address where the tokens will be received
    - **percent**: `uint64` - the dilutable share of the committee treasury pool
    - **chainId**: `uint64` - the committee pool from which the payment is distributed
  - **numberOfSamples**: `uint64` - the total count of processed Certificate Result Transactions, used to normalize reward percentages accurately

**Example**:

```
$ curl -X POST localhost:50002/v1/query/committees-data \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000
      }'

> {
  "list": [
    {
      "chainID": 1,
      "lastRootHeightUpdated": 998,
      "lastChainHeightUpdated": 998,
      "paymentPercents": null,
      "numberOfSamples": 0
    }
  ]
}
```

## Subsidized-Committees

**Route:** `/v1/query/subsidized-committees`

**Description**: responds with a list of chainIds that receive a portion of the 'block reward'

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)

**Response**: `uint64 array` - a list of subsidized chain ids

**Example**:

```
$ curl -X POST localhost:50002/v1/query/subsidized-committees \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000
      }'

> [1, 2, 3]
```

## Retired-Committees

**Route:** `/v1/query/retired-committees`

**Description**: responds with a list of the retired chain ids

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)

**Response**: `uint64 array` - a list of retired chain ids

**Example**:

```
$ curl -X POST localhost:50002/v1/query/retired-committees \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000
      }'

> [1, 2, 3]
```

## Non-Signers

**Route:** `/v1/query/non-signers`

**Description**: responds with data on validators who did not sign recent blocks

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)

**Response**:
- `array` - a list of non-signer data
  - **address**: 20 byte identifier of non-signer
  - **counter**: increments when a validator doesn't sign a block and resets every `non-sign-window`

**Example**:

```
$ curl -X POST localhost:50002/v1/query/non-signers \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000
      }'

> [{
  "address": "551f21e333012027b81701a35023efc88b864975",
  "counter": 1,
}]
```

## Params

**Route:** `/v1/query/params`

**Description**: responds with parameters from all categories in a single object

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)

**Response**:
- **consensus**: `object` - the governance parameters listed under the consensus params space
  - **blockSize**: `uint64` - the maximum block size in bytes
  - **protocolVersion**: `string` - the protocol version and height when the version updated separated by delimiter: `/` (2/100 is version=2 height=100)
  - **rootChainID**: `uint64` - the committee id of the root chain (may be self chain id)
- **validator**: `object` - the governance parameters listed under the validator params space
  - **unstakingBlocks**: `uint64` - the number of blocks a validator is 'unstaking' before it is unstaked and the bonded funds are returned
  - **maxPauseBlocks**: `uint64` - the number of blocks a non-delegate validator may be consecutively paused for before automatically it begins 'unstaking'
  - **doubleSignSlashPercentage**: `uint64` - the percent of stake that is burned for a validator when a double sign is proved with evidence
  - **maxNonSign**: `uint64` - the number of blocks a validator may not sign within a `nonSignWindow` before being slashed
  - **nonSignWindow**: `uint64` - the number of blocks the validator non-sign counter is reset
  - **maxCommittees**: `uint64` - the maximum count of committees a validator may be simultaneously staked for
  - **maxCommitteeSize**: `uint64` - the maximum count of validators simulatneously active in BFT for any 1 committee (sorted highest stake to lowest, above the limit is non-active and non-rewarded)
  - **earlyWithdrawalPenalty**: `uint64` - the percent of the reward that is burned when it is directly withdrawn instead of compounded to a validator's stake
  - **delegateUnstakingBlocks**: `uint64` - the number of blocks a delegate validator is 'unstaking' before it is unstaked and the bonded funds are returned
  - **minimumOrderSize**: `uint64` - the minimum `sell order amount` in micro denomination
  - **stakePercentForSubsidizedCommittee**: `uint64` — minimum percentage of total stake that must be committed to a committee for it to be considered subsidized
  - **maxSlashPerCommittee**: `uint64` - the maximum slash per block a committee may penalize a validator - if exceeded the validator is auto-ejected from the committee and the slash is limited
  - **delegateRewardPercentage**: `uint64` - the percent of the block reward a pseudo-randomly chosen delegate validator, nested-validator, and nested-delegate receives
  - **buyDeadlineBlocks**: `uint64` - the deadline in blocks before a 'locked' sell order is released on the root-chain - (applies only to the 'buyer-side' of token swaps)
  - **lockOrderFeeMultiplier**: `uint64` - the multiplier to a `sendFee` required to `lock` a sell order on the root-chain  - (applies only to the 'buyer-side' of token swaps)
- **fee**: `object` - the governance parameters listed under the fee params space
  - **sendFee**: `uint64` - the minimum fee in micro denomination needed to execute a `send` transaction
  - **stakeFee**: `uint64` - the minimum fee in micro denomination needed to execute a `stake` transaction
  - **editStakeFee**: `uint64` - the minimum fee in micro denomination needed to execute a `editStake` transaction
  - **pauseFee**: `uint64` - the minimum fee in micro denomination needed to execute a `pause` transaction
  - **unpauseFee**: `uint64` - the minimum fee in micro denomination needed to execute a `unpause` transaction
  - **changeParameterFee**: `uint64` - the minimum fee in micro denomination needed to execute a `changeParameter` transaction
  - **daoTransferFee**: `uint64` - the minimum fee in micro denomination needed to execute a `daoTransfer` transaction
  - **certificateResultsFee**: `uint64` - the minimum fee in micro denomination needed to execute a `certificateResults` transaction
  - **subsidyFee**: `uint64` - the minimum fee in micro denomination needed to execute a `subsidy` transaction
  - **createOrderFee**: `uint64` - the minimum fee in micro denomination needed to execute a `createOrder` transaction
  - **editOrderFee**: `uint64` - the minimum fee in micro denomination needed to execute a `editOrder` transaction
  - **deleteOrderFee**: `uint64` - the minimum fee in micro denomination needed to execute a `deleteOrder` transaction
- **governance**: `object` - the governance parameters listed under the governance params space
  - **daoRewardPercentage**: `uint64` - the percent of the block reward that is awarded to the DAO treasury pool

**Example**:

```
$ curl -X POST localhost:50002/v1/query/params \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000
      }'

> {
  "consensus": {
    "blockSize": 1000000,
    "protocolVersion": "1/0",
    "rootChainID": 1
  },
  "validator": {
    "unstakingBlocks": 200000,
    "maxPauseBlocks": 4380,
    "doubleSignSlashPercentage": 10,
    "nonSignSlashPercentage": 1,
    "maxNonSign": 3,
    "nonSignWindow": 5,
    "maxCommittees": 15,
    "maxCommitteeSize": 100,
    "earlyWithdrawalPenalty": 20,
    "delegateUnstakingBlocks": 2,
    "minimumOrderSize": 1000000000,
    "stakePercentForSubsidizedCommittee": 33,
    "maxSlashPerCommittee": 15,
    "delegateRewardPercentage": 10,
    "buyDeadlineBlocks": 60,
    "lockOrderFeeMultiplier": 2
  },
  "fee": {
    "sendFee": 10000,
    "stakeFee": 10000,
    "editStakeFee": 10000,
    "unstakeFee": 10000,
    "pauseFee": 10000,
    "unpauseFee": 10000,
    "changeParameterFee": 10000,
    "daoTransferFee": 10000,
    "certificateResultsFee": 0,
    "subsidyFee": 10000,
    "createOrderFee": 10000,
    "editOrderFee": 10000,
    "deleteOrderFee": 10000
  },
  "governance": {
    "daoRewardPercentage": 5
  }
}
```

## Supply

**Route:** `/v1/query/supply`

**Description**: responds with the token supply information for the blockchain

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)

**Response**:
- **total**: `uint64` - the total tokens existing in the system (minted tokens are added, burned tokens are removed)
- **staked**: `uint64` - the total locked tokens in the protocol (includes delegated)
- **delegatedOnly**: `uint64` - the total locked tokens that are delegated only
- **committeeStaked**: `array` - a list of committees and their committed (staked + delegated) tokens, sorted by stake
  - **id**: `uint64` - the unique identifier of the committee
  - **amount**: `uint64` - the total amount of tokens committed to this committee
  - **poolPoints**: `array` - the list of pool share holders and their respective points
  - **totalPoolPoints**: `uint64` - the total number of pool points for this committee
- **committeeDelegatedOnly**: `array` - a list of committees and their delegated only tokens, sorted by stake
  - **id**: `uint64` - the unique identifier of the committee
  - **amount**: `uint64` - the total amount of delegated tokens to this committee
  - **poolPoints**: `array` - the list of pool share holders and their respective points
  - **totalPoolPoints**: `uint64` - the total number of pool points for this committee

**Example**:

```
$ curl -X POST localhost:50002/v1/query/supply \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000
      }'

> {
  "total": 1000000000000,
  "staked": 500000000000,
  "delegatedOnly": 100000000000,
  "committeeStaked": [
    {
      "id": 1,
      "amount": 150000000000,
      "poolPoints": [],
      "totalPoolPoints": 0
    }
  ],
  "committeeDelegatedOnly": [
    {
      "id": 1,
      "amount": 100000000000,
      "poolPoints": [],
      "totalPoolPoints": 0
    }
  ]
}
```

## Fee Params

**Route:** `/v1/query/fee-params`

**Description**: responds with a fee parameters object

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)

**Response**:
- **sendFee**: `uint64` - the minimum fee in micro denomination needed to execute a `send` transaction
- **stakeFee**: `uint64` - the minimum fee in micro denomination needed to execute a `stake` transaction
- **editStakeFee**: `uint64` - the minimum fee in micro denomination needed to execute a `editStake` transaction
- **pauseFee**: `uint64` - the minimum fee in micro denomination needed to execute a `pause` transaction
- **unpauseFee**: `uint64` - the minimum fee in micro denomination needed to execute a `unpause` transaction
- **changeParameterFee**: `uint64` - the minimum fee in micro denomination needed to execute a `changeParameter` transaction
- **daoTransferFee**: `uint64` - the minimum fee in micro denomination needed to execute a `daoTransfer` transaction
- **certificateResultsFee**: `uint64` - the minimum fee in micro denomination needed to execute a `certificateResults` transaction
- **subsidyFee**: `uint64` - the minimum fee in micro denomination needed to execute a `subsidy` transaction
- **createOrderFee**: `uint64` - the minimum fee in micro denomination needed to execute a `createOrder` transaction
- **editOrderFee**: `uint64` - the minimum fee in micro denomination needed to execute a `editOrder` transaction
- **deleteOrderFee**: `uint64` - the minimum fee in micro denomination needed to execute a `deleteOrder` transaction

**Example**:

```
$ curl -X POST localhost:50002/v1/query/fee-params \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000
      }'

> {
    "sendFee": 10000,
    "stakeFee": 10000,
    "editStakeFee": 10000,
    "unstakeFee": 10000,
    "pauseFee": 10000,
    "unpauseFee": 10000,
    "changeParameterFee": 10000,
    "daoTransferFee": 10000,
    "certificateResultsFee": 0,
    "subsidyFee": 10000,
    "createOrderFee": 10000,
    "editOrderFee": 10000,
    "deleteOrderFee": 10000
  }
```

## Consensus Params

**Route:** `/v1/query/con-params`

**Description**: responds with a consensus parameters object

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)

**Response**:
- **blockSize**: `uint64` - the maximum block size in bytes
- **protocolVersion**: `string` - the protocol version and height when the version updated separated by delimiter: `/` (2/100 is version=2 height=100)
- **rootChainID**: `uint64` - the committee id of the root chain (may be self chain id)

**Example**:

```
$ curl -X POST localhost:50002/v1/query/con-params \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000
      }'

> {
    "blockSize": 1000000,
    "protocolVersion": "1/0",
    "rootChainID": 1
  }
```

## Validator Params

**Route:** `/v1/query/val-params`

**Description**: responds with a validator parameters object

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)

**Response**:
- **unstakingBlocks**: `uint64` - the number of blocks a validator is 'unstaking' before it is unstaked and the bonded funds are returned
- **maxPauseBlocks**: `uint64` - the number of blocks a non-delegate validator may be consecutively paused for before automatically it begins 'unstaking'
- **doubleSignSlashPercentage**: `uint64` - the percent of stake that is burned for a validator when a double sign is proved with evidence
- **maxNonSign**: `uint64` - the number of blocks a validator may not sign within a `nonSignWindow` before being slashed
- **nonSignWindow**: `uint64` - the number of blocks the validator non-sign counter is reset
- **maxCommittees**: `uint64` - the maximum count of committees a validator may be simultaneously staked for
- **maxCommitteeSize**: `uint64` - the maximum count of validators simultaneously active in BFT for any 1 committee (sorted highest stake to lowest, above the limit is non-active and non-rewarded)
- **earlyWithdrawalPenalty**: `uint64` - the percent of the reward that is burned when it is directly withdrawaled instead of compounded to a validator's stake
- **delegateUnstakingBlocks**: `uint64` - the number of blocks a delegate validator is 'unstaking' before it is unstaked and the bonded funds are returned
- **minimumOrderSize**: `uint64` - the minimum `sell order amount` in micro denomination
- **stakePercentForSubsidizedCommittee**: `uint64` — minimum percentage of total stake that must be committed to a committee for it to be considered subsidized
- **maxSlashPerCommittee**: `uint64` - the maximum slash per block a committee may penalize a validator - if exceeded the validator is auto-ejected from the committee and the slash is limited
- **delegateRewardPercentage**: `uint64` - the percent of the block reward a pseudo-randomly chosen delegate validator, nested-validator, and nested-delegate receives
- **buyDeadlineBlocks**: `uint64` - the deadline in blocks before a 'locked' sell order is released on the root-chain - (applies only to the 'buyer-side' of token swaps)
- **lockOrderFeeMultiplier**: `uint64` - the multiplier to a `sendFee` required to `lock` a sell order on the root-chain  - (applies only to the 'buyer-side' of token swaps)

**Example**:

```
$ curl -X POST localhost:50002/v1/query/val-params \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000
      }'

> {
    "unstakingBlocks": 200000,
    "maxPauseBlocks": 4380,
    "doubleSignSlashPercentage": 10,
    "nonSignSlashPercentage": 1,
    "maxNonSign": 3,
    "nonSignWindow": 5,
    "maxCommittees": 15,
    "maxCommitteeSize": 100,
    "earlyWithdrawalPenalty": 20,
    "delegateUnstakingBlocks": 2,
    "minimumOrderSize": 1000000000,
    "stakePercentForSubsidizedCommittee": 33,
    "maxSlashPerCommittee": 15,
    "delegateRewardPercentage": 10,
    "buyDeadlineBlocks": 60,
    "lockOrderFeeMultiplier": 2
  }
```

## Governance Params

**Route:** `/v1/query/gov-params`

**Description**: responds with a governance parameters object

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)

**Response**:
- **daoRewardPercentage**: `uint64` - the percent of the block reward that is awarded to the DAO treasury pool

**Example**:

```
$ curl -X POST localhost:50002/v1/query/gov-params \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000
      }'

> {
    "daoRewardPercentage": 5
  }
```

## Economics Params

**Route:** `/v1/query/eco-params`

**Description**: responds with a economic parameters object

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)

**Response**:
- **mintPerBlock**: `uint64` - the amount minted per block in micro denominiation
- **mintPerCommittee**: `uint64` - the amount of mint distributed to each committee reward pool
- **daoCut**: `uint64` - the amount of mint distributed to the DAO treasury pool
- **proposerCut**: `uint64` - the percent of the mint in this chain's committee pool that is distributed to the block proposer
- **delegateCut**: `uint64` - the percent of the mint in this chain's committee pool that is distributed to the delegate, nested proposer (if applicable), and nested delegate (if applicable)

**Example**:

```
$ curl -X POST localhost:50002/v1/query/eco-params \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000
      }'

> {
  "MintPerBlock": 80000000,
  "MintPerCommittee": 19000000,
  "DAOCut": 4000000,
  "ProposerCut": 70,
  "DelegateCut": 10
}
```

## State

**Route:** `/v1/query/state`

**Description**: responds with the state ledger export in `genesis.json` format

**HTTP Method**: `GET`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)

**Response**:
- **pools**: `array` - see `Pools` response
- **accounts**: `array` - see `Accounts` response
- **nonSigners**: `object` - see `Non-Signers` response
- **validators**: `array` - see `Validator-Set` response
- **params**: `object` - see `Params` response
- **supply**: `object` - see `Supply` response
- **orderBooks**: `array` - see `order-books` response

**Example**:

```
$ curl -X GET "localhost:50002/v1/query/state?height=1000"

> {
  "pools": [
    ...
  ],
  "accounts": [
    ...
  ],
  "validators": [
    ...
  ],
  "params": {
    ...
  },
  "supply": {
    ...
  },
  "orderBooks": [...]
}
```

## State Diff (Browser View)

**Route:** `/v1/query/state-diff`

**Description**: view the differences in state (ledger) between two heights

**HTTP Method**: `GET`

**Request**:

- **height**: `uint64` – the block height to act as the end to calculate the state deltas (optional: use 0 to read from the latest block)
- **start-height**: `uint64` – the start height to act as the 'base' (optional: will auto-use `height-1`)

**Response**:
- **state**: `object` - see `State` but with diff highlights (ex. `"amount": 25257333001 <- 25206666335`)

**Example**:
VIEW IN BROWSER: `http://localhost:50002/v1/query/state-diff?height=1000&startHeight=998`
```
{
  "pools": [
        {
            "amount": 25257333001 <- 25206666335,
            "id": 2
        },
    ...
  ]
  "accounts": [
    ...
  ],
  "validators": [
    ...
  ],
  "params": {
    ...
  },
  "supply": {
    ...
  },
  "orderBooks": [...]
}
```

## State Diff

**Route:** `/v1/query/state-diff`

**Description**: view the differences in state (ledger) between two heights

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to act as the end to calculate the state deltas (optional: use 0 to read from the latest block)
- **start-height**: `uint64` – the start height to act as the 'base' (optional: will auto-use `height-1`)

**Response**:
- **state**: `object` - see `State` but with diff highlights (ex. `"amount": 18239045002 => 18202565004`)

```
$ curl -X POST localhost:50002/v1/query/state-diff \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000,
        "startHeight": 998
      }'

> {
  "pools": [
    ...
  ]
  "accounts": [
      {
          "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
          "amount": 18239045002 => 18202565004
      },
  ],
  "validators": [
    ...
  ],
  "params": {
    ...
  },
  "supply": {
    ...
  },
  "orderBooks": [...]
}
```


## Quorum Certificate By Height

**Route:** `/v1/query/cert-by-height`

**Description**: view a quorum certificate for a specific block height

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to act as the end to calculate the state deltas (optional: use 0 to read from the latest block)

**Response**:
- **header**: `object` - the view of the certificate
  - **height**: `uint64` - the block height of the certificate
  - **committeeHeight**: `uint64` - the block height of the root-chain that corresponds to the committee that signed this block
  - **round**: `uint64` - the 0-indexed iteration count of BFT that finalized this block
  - **phase**: `string` - the phase identifier of the view for this certificate
  - **networkID**: `uint64` - the unique identifier of the network (ex. 1=mainnet, 2=testnet)
  - **chainID**: `uint64` - the unique identifier of the chain this QC corresponds to (ex. 1=Canopy, 2=Canary)
- **block**: `hex string` - the protobuf encoded bytes of the block
- **blockHash**: `hex string` - the SHA256 hash of the block protobuf bytes
- **resultsHash**: `hex string` - the SHA256 hash of the proposal results structure (see below)
- **results**: `object` - the quorum certificate results summary
  - **rewardRecipients**: `object` -  the recipients who are rewarded based on the quorum certificate, specifically who the committee agreed to reward from the committee treasury
    - **paymentPercents**: `array` - the percentage of rewards allocated to each recipient
      - **address**: `hex string` - the 20 byte unique identifier of the reward recipient
      - **percent**: `uint64` - the dilutable share of the committee treasury pool
      - **chainID**: `uint64` - the chain_id where the payment is distributed
    - **slashRecipients**: `object` - the recipients who are penalized (slashed) based on the quorum certificate specifically who the committee agreed to slash due to evidence of bad behavior
      - **doubleSigners**: `array` - a list of actors who the committee agreed double-signed based on evidence
        - **id**: `hex string` - the identifier of the malicious validator
        - **heights**: `uint64 array` - the list of heights when the infractions occurred
    - **orders**: `object` - contains information regarding the 'buying side' of sell orders including actions like 'buy/reserve order' or 'close/complete order'
      - **lockOrders**: `array` - a list of actions where a buyer expresses an intent to purchase an order, often referred to as 'claiming' the order
        - **orderId**: `hex string` - is the id that is unique to this committee to identify the order
        - **chainId**: `uint64` - is the id of the committee
        - **buyerReceiveAddress**: `hex string` - the address where the sold may be received
        - **buyerSendAddress**: `hex string` - the 'counter asset' address where the tokens will be sent from
        - **buyerChainDeadline**: `uint64` - the 'counter asset' chain height at which the buyer must send the 'counter asset' by or the 'intent to buy' will be voided
      - **resetOrders**: `uint64 array` - a list of orders where no funds were sent before the deadline
      - **closeOrders**: `uint64 array` -  list of orders where funds were sent, signaling the committee to transfer escrowed tokens to the buyer's receive address
    - **checkpoint**: `object` - contains information from the 3rd party chain in order for Canopy to provide Checkpoint-as-a-Service
      - **height**: `uint64` - the height of the third party chain
      - **blockHash**: `hex string` - the cryptographic hash of the third party chain block for the height
    - **retired**: `boolean` - signals if the committee wants to shut down and mark itself as 'forever unsubsidized' on the root-chain
- **proposerKey**: `hex string` - the public key of the proposer
- **signature**: `object` - the BLS aggregate signature of the committee
  - **signature**: `hex string` - the bytes of the BLS signature
  - **bitmap**: `hex string` - bitmask of the BLS signature
```
$ curl -X POST localhost:50002/v1/query/cert-by-height \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000
      }'

> {
  "header": {
    "height": 1000,
    "committeeHeight": 1000,
    "round": 0,
    "phase": "PRECOMMIT_VOTE",
    "networkID": 1,
    "chainId": 1
  },
  "block": "0ac00808e80712208a885b1f0f55cb20418a1d6bebb47fd05902d0ab5681640aa1cd2b7c7b943341180120e2a6c9f7fcab8d033005389ba8ba0242203876a5789457f8332129bbc5a4f0adf1aa344dbb0578cada47e13db370faf01e4a20314ad18fd2e0dbf822f90ad8015a8b2c6dfb16c8c70dd5bab3cc47dff2cb48a3522046464646464646464646464646464646464646464646464646464646464646465a20d32453d18883c5842f3a8f65c8636f7397995d97b0a17bcaa6494fb787ce08a76220d32453d18883c5842f3a8f65c8636f7397995d97b0a17bcaa6494fb787ce08a76a14502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c7117298040a88020a8101001f25171bcc85d9711a3a55844d962b06bfa86bb27391a753135b34f42597c267d3261d3c959348ea227d7a698cdcda865ae621db8398d3233ac207bf3d5b12b5e40645f7d9adf1529606022c6e316bbed9014550364d91d45f0b2dee8e949d6799bcdc01b939b9184d56d8fe7fa4d272475719438075f4bfe6aaf64944ed9546128101011e1350c3eeecf89db527e194f95ab7b319ad6efddc842dbd3b55b05e741b6be376934cf7ceb8d7e17f8f62c7dbcc18be4bc521dfdb3e264268c789ca1d2348bc811d563af73a005f21ef58fc743d82c7a31722cc61113f50ec0f1fdfd24935c1fe6e51c4cda087a02bfbfce3faf03cb497ebf95f31f82daf1d27da74950d73b51287020a8101000bd5fb0dfe06245b51453e71e7bcaecb0ccf3cf9447734a9a0bcff97e3c3811cac2d8530ffaee3bd564f3848d308eb3c4a1beccbc37c42973b0d446fcfb6699c82fdf4fb39514aaa234bda4f5a9018308af1ad01c28c47c4f323d5af7f368037db00a0e4a4e591a2fd22d0b0bdaa3bfc507f070ece3e496d4d757f28c0c11eaa12800100b5ba75dfb6fd10e74a5eb5738d98b36cf2d2bbc3d200e7c955a94d5a6b4a644851005d64c516fb1f6b30ed8758a9fe0c365de5b44fdc2acafd6d06c3b8993c1359b4f7e83c4b0ce1fffb99626e0f7d464bca9791761f91bbcfc44f2593e70e2e0cb5fee47f55de7484889c9c998b3b73b952bb4a190c2ffb3bd0f9821afd5518f91c7aab020a0c0801100118e70720e7073006123e0a380a1a0a14551f21e333012027b81701a35023efc88b864975100a18010a1a0a14502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711105a180112001a001a20d540117c8ad0e152f55d06eec5a8f2949369611d8b7ee712526d1ef29dc299ad2a203876a5789457f8332129bbc5a4f0adf1aa344dbb0578cada47e13db370faf01e3230b2947db37385bb43c46244cef15f2451a446cea011fc1a2e1d52b1cecc7a50a8924e0e062555793bbd55a91a685017ee3a650a60a9a4ad120bc737d6191cc489539035108c7e71e887493c297f68bdd42d3c2e5f953790f91d3321789c26cc5fe2831b6114f1c7863e4413a88ce8cbef6e1b13b0b89a440fc0d1e5edb20aff245f6cb0e4481c3fe54fe3e0cc15bdb6a1db64ed1d120101",
  "blockHash": "8a885b1f0f55cb20418a1d6bebb47fd05902d0ab5681640aa1cd2b7c7b943341",
  "resultsHash": "13118b11931aca28dac00e3ca7437f75245728a6bc8e1b9a56fa2f4a06b5380b",
  "results": {
    "rewardRecipients": {
      "paymentPercents": [
        {
          "address": "551f21e333012027b81701a35023efc88b864975",
          "percents": 10,
          "chainId": 1
        },
        {
          "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
          "percents": 90,
          "chainId": 1
        }
      ]
    },
    "slashRecipients": {},
    "orders": {
      "lockOrders": null,
      "resetOrders": null,
      "closeOrders": null
    },
    "checkpoint": {
      "height": 1000,
      "blockHash": "iohbHw9VyyBBih1r67R/0FkC0KtWgWQKoc0rfHuUM0E="
    }
  },
  "proposerKey": "b2947db37385bb43c46244cef15f2451a446cea011fc1a2e1d52b1cecc7a50a8924e0e062555793bbd55a91a685017ee",
  "signature": {
    "signature": "81e8d9d0a72631166d3ef22bc5255d8f7f69634cce9bcd27c2367a8982b72b8fd0e01ee426733047f954a7247bdeed5811221d964dcc6bfb51fa72448eed04080025a9fcb6c984a0ef1987fa76e56d6eb0babd4601e6933c971d672d27227896",
    "bitmap": "01"
  }
}
```

## Blocks

**Route:** `/v1/query/blocks`

**Description**: view a page of blocks

**HTTP Method**: `POST`

**Request**:

- **perPage**: `int` - the number of elements per page (the default is 10 and max is 5,000)
- **pageNumber**: `int` - the number of the page (the default is 1)

**Response**:
- **perPage**: `int` - the number of elements per page
- **pageNumber**: `int` - the number of the page
- **results**: `array` - the list of result objects
  - **blockHeader**: `object` - the summary of the block
    - **height**: `uint64` - the number of blocks in the blockchain before a specific block, indicating its position in the chain and representing the blockchain's length
    - **hash**: `hex string` - a unique identifier for a block, generated by applying a cryptographic hash function to the block's contents, ensuring its integrity and connecting it to the previous block in the chain
    - **networkID**: `uint64` - a unique identifier used to distinguish different canopy blockchain networks, ensuring that transactions and blocks are only processed within the correct network
    - **time**: `unix micro timestamp` - specific time recorded in a block when it is created by the proposer, indicating when the block was proposed
    - **numTxs**: `uint64` - the number of transactions in the block
    - **totalTxs**: `uint64` - the total transactions in the blockchain
    - **totalVDFIterations**: `uint64` - the total number of verifiable random delay function iterations in the blockchain
    - **lastBlockHash**:  `hex string` - the unique identifier of the previous block, chaining this block to the previous
    - **stateRoot**: `hex string` - the merkle root of the 'state commit store' representing  the entire state of the blockchain at this height
    - **transactionRoot**: `hex string` -  the merkle root of the 'transactions' included in this block
    - **validatorRoot**: `hex string` -  the merkle root of the validators that signed the quorum certificate for this height
    - **nextValidatorRoot**: `hex string` - the merkle root of the validators who are responsible for proposing and validating the next block
    - **proposerAddress**: `hex string` - is the short version of the public key of the Validator who proposed this block
    - **vdf**: `object` - the verifiable delay proof for this block. The VDF serves as a protection mechanism against historical forking attacks
      - **proof**: `hex string` - a proof of function completion given a specific seed
      - **output**: `hex string` - the final output of the calculated 'squarings'
      - **iterations**: `uint64` - number of serial executions (proxy for time)
    - **lastQuorumCertificate**: `object` - the quorum certificate from the previous block is included in the block header to ensure all nodes have the same record of blockchain certificates (See `cert-by-height` response above)
  - **transactions**: `array` - list of transactions in the block (see `tx-by-hash`)
- **type**: `string` - the type of results
- **count**: `int` - length of results
- **totalPages**: `int` - number of pages
- **totalCount**: `int` - total number of items that exist in all pages

```
$ curl -X POST localhost:50002/v1/query/blocks \
  -H "Content-Type: application/json" \
  -d '{
        "perPage": 1
      }'

> {
  "pageNumber": 1,
  "perPage": 1,
  "results": [
    {
      "blockHeader": {
        "height": 17424,
        "hash": "1796fdfad7eee44aeb281f68b2b73d5267e3cf4c1f89e69e46377fc740e8ed92",
        "networkID": 1,
        "time": 1749067029952420,
        "totalTxs": 24,
        "totalVDFIterations": 91342093,
        "lastBlockHash": "9335d7bc725665229051a25d34ca8e19a362f5395ce561931a713bd5d6d0bbda",
        "stateRoot": "f4ba7dcfd2b1385cee39838a70c57b5015f5108263feb978a3caaac56891d1ae",
        "transactionRoot": "4646464646464646464646464646464646464646464646464646464646464646",
        "validatorRoot": "d32453d18883c5842f3a8f65c8636f7397995d97b0a17bcaa6494fb787ce08a7",
        "nextValidatorRoot": "d32453d18883c5842f3a8f65c8636f7397995d97b0a17bcaa6494fb787ce08a7",
        "proposerAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
        "vdf": {
          "proof": "0a81010016d93a706a5f9afacd27ebb6dd9f8715a09273c1d7a4ab3e9d4831e34c21f3b23bc39b052af876e3cb3e9eb63dbdddddfb11b290c200bf5c6fda294b8b10afd76602a1ec7a5d23493dfd8d6418562767a5e6af3b498447953bb8c2ed3bd8e37b679eb9367bcb3ca4a752acecaed640e96714e16a398813919a4736325d6f497e12810100058f1ed747a41c0842ea8d556a7c9dcb61e3f789a35460737399c3e78c2c5d3ebc8462c7ef2f52460c46f7f590a7ffe4875b484a2a410efcdd3da2a3727d17c1a337b00a1afffa641016d00cc4c36fbbd7284ad0993737a97f37864963d9092acc03d38201bab61cf5516788ee9581321b6f5dbe190e49983a2d50ed791d09bd",
          "output": "0a81010035dc7cce8849ab056c4d2a028b604f3a6e92fa68292e4a4ec769bf6c3324cf7097948b0e5e0a673d483198ae8d910c033d2083e7e535da93ad3bc55cf8068de72cd4f46f116e031f839d8498a1a38c6032030e69978eb74ff41e3d04e4b7a683ac8495f0aae03f46622410afd3ee35c3e4f3a8822ba64fbf654c6b908699f2091281010124ca09878937f1dc564af4018550e8ac6903a410c3e2cc1ee60ca7b39419e6597a1f2048e348c8bf6e5c97164e71cea1cf5ccda6627252f4c5c7b9921a302b25c8d01837708bf99a716b2faee84e2be6477d559f16142de6430c735cd362ab35b221b3850e0f7c52e7e760501a574ec0c09114807965028040a136db6c7800b5",
          "iterations": 3918
        },
        "lastQuorumCertificate": {
          "header": {
            "height": 17423,
            "committeeHeight": 17423,
            "round": 0,
            "phase": "PRECOMMIT_VOTE",
            "networkID": 1,
            "chainId": 1
          },
          "blockHash": "9335d7bc725665229051a25d34ca8e19a362f5395ce561931a713bd5d6d0bbda",
          "resultsHash": "d540117c8ad0e152f55d06eec5a8f2949369611d8b7ee712526d1ef29dc299ad",
          "results": {
            "rewardRecipients": {
              "paymentPercents": [
                {
                  "address": "551f21e333012027b81701a35023efc88b864975",
                  "percents": 10,
                  "chainId": 1
                },
                {
                  "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
                  "percents": 90,
                  "chainId": 1
                }
              ]
            },
            "slashRecipients": {},
            "orders": {
              "lockOrders": null,
              "resetOrders": null,
              "closeOrders": null
            }
          },
          "proposerKey": "b2947db37385bb43c46244cef15f2451a446cea011fc1a2e1d52b1cecc7a50a8924e0e062555793bbd55a91a685017ee",
          "signature": {
            "signature": "8d06f5beb1154e462961b7155a4a50426f206c429aad8946cc7c78b533ce39f1d706080e902a4e2f6faf2f01aeb0b70b19aed27c915459339513257fc9f015a350034ad50218529792732c55271694eba72480ce89a08f45af507848f2a02284",
            "bitmap": "01"
          }
        }
      },
      "meta": {
        "size": 1092,
        "took": 5027
      }
    }
  ],
  "type": "block-results-page",
  "count": 1,
  "totalPages": 17424,
  "totalCount": 17424
}
```

## Block By Height

**Route:** `/v1/query/block-by-height`

**Description**: view a block by its height

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to act as the end to calculate the state deltas (optional: use 0 to read from the latest block)

**Response**:
- **blockHeader**: `object` - the summary of the block
  - **height**: `uint64` - the number of blocks in the blockchain before a specific block, indicating its position in the chain and representing the blockchain's length
  - **hash**: `hex string` - a unique identifier for a block, generated by applying a cryptographic hash function to the block's contents, ensuring its integrity and connecting it to the previous block in the chain
  - **networkID**: `uint64` - a unique identifier used to distinguish different canopy blockchain networks, ensuring that transactions and blocks are only processed within the correct network
  - **time**: `unix micro timestamp` - specific time recorded in a block when it is created by the proposer, indicating when the block was proposed
  - **numTxs**: `uint64` - the number of transactions in the block
  - **totalTxs**: `uint64` - the total transactions in the blockchain
  - **totalVDFIterations**: `uint64` - the total number of verifiable random delay function iterations in the blockchain
  - **lastBlockHash**:  `hex string` - the unique identifier of the previous block, chaining this block to the previous
  - **stateRoot**: `hex string` - the merkle root of the 'state commit store' representing  the entire state of the blockchain at this height
  - **transactionRoot**: `hex string` -  the merkle root of the 'transactions' included in this block
  - **validatorRoot**: `hex string` -  the merkle root of the validators that signed the quorum certificate for this height
  - **nextValidatorRoot**: `hex string` - the merkle root of the validators who are responsible for proposing and validating the next block
  - **proposerAddress**: `hex string` - is the short version of the public key of the Validator who proposed this block
  - **vdf**: `object` - the verifiable delay proof for this block. The VDF serves as a protection mechanism against historical forking attacks
    - **proof**: `hex string` - a proof of function completion given a specific seed
    - **output**: `hex string` - the final output of the calculated 'squarings'
    - **iterations**: `uint64` - number of serial executions (proxy for time)
  - **lastQuorumCertificate**: `object` - the quorum certificate from the previous block is included in the block header to ensure all nodes have the same record of blockchain certificates (See `cert-by-height` response above)
- **transactions**: `array` - list of transactions in the block (see `tx-by-hash`)

```
$ curl -X POST localhost:50002/v1/query/block-by-height \
  -H "Content-Type: application/json" \
  -d '{
        "height": 17585
      }'

>{
  "blockHeader": {
        "height": 17585,
        "hash": "b269d340ada92332c262c8db97d12a0d695bb8130a328d16fe930995386fe98a",
        "networkID": 1,
        "time": 1749070110492632,
        "numTxs": 1,
        "totalTxs": 25,
        "totalVDFIterations": 92049173,
        "lastBlockHash": "1c67bccae050c751ab3dcee82d85dd69308a139235961cb33ac053ad455b7903",
        "stateRoot": "a023ecf76bfafb896b2b10a4770ada619e01a66ccd200d0e1f4a39464b976d9b",
        "transactionRoot": "73a60df39f3828349f76380e267c1aa17e38cdc1be23812dc18f72acffafb0b6",
        "validatorRoot": "d32453d18883c5842f3a8f65c8636f7397995d97b0a17bcaa6494fb787ce08a7",
        "nextValidatorRoot": "d32453d18883c5842f3a8f65c8636f7397995d97b0a17bcaa6494fb787ce08a7",
        "proposerAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
        "vdf": {
          "proof": "0a8101005839de058a7d12cebe8561088d086d99f747b61488a3b44753af98cfc45224a9928e368c53042a6bd46e8d25dfb62d53ae20b48b7b00a69f47672b7066762536091e91e99ea070fe024cdbed6e18d54b4a3e43b2a6f62576bc1832a067632b7488b5f24d2b6868179ccd5928b3b1cbf77cd5f4fb8e8d91b5425a2445112f98ed128101001be88ca13e77f7f3fb68949c9a7cdd6e4337cd01b9e2da182dd43c1bcb45ece37bf26df4c1df31139e65990a5f358da2cd93371536b32612ef93bed1ffaccff5e81dced6a9533a42820e7f8925e22adaf60b631ee8052bdc255c0cb857f0b250db4dd12788aaa664e29f10db57bafbb085d3ed934f8ac1806a8d20b88f1be197",
          "output": "0a800100581178609afb7c29060ac8b3be0aacc7bc787c541e48b1f60b43e5c66e5e3535bac55ffb553dca0e3c2b9a5c2fdd2c8ed478683066a85434772697f8519e3160b12ec6f9c79af0c7841e2f99c4812443ede0be3add454d9e1ffff3821ea4fe15277b463699c4dd563839006fbe2e4a3ea459f8b44ba222b8e19c4c6af28bb0128001001da6caf12afe6ef40e20ecc5e6ae32f7710dc830146ca5258b76e86611603fcefad0c7d56dd0d40e0eb3608ab3b66e40b5ed18a1fa68a1f200a406e26ed9d9d58a0f8faacbdd5ab70410441cb38376e2d3d58b756329f8bbbca2f1ceb601cce7bb1b90f2c96d3b3a6b4be756c184bac6b3542c3340cca8b5a2d92532516543",
          "iterations": 5560
        },
        "lastQuorumCertificate": {
          "header": {
            "height": 17584,
            "committeeHeight": 17584,
            "round": 0,
            "phase": "PRECOMMIT_VOTE",
            "networkID": 1,
            "chainId": 1
          },
          "blockHash": "1c67bccae050c751ab3dcee82d85dd69308a139235961cb33ac053ad455b7903",
          "resultsHash": "d540117c8ad0e152f55d06eec5a8f2949369611d8b7ee712526d1ef29dc299ad",
          "results": {
            "rewardRecipients": {
              "paymentPercents": [
                {
                  "address": "551f21e333012027b81701a35023efc88b864975",
                  "percents": 10,
                  "chainId": 1
                },
                {
                  "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
                  "percents": 90,
                  "chainId": 1
                }
              ]
            },
            "slashRecipients": {},
            "orders": {
              "lockOrders": null,
              "resetOrders": null,
              "closeOrders": null
            }
          },
          "proposerKey": "b2947db37385bb43c46244cef15f2451a446cea011fc1a2e1d52b1cecc7a50a8924e0e062555793bbd55a91a685017ee",
          "signature": {
            "signature": "a39e08afbb10ff1c95d6bb71a90b13ee22873aa88b918d4d73b6a8b38f69decea98d5a8010e5a5671e62e0774edd88a61170b781e37dfdbf76d73a9d9016e8462e2e97c849eaf0aaddb6d97414a5ba769646b4068c1613725319ca8d75e86168",
            "bitmap": "01"
          }
        }
      },
      "transactions": [
        {
          "sender": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
          "recipient": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
          "messageType": "send",
          "height": 17585,
          "transaction": {
            "type": "send",
            "msg": {
              "fromAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
              "toAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
              "amount": 100000
            },
            "signature": {
              "publicKey": "b2947db37385bb43c46244cef15f2451a446cea011fc1a2e1d52b1cecc7a50a8924e0e062555793bbd55a91a685017ee",
              "signature": "8a427223eae305e0816d66c7940320f7fa17b3a084c1a27130fd4cd0e6fe68eae2edecf1ec197cac1ac049899cdf9e0e10ae7855b2b2aacd213950eb4856dd10b6bc0b77eb34bc3e49e82cabb21aaf85f588876b0568f1b4981961c3e9c525a9"
            },
            "time": 1749070106734452,
            "createdHeight": 17584,
            "fee": 10000,
            "networkID": 1,
            "chainID": 1
          },
          "txHash": "ff22f214f5f300d315a56581ccb0f10fe1665f90c8f09666f7c58abcabfbcedb"
        }
      ],
    "meta": {
      "size": 1092,
      "took": 5022
    }
}
```

## Block By Hash

**Route:** `/v1/query/block-by-hash`

**Description**: view a block by its hash

**HTTP Method**: `POST`

**Request**:

- **hash**: `hex string` – the 32 byte unique identifier of the block

**Response**:
- **blockHeader**: `object` - the summary of the block
  - **height**: `uint64` - the number of blocks in the blockchain before a specific block, indicating its position in the chain and representing the blockchain's length
  - **hash**: `hex string` - a unique identifier for a block, generated by applying a cryptographic hash function to the block's contents, ensuring its integrity and connecting it to the previous block in the chain
  - **networkID**: `uint64` - a unique identifier used to distinguish different canopy blockchain networks, ensuring that transactions and blocks are only processed within the correct network
  - **time**: `unix micro timestamp` - specific time recorded in a block when it is created by the proposer, indicating when the block was proposed
  - **numTxs**: `uint64` - the number of transactions in the block
  - **totalTxs**: `uint64` - the total transactions in the blockchain
  - **totalVDFIterations**: `uint64` - the total number of verifiable random delay function iterations in the blockchain
  - **lastBlockHash**:  `hex string` - the unique identifier of the previous block, chaining this block to the previous
  - **stateRoot**: `hex string` - the merkle root of the 'state commit store' representing  the entire state of the blockchain at this height
  - **transactionRoot**: `hex string` -  the merkle root of the 'transactions' included in this block
  - **validatorRoot**: `hex string` -  the merkle root of the validators that signed the quorum certificate for this height
  - **nextValidatorRoot**: `hex string` - the merkle root of the validators who are responsible for proposing and validating the next block
  - **proposerAddress**: `hex string` - is the short version of the public key of the Validator who proposed this block
  - **vdf**: `object` - the verifiable delay proof for this block. The VDF serves as a protection mechanism against historical forking attacks
    - **proof**: `hex string` - a proof of function completion given a specific seed
    - **output**: `hex string` - the final output of the calculated 'squarings'
    - **iterations**: `uint64` - number of serial executions (proxy for time)
  - **lastQuorumCertificate**: `object` - the quorum certificate from the previous block is included in the block header to ensure all nodes have the same record of blockchain certificates (See `cert-by-height` response above)
- **transactions**: `array` - list of transactions in the block (see `tx-by-hash`)

```
$ curl -X POST localhost:50002/v1/query/block-by-hash \
  -H "Content-Type: application/json" \
  -d '{
        "hash": "b269d340ada92332c262c8db97d12a0d695bb8130a328d16fe930995386fe98a"
      }'

>{
  "blockHeader": {
        "height": 17585,
        "hash": "b269d340ada92332c262c8db97d12a0d695bb8130a328d16fe930995386fe98a",
        "networkID": 1,
        "time": 1749070110492632,
        "numTxs": 1,
        "totalTxs": 25,
        "totalVDFIterations": 92049173,
        "lastBlockHash": "1c67bccae050c751ab3dcee82d85dd69308a139235961cb33ac053ad455b7903",
        "stateRoot": "a023ecf76bfafb896b2b10a4770ada619e01a66ccd200d0e1f4a39464b976d9b",
        "transactionRoot": "73a60df39f3828349f76380e267c1aa17e38cdc1be23812dc18f72acffafb0b6",
        "validatorRoot": "d32453d18883c5842f3a8f65c8636f7397995d97b0a17bcaa6494fb787ce08a7",
        "nextValidatorRoot": "d32453d18883c5842f3a8f65c8636f7397995d97b0a17bcaa6494fb787ce08a7",
        "proposerAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
        "vdf": {
          "proof": "0a8101005839de058a7d12cebe8561088d086d99f747b61488a3b44753af98cfc45224a9928e368c53042a6bd46e8d25dfb62d53ae20b48b7b00a69f47672b7066762536091e91e99ea070fe024cdbed6e18d54b4a3e43b2a6f62576bc1832a067632b7488b5f24d2b6868179ccd5928b3b1cbf77cd5f4fb8e8d91b5425a2445112f98ed128101001be88ca13e77f7f3fb68949c9a7cdd6e4337cd01b9e2da182dd43c1bcb45ece37bf26df4c1df31139e65990a5f358da2cd93371536b32612ef93bed1ffaccff5e81dced6a9533a42820e7f8925e22adaf60b631ee8052bdc255c0cb857f0b250db4dd12788aaa664e29f10db57bafbb085d3ed934f8ac1806a8d20b88f1be197",
          "output": "0a800100581178609afb7c29060ac8b3be0aacc7bc787c541e48b1f60b43e5c66e5e3535bac55ffb553dca0e3c2b9a5c2fdd2c8ed478683066a85434772697f8519e3160b12ec6f9c79af0c7841e2f99c4812443ede0be3add454d9e1ffff3821ea4fe15277b463699c4dd563839006fbe2e4a3ea459f8b44ba222b8e19c4c6af28bb0128001001da6caf12afe6ef40e20ecc5e6ae32f7710dc830146ca5258b76e86611603fcefad0c7d56dd0d40e0eb3608ab3b66e40b5ed18a1fa68a1f200a406e26ed9d9d58a0f8faacbdd5ab70410441cb38376e2d3d58b756329f8bbbca2f1ceb601cce7bb1b90f2c96d3b3a6b4be756c184bac6b3542c3340cca8b5a2d92532516543",
          "iterations": 5560
        },
        "lastQuorumCertificate": {
          "header": {
            "height": 17584,
            "committeeHeight": 17584,
            "round": 0,
            "phase": "PRECOMMIT_VOTE",
            "networkID": 1,
            "chainId": 1
          },
          "blockHash": "1c67bccae050c751ab3dcee82d85dd69308a139235961cb33ac053ad455b7903",
          "resultsHash": "d540117c8ad0e152f55d06eec5a8f2949369611d8b7ee712526d1ef29dc299ad",
          "results": {
            "rewardRecipients": {
              "paymentPercents": [
                {
                  "address": "551f21e333012027b81701a35023efc88b864975",
                  "percents": 10,
                  "chainId": 1
                },
                {
                  "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
                  "percents": 90,
                  "chainId": 1
                }
              ]
            },
            "slashRecipients": {},
            "orders": {
              "lockOrders": null,
              "resetOrders": null,
              "closeOrders": null
            }
          },
          "proposerKey": "b2947db37385bb43c46244cef15f2451a446cea011fc1a2e1d52b1cecc7a50a8924e0e062555793bbd55a91a685017ee",
          "signature": {
            "signature": "a39e08afbb10ff1c95d6bb71a90b13ee22873aa88b918d4d73b6a8b38f69decea98d5a8010e5a5671e62e0774edd88a61170b781e37dfdbf76d73a9d9016e8462e2e97c849eaf0aaddb6d97414a5ba769646b4068c1613725319ca8d75e86168",
            "bitmap": "01"
          }
        }
      },
      "transactions": [
        {
          "sender": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
          "recipient": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
          "messageType": "send",
          "height": 17585,
          "transaction": {
            "type": "send",
            "msg": {
              "fromAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
              "toAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
              "amount": 100000
            },
            "signature": {
              "publicKey": "b2947db37385bb43c46244cef15f2451a446cea011fc1a2e1d52b1cecc7a50a8924e0e062555793bbd55a91a685017ee",
              "signature": "8a427223eae305e0816d66c7940320f7fa17b3a084c1a27130fd4cd0e6fe68eae2edecf1ec197cac1ac049899cdf9e0e10ae7855b2b2aacd213950eb4856dd10b6bc0b77eb34bc3e49e82cabb21aaf85f588876b0568f1b4981961c3e9c525a9"
            },
            "time": 1749070106734452,
            "createdHeight": 17584,
            "fee": 10000,
            "networkID": 1,
            "chainID": 1
          },
          "txHash": "ff22f214f5f300d315a56581ccb0f10fe1665f90c8f09666f7c58abcabfbcedb"
        }
      ],
    "meta": {
      "size": 1092,
      "took": 5022
    }
}
```

## Events By Height

**Route:** `/v1/query/events-by-height`

**Description**: view the events at a specific block height

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to query events for (optional: use 0 to read from the latest block)
- **perPage**: `int` - the number of elements per page (the default is 10 and max is 5,000)
- **pageNumber**: `int` - the number of the page (the default is 1)

**Response**:
- **perPage**: `int` - the number of elements per page
- **pageNumber**: `int` - the number of the page
- **results**: `array` - the list of result objects
  - **eventType**: `string` - the type of the event: 'reward', 'slash', 'dex-liquidity-deposit', 'dex-liquidity-withdraw', 'dex-swap', 'order-book-swap', 'automatic-pause', 'automatic-begin-unstaking', or 'automatic-finish-unstaking'
  - **msg**: `object` - the actual event message payload, which varies by event type:
    - **reward**: `{ "amount": uint64 }` - amount of reward
    - **slash**: `{ "amount": uint64 }` - amount of slash
    - **dex-liquidity-deposit**: `{ "amount": uint64, "localOrigin": bool, "orderId": hex string, "points": uint64 }` - deposit amount, whether it was made on this chain or the counter, unique order identifier, and amount of points created
    - **dex-liquidity-withdraw**: `{ "localAmount": uint64, "remoteAmount": uint64, "orderId": hex string, "pointsBurned": uint64 }` - amount of liquidity received on local and remote chains, unique order identifier, and amount of points burned
    - **dex-swap**: `{ "soldAmount": uint64, "boughtAmount": uint64, "localOrigin": bool, "success": bool, "orderId": hex string }` - amounts sold/bought, direction, success status, and unique order identifier
    - **order-book-swap**: `{ "soldAmount": uint64, "boughtAmount": uint64, "data": hex string, "sellerReceiveAddress": hex string, "buyerReceiveAddress": hex string, "sellersSendAddress": hex string, "orderId": hex string }` - order book swap details including addresses and order information
    - **automatic-pause**: `{}` - empty object
    - **automatic-begin-unstaking**: `{}` - empty object
    - **automatic-finish-unstaking**: `{}` - empty object
  - **height**: `uint64` - the block height at which the event occurred
  - **reference**: `string` - the reference for the event: 'begin_block', transaction hash, or 'end_block'
  - **chainId**: `uint64` - the chain id associated with the event
  - **address**: `hex string` - the address the event is indexed for
- **type**: `string` - the type of results
- **count**: `int` - length of results
- **totalPages**: `int` - number of pages
- **totalCount**: `int` - total number of items that exist in all pages

```
$ curl -X POST localhost:50002/v1/query/events-by-height \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000
      }'

> {
  "pageNumber": 1,
  "perPage": 10,
  "results": [
    {
      "eventType": "reward",
      "msg": {
        "amount": 10000
      },
      "height": 1000,
      "reference": "end_block",
      "chainId": 1,
      "address": "551f21e333012027b81701a35023efc88b864975"
    },
    {
      "eventType": "dex-swap",
      "msg": {
        "soldAmount": 1000000,
        "boughtAmount": 950000,
        "localOrigin": true,
        "success": true,
        "orderId": "ff22f214f5f300d315a5"
      },
      "height": 1000,
      "reference": "ff22f214f5f300d315a56581ccb0f10fe1665f90c8f09666f7c58abcabfbcedb",
      "chainId": 1,
      "address": "551f21e333012027b81701a35023efc88b864975"
    },
    {
      "eventType": "order-book-swap",
      "msg": {
        "soldAmount": 500000,
        "boughtAmount": 475000,
        "data": "",
        "sellerReceiveAddress": "551f21e333012027b81701a35023efc88b864975",
        "sellerReceiveAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
        "sellersSendAddress": "551f21e333012027b81701a35023efc88b864975",
        "orderId": "a1b2c3d4e5f6"
      },
      "height": 1000,
      "reference": "a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdef0",
      "chainId": 1,
      "address": "551f21e333012027b81701a35023efc88b864975"
    }
  ],
  "type": "event-results-page",
  "count": 3,
  "totalPages": 1,
  "totalCount": 3
}
```

## Events By Address

**Route:** `/v1/query/events-by-address`

**Description**: view the events for a specific address

**HTTP Method**: `POST`

**Request**:

- **address**: `hex string` – the address to query events for
- **perPage**: `int` - the number of elements per page (the default is 10 and max is 5,000)
- **pageNumber**: `int` - the number of the page (the default is 1)

**Response**:
- **perPage**: `int` - the number of elements per page
- **pageNumber**: `int` - the number of the page
- **results**: `array` - the list of result objects
  - **eventType**: `string` - the type of the event: 'reward', 'slash', 'dex-liquidity-deposit', 'dex-liquidity-withdraw', 'dex-swap', 'order-book-swap', 'automatic-pause', 'automatic-begin-unstaking', or 'automatic-finish-unstaking'
  - **msg**: `object` - the actual event message payload, which varies by event type:
    - **reward**: `{ "amount": uint64 }` - amount of reward
    - **slash**: `{ "amount": uint64 }` - amount of slash
    - **dex-liquidity-deposit**: `{ "amount": uint64, "localOrigin": bool, "orderId": hex string, "points": uint64 }` - deposit amount, whether it was made on this chain or the counter, unique order identifier, and amount of points created
    - **dex-liquidity-withdraw**: `{ "localAmount": uint64, "remoteAmount": uint64, "orderId": hex string, "pointsBurned": uint64 }` - amount of liquidity received on local and remote chains, unique order identifier, and amount of points burned
    - **dex-swap**: `{ "soldAmount": uint64, "boughtAmount": uint64, "localOrigin": bool, "success": bool, "orderId": hex string }` - amounts sold/bought, direction, success status, and unique order identifier
    - **order-book-swap**: `{ "soldAmount": uint64, "boughtAmount": uint64, "data": hex string, "sellerReceiveAddress": hex string, "buyerReceiveAddress": hex string, "sellersSendAddress": hex string, "orderId": hex string }` - order book swap details including addresses and order information
    - **automatic-pause**: `{}` - empty object
    - **automatic-begin-unstaking**: `{}` - empty object
    - **automatic-finish-unstaking**: `{}` - empty object
  - **height**: `uint64` - the block height at which the event occurred
  - **reference**: `string` - the reference for the event: 'begin_block', transaction hash, or 'end_block'
  - **chainId**: `uint64` - the chain id associated with the event
  - **address**: `hex string` - the address the event is indexed for
- **type**: `string` - the type of results
- **count**: `int` - length of results
- **totalPages**: `int` - number of pages
- **totalCount**: `int` - total number of items that exist in all pages

```
$ curl -X POST localhost:50002/v1/query/events-by-address \
  -H "Content-Type: application/json" \
  -d '{
        "address": "551f21e333012027b81701a35023efc88b864975"
      }'

> {
  "pageNumber": 1,
  "perPage": 10,
  "results": [
    {
      "eventType": "reward",
      "msg": {
        "amount": 10000
      },
      "height": 1000,
      "reference": "end_block",
      "chainId": 1,
      "address": "551f21e333012027b81701a35023efc88b864975"
    },
    {
      "eventType": "dex-swap",
      "msg": {
        "soldAmount": 1000000,
        "boughtAmount": 950000,
        "localOrigin": true,
        "success": true,
        "orderId": "ff22f214f5f300d315a5"
      },
      "height": 1000,
      "reference": "ff22f214f5f300d315a56581ccb0f10fe1665f90c8f09666f7c58abcabfbcedb",
      "chainId": 1,
      "address": "551f21e333012027b81701a35023efc88b864975"
    },
    {
      "eventType": "order-book-swap",
      "msg": {
        "soldAmount": 500000,
        "boughtAmount": 475000,
        "data": "0x",
        "sellerReceiveAddress": "551f21e333012027b81701a35023efc88b864975",
        "sellerReceiveAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
        "sellersSendAddress": "551f21e333012027b81701a35023efc88b864975",
        "orderId": "a1b2c3d4e5f6"
      },
      "height": 1000,
      "reference": "a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdef0",
      "chainId": 1,
      "address": "551f21e333012027b81701a35023efc88b864975"
    }
  ],
  "type": "event-results-page",
  "count": 3,
  "totalPages": 1,
  "totalCount": 3
}
```

## Events By Chain

**Route:** `/v1/query/events-by-chain`

**Description**: view the events for a specific chain id

**HTTP Method**: `POST`

**Request**:

- **chainId**: `uint64` – the chain id to query events for
- **perPage**: `int` - the number of elements per page (the default is 10 and max is 5,000)
- **pageNumber**: `int` - the number of the page (the default is 1)

**Response**:
- **perPage**: `int` - the number of elements per page
- **pageNumber**: `int` - the number of the page
- **results**: `array` - the list of result objects
  - **eventType**: `string` - the type of the event: 'reward', 'slash', 'dex-liquidity-deposit', 'dex-liquidity-withdraw', 'dex-swap', 'order-book-swap', 'automatic-pause', 'automatic-begin-unstaking', or 'automatic-finish-unstaking'
  - **msg**: `object` - the actual event message payload, which varies by event type:
    - **reward**: `{ "amount": uint64 }` - amount of reward
    - **slash**: `{ "amount": uint64 }` - amount of slash
    - **dex-liquidity-deposit**: `{ "amount": uint64, "localOrigin": bool, "orderId": hex string, "points": uint64 }` - deposit amount, whether it was made on this chain or the counter, unique order identifier, and amount of points created
    - **dex-liquidity-withdraw**: `{ "localAmount": uint64, "remoteAmount": uint64, "orderId": hex string, "pointsBurned": uint64 }` - amount of liquidity received on local and remote chains, unique order identifier, and amount of points burned
    - **dex-swap**: `{ "soldAmount": uint64, "boughtAmount": uint64, "localOrigin": bool, "success": bool, "orderId": hex string }` - amounts sold/bought, direction, success status, and unique order identifier
    - **order-book-swap**: `{ "soldAmount": uint64, "boughtAmount": uint64, "data": hex string, "sellerReceiveAddress": hex string, "buyerReceiveAddress": hex string, "sellersSendAddress": hex string, "orderId": hex string }` - order book swap details including addresses and order information
    - **automatic-pause**: `{}` - empty object
    - **automatic-begin-unstaking**: `{}` - empty object
    - **automatic-finish-unstaking**: `{}` - empty object
  - **height**: `uint64` - the block height at which the event occurred
  - **reference**: `string` - the reference for the event: 'begin_block', transaction hash, or 'end_block'
  - **chainId**: `uint64` - the chain id associated with the event
  - **address**: `hex string` - the address the event is indexed for
- **type**: `string` - the type of results
- **count**: `int` - length of results
- **totalPages**: `int` - number of pages
- **totalCount**: `int` - total number of items that exist in all pages

```
$ curl -X POST localhost:50002/v1/query/events-by-chain \
  -H "Content-Type: application/json" \
  -d '{
        "chainId": 1
      }'

> {
  "pageNumber": 1,
  "perPage": 10,
  "results": [
    {
      "eventType": "reward",
      "msg": {
        "amount": 10000
      },
      "height": 1000,
      "reference": "end_block",
      "chainId": 1,
      "address": "551f21e333012027b81701a35023efc88b864975"
    },
    {
      "eventType": "dex-swap",
      "msg": {
        "soldAmount": 1000000,
        "boughtAmount": 950000,
        "localOrigin": true,
        "success": true,
        "orderId": "ff22f214f5f300d315a5"
      },
      "height": 1000,
      "reference": "ff22f214f5f300d315a56581ccb0f10fe1665f90c8f09666f7c58abcabfbcedb",
      "chainId": 1,
      "address": "551f21e333012027b81701a35023efc88b864975"
    },
    {
      "eventType": "order-book-swap",
      "msg": {
        "soldAmount": 500000,
        "boughtAmount": 475000,
        "data": "",
        "sellerReceiveAddress": "551f21e333012027b81701a35023efc88b864975",
        "sellerReceiveAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
        "sellersSendAddress": "551f21e333012027b81701a35023efc88b864975",
        "orderId": "a1b2c3d4e5f6"
      },
      "height": 1000,
      "reference": "a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdef0",
      "chainId": 1,
      "address": "551f21e333012027b81701a35023efc88b864975"
    }
  ],
  "type": "event-results-page",
  "count": 3,
  "totalPages": 1,
  "totalCount": 3
}
```

## Transactions By Height

**Route:** `/v1/query/txs-by-height`

**Description**: view the transactions included in a block at a specific height

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to act as the end to calculate the state deltas (optional: use 0 to read from the latest block)
- **perPage**: `int` - the number of elements per page (the default is 10 and max is 5,000)
- **pageNumber**: `int` - the number of the page (the default is 1)

**Response**:
- **perPage**: `int` - the number of elements per page
- **pageNumber**: `int` - the number of the page
- **results**: `array` - the list of result objects
  - **blockHeader**: `object` - the summary of the block
  - **transactions**: `array` - list of transactions in the block
    - **sender**: `hex-string` - the address of the user sending the transaction
    - **recipient**: `hex-string` - the address of the user receiving the transaction
    - **messageType**: `string` - the name of the of the message like 'send' or 'stake'
    - **height**: `uint64` - the block height at which the transaction was included
    - **index**: `uint64` - the position of the transaction within the block
    - **transaction**: `object` - original transaction object
      - **messageType**: `string` - type of the transaction like 'send' or 'stake'
      - **msg**: `object` - the actual transaction message payload, which is encapsulated in a generic message format. (See `tx-by-hash`)
      - **signature**: `object` - the cryptographic signature used to verify the authenticity of the transaction
      - **createdHeight**: `uint64` - the height when the transaction was created - allows 'safe pruning'
      - **time**: `unix micro timestamp` - timestamp when the transaction was created - used as temporal entropy to prevent hash collisions in txs
      - **fee**: `uint64` - fee associated with processing the transaction in micro denomination
      - **memo**: `string` - an optional message or note attached to the transaction
      - **networkId**: `uint64` - the identity of the network the transaction is intended for
      - **chainId**: `uint64` - the identity of the committee the transaction is intended for
    - **txHash**: `hex-string` - the unique hash that identifies the transaction
- **type**: `string` - the type of results
- **count**: `int` - length of results
- **totalPages**: `int` - number of pages
- **totalCount**: `int` - total number of items that exist in all pages


```
$ curl -X POST localhost:50002/v1/query/txs-by-height \
  -H "Content-Type: application/json" \
  -d '{
        "height": 17585
      }'

>{
  "pageNumber": 1,
  "perPage": 10,
  "results": [
    {
      "sender": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
      "recipient": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
      "messageType": "send",
      "height": 17585,
      "transaction": {
        "type": "send",
        "msg": {
          "fromAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
          "toAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
          "amount": 100000
        },
        "signature": {
          "publicKey": "b2947db37385bb43c46244cef15f2451a446cea011fc1a2e1d52b1cecc7a50a8924e0e062555793bbd55a91a685017ee",
          "signature": "8a427223eae305e0816d66c7940320f7fa17b3a084c1a27130fd4cd0e6fe68eae2edecf1ec197cac1ac049899cdf9e0e10ae7855b2b2aacd213950eb4856dd10b6bc0b77eb34bc3e49e82cabb21aaf85f588876b0568f1b4981961c3e9c525a9"
        },
        "time": 1749070106734452,
        "createdHeight": 17584,
        "fee": 10000,
        "networkID": 1,
        "chainID": 1
      },
      "txHash": "ff22f214f5f300d315a56581ccb0f10fe1665f90c8f09666f7c58abcabfbcedb"
    }
  ],
  "type": "tx-results-page",
  "count": 1,
  "totalPages": 1,
  "totalCount": 1
}
```

## Transactions By Sender

**Route:** `/v1/query/txs-by-sender`

**Description**: view the transactions sent by an address

**HTTP Method**: `POST`

**Request**:

- **address**: `hex-string` – the address of the user sending the transaction
- **perPage**: `int` - the number of elements per page (the default is 10 and max is 5,000)
- **pageNumber**: `int` - the number of the page (the default is 1)

**Response**:
- **perPage**: `int` - the number of elements per page
- **pageNumber**: `int` - the number of the page
- **results**: `array` - the list of result objects
  - **blockHeader**: `object` - the summary of the block
  - **transactions**: `array` - list of transactions in the block
    - **sender**: `hex-string` - the address of the user sending the transaction
    - **recipient**: `hex-string` - the address of the user receiving the transaction
    - **messageType**: `string` - the name of the of the message like 'send' or 'stake'
    - **height**: `uint64` - the block height at which the transaction was included
    - **index**: `uint64` - the position of the transaction within the block
    - **transaction**: `object` - original transaction object
      - **messageType**: `string` - type of the transaction like 'send' or 'stake'
      - **msg**: `object` - the actual transaction message payload, which is encapsulated in a generic message format. (See `tx-by-hash`)
      - **signature**: `object` - the cryptographic signature used to verify the authenticity of the transaction
      - **createdHeight**: `uint64` - the height when the transaction was created - allows 'safe pruning'
      - **time**: `unix micro timestamp` - timestamp when the transaction was created - used as temporal entropy to prevent hash collisions in txs
      - **fee**: `uint64` - fee associated with processing the transaction in micro denomination
      - **memo**: `string` - an optional message or note attached to the transaction
      - **networkId**: `uint64` - the identity of the network the transaction is intended for
      - **chainId**: `uint64` - the identity of the committee the transaction is intended for
    - **txHash**: `hex-string` - the unique hash that identifies the transaction
- **type**: `string` - the type of results
- **count**: `int` - length of results
- **totalPages**: `int` - number of pages
- **totalCount**: `int` - total number of items that exist in all pages


```
$ curl -X POST localhost:50002/v1/query/txs-by-sender \
  -H "Content-Type: application/json" \
  -d '{
        "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711"
      }'

>{
  "pageNumber": 1,
  "perPage": 10,
  "results": [
    {
      "sender": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
      "recipient": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
      "messageType": "send",
      "height": 17585,
      "transaction": {
        "type": "send",
        "msg": {
          "fromAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
          "toAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
          "amount": 100000
        },
        "signature": {
          "publicKey": "b2947db37385bb43c46244cef15f2451a446cea011fc1a2e1d52b1cecc7a50a8924e0e062555793bbd55a91a685017ee",
          "signature": "8a427223eae305e0816d66c7940320f7fa17b3a084c1a27130fd4cd0e6fe68eae2edecf1ec197cac1ac049899cdf9e0e10ae7855b2b2aacd213950eb4856dd10b6bc0b77eb34bc3e49e82cabb21aaf85f588876b0568f1b4981961c3e9c525a9"
        },
        "time": 1749070106734452,
        "createdHeight": 17584,
        "fee": 10000,
        "networkID": 1,
        "chainID": 1
      },
      "txHash": "ff22f214f5f300d315a56581ccb0f10fe1665f90c8f09666f7c58abcabfbcedb"
    }
  ],
  "type": "tx-results-page",
  "count": 1,
  "totalPages": 1,
  "totalCount": 1
}
```

## Transactions By Recipient

**Route:** `/v1/query/txs-by-rec`

**Description**: view the transactions received by an address

**HTTP Method**: `POST`

**Request**:

- **address**: `hex-string` – the address of the user sending the transaction
- **perPage**: `int` - the number of elements per page (the default is 10 and max is 5,000)
- **pageNumber**: `int` - the number of the page (the default is 1)

**Response**:
- **perPage**: `int` - the number of elements per page
- **pageNumber**: `int` - the number of the page
- **results**: `array` - the list of result objects
  - **blockHeader**: `object` - the summary of the block
  - **transactions**: `array` - list of transactions in the block
    - **sender**: `hex-string` - the address of the user sending the transaction
    - **recipient**: `hex-string` - the address of the user receiving the transaction
    - **messageType**: `string` - the name of the of the message like 'send' or 'stake'
    - **height**: `uint64` - the block height at which the transaction was included
    - **index**: `uint64` - the position of the transaction within the block
    - **transaction**: `object` - original transaction object
      - **messageType**: `string` - type of the transaction like 'send' or 'stake'
      - **msg**: `object` - the actual transaction message payload, which is encapsulated in a generic message format. (See `tx-by-hash`)
      - **signature**: `object` - the cryptographic signature used to verify the authenticity of the transaction
      - **createdHeight**: `uint64` - the height when the transaction was created - allows 'safe pruning'
      - **time**: `unix micro timestamp` - timestamp when the transaction was created - used as temporal entropy to prevent hash collisions in txs
      - **fee**: `uint64` - fee associated with processing the transaction in micro denomination
      - **memo**: `string` - an optional message or note attached to the transaction
      - **networkId**: `uint64` - the identity of the network the transaction is intended for
      - **chainId**: `uint64` - the identity of the committee the transaction is intended for
    - **txHash**: `hex-string` - the unique hash that identifies the transaction
- **type**: `string` - the type of results
- **count**: `int` - length of results
- **totalPages**: `int` - number of pages
- **totalCount**: `int` - total number of items that exist in all pages


```
$ curl -X POST localhost:50002/v1/query/txs-by-rec \
  -H "Content-Type: application/json" \
  -d '{
        "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711"
      }'

>{
  "pageNumber": 1,
  "perPage": 10,
  "results": [
    {
      "sender": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
      "recipient": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
      "messageType": "send",
      "height": 17585,
      "transaction": {
        "type": "send",
        "msg": {
          "fromAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
          "toAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
          "amount": 100000
        },
        "signature": {
          "publicKey": "b2947db37385bb43c46244cef15f2451a446cea011fc1a2e1d52b1cecc7a50a8924e0e062555793bbd55a91a685017ee",
          "signature": "8a427223eae305e0816d66c7940320f7fa17b3a084c1a27130fd4cd0e6fe68eae2edecf1ec197cac1ac049899cdf9e0e10ae7855b2b2aacd213950eb4856dd10b6bc0b77eb34bc3e49e82cabb21aaf85f588876b0568f1b4981961c3e9c525a9"
        },
        "time": 1749070106734452,
        "createdHeight": 17584,
        "fee": 10000,
        "networkID": 1,
        "chainID": 1
      },
      "txHash": "ff22f214f5f300d315a56581ccb0f10fe1665f90c8f09666f7c58abcabfbcedb"
    }
  ],
  "type": "tx-results-page",
  "count": 1,
  "totalPages": 1,
  "totalCount": 1
}
```


## Transaction By Hash

**Route:** `/v1/query/tx-by-hash`

**Description**: view a transaction by its hash

**HTTP Method**: `POST`

**Request**:

- **hash**: `hex-string` – the 32 byte unique id of the transaction

**Response**:
- **sender**: `hex-string` - the address of the user sending the transaction
- **recipient**: `hex-string` - the address of the user receiving the transaction
- **messageType**: `string` - the name of the of the message like 'send' or 'stake'
- **height**: `uint64` - the block height at which the transaction was included
- **index**: `uint64` - the position of the transaction within the block
- **transaction**: `object` - original transaction object
  - **messageType**: `string` - type of the transaction like 'send' or 'stake'
  - **msg**: `object` - the actual transaction message payload, which is encapsulated in a generic message format.
    **Oneof:**
    - **messageSend**:
      - **fromAddress**: `hex string` - the sender of the funds
      - **toAddress**: `hex string` - the recipient of the funds
      - **amount**: `uint64` - the amount of tokens in micro-denomination (uCNPY)
    - **messageStake**:
      - **publicKey**: `hex string` - the public cryptographic identity of the Validator operator
      - **amount**: `uint64` - the amount of tokens to be locked as a surety bond
      - **committees**: `repeated uint64` - the list of committees the validator is staking towards
      - **netAddress**: `string` - the tcp peer-to-peer address of the peer
      - **outputAddress**: `hex string` - the reward/unstaking address
      - **delegate**: `bool` - indicates if the validator is a delegate
      - **compound**: `bool` - indicates if the validator is auto-compounding
      - **signer**: `hex string` - the authorized signer of the transaction
    - **messageEditStake**:
      - **address**: `hex string` - the identifier linked to the Validator structure
      - **amount**: `uint64` - updated amount of tokens being staked
      - **committees**: `repeated uint64` - updated list of committees
      - **netAddress**: `string` - updated tcp peer-to-peer address
      - **outputAddress**: `hex string` - updated reward address
      - **compound**: `bool` - updated auto-compounding status
      - **signer**: `hex string` - identifies the signer authorized to edit output
    - **messageUnstake**:
      - **address**: `hex string` - the identifier linked to the Validator structure
    - **messagePause**:
      - **address**: `hex string` - the identifier linked to the Validator structure
    - **messageUnpause**:
      - **address**: `hex string` - the identifier linked to the Validator structure
    - **messageChangeParameter**:
      - **parameterSpace**: `string` - the path where the parameter is located (val, cons, fee, gov)
      - **parameterKey**: `string` - the name of the parameter
      - **parameterValue**: `Any` - the value of the parameter (uint64 or string)
      - **startHeight**: `uint64` - start block height for proposal validity
      - **endHeight**: `uint64` - end block height for proposal validity
      - **signer**: `hex string` - the sender/creator of the proposal
      - **proposalHash**: `string` - internal hash to track the proposal
    - **messageDAOTransfer**:
      - **address**: `hex string` - recipient address and sender of the message
      - **amount**: `uint64` - amount of tokens
      - **startHeight**: `uint64` - start block height for vote window
      - **endHeight**: `uint64` - end block height for vote window
      - **proposalHash**: `string` - internal hash to track the proposal
    - **messageSubsidy**:
      - **address**: `hex string` - sender of the funds
      - **chainID**: `uint64` - id of the committee receiving the funds
      - **amount**: `uint64` - amount of tokens transferred to recipient pool
      - **opcode**: `hex string` - code designating fund instructions
    - **messageCertificateResult**:
      - **qc**: `object` - see `cert-by-height`
    - **messageCreateOrder**:
      - **chainID**: `uint64` - id of the committee handling the counter asset
      - **data**: `hex string` - generic data for swap logic
      - **amountForSale**: `uint64` - uCNPY listed for sale and moved to escrow
      - **requestedAmount**: `uint64` - counter asset required for swap completion
      - **sellerReceiveAddress**: `hex string` - address to receive counter asset
      - **sellersSendAddress**: `hex string` - Canopy address from which seller signs
      - **orderId**: `hex string` - unique identifier auto-populated by state machine
    - **messageEditOrder**:
      - **orderID**: `hex string` - unique identifier for the order
      - **chainID**: `uint64` - committee responsible for the counter asset
      - **data**: `hex string` - updated swap logic data
      - **amountForSale**: `uint64` - updated uCNPY for sale
      - **requestedAmount**: `uint64` - updated counter asset amount
      - **sellerReceiveAddress**: `hex string` - updated recipient address for counter asset
    - **messageDeleteOrder**:
      - **orderID**: `hex string` - identifier of the order to delete
      - **chainID**: `uint64` - committee responsible for the order
  - **signature**: `object` - the cryptographic signature used to verify the authenticity of the transaction
  - **createdHeight**: `uint64` - the height when the transaction was created - allows 'safe pruning'
  - **time**: `unix micro timestamp` - timestamp when the transaction was created - used as temporal entropy to prevent hash collisions in txs
  - **fee**: `uint64` - fee associated with processing the transaction in micro denomination
  - **memo**: `string` - an optional message or note attached to the transaction
  - **networkId**: `uint64` - the identity of the network the transaction is intended for
  - **chainId**: `uint64` - the identity of the committee the transaction is intended for
- **txHash**: `hex-string` - the unique hash that identifies the transaction


```
$ curl -X POST localhost:50002/v1/query/tx-by-hash \
  -H "Content-Type: application/json" \
  -d '{
        "hash": "ff22f214f5f300d315a56581ccb0f10fe1665f90c8f09666f7c58abcabfbcedb"
      }'

>{
  "sender": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
  "recipient": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
  "messageType": "send",
  "height": 17585,
  "transaction": {
    "type": "send",
    "msg": {
      "fromAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
      "toAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
      "amount": 100000
    },
    "signature": {
      "publicKey": "b2947db37385bb43c46244cef15f2451a446cea011fc1a2e1d52b1cecc7a50a8924e0e062555793bbd55a91a685017ee",
      "signature": "8a427223eae305e0816d66c7940320f7fa17b3a084c1a27130fd4cd0e6fe68eae2edecf1ec197cac1ac049899cdf9e0e10ae7855b2b2aacd213950eb4856dd10b6bc0b77eb34bc3e49e82cabb21aaf85f588876b0568f1b4981961c3e9c525a9"
    },
    "time": 1749070106734452,
    "createdHeight": 17584,
    "fee": 10000,
    "networkID": 1,
    "chainID": 1
  },
  "txHash": "ff22f214f5f300d315a56581ccb0f10fe1665f90c8f09666f7c58abcabfbcedb"
}
```


## Order

**Route:** `/v1/query/order`

**Description**: view a sell order by its unique identifier

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **committee**: `uint64` – the unique identifier of the committee
- **orderId**: `hex-string` – the unique identifier of the order

**Response**:
- **id**: `hex-string` - the unique identifier of the order
- **committee**: `uint64` - the id of the committee that is in-charge of escrow for the swap
- **data**: `hex-string` - a generic data field which can allow a committee to execute specific functionality for the swap
- **amountForSale**: `uint64` - amount of 'root-chain-asset' for sale  in smallest unit
- **requestedAmount**: `uint64` - amount of 'counter-asset' the seller of the 'root-chain-asset' receives
- **sellerReceiveAddress**: `hex-string` - the external chain address to receive the 'counter-asset' in smallest unit
- **buyerSendAddress**: `hex-string` - if reserved (locked): the address the buyer will be transferring the funds from
- **buyerChainDeadline**: `hex-string` - the external chain height deadline to send the 'tokens' to SellerReceiveAddress
- **sellersSendAddress**: `hex-string` - the signing address of seller who is selling the CNPY


```
$ curl -X POST localhost:50002/v1/query/order \
  -H "Content-Type: application/json" \
  -d '{
        "committee": 1,
        "orderId": "abb1f314f5f300d315a56581ccb0f10fe1665f90c8f09666f7c58abcabfbcedb",
        "height": 1000
      }'

>{
  "id": "abb1f314f5f300d315a56581ccb0f10fe1665f90c8f09666f7c58abcabfbcedb",
  "committee": "1",
  "data": "",
  "amountForSale": 1000000000000,
  "requestedAmount": 2000000000000,
  "sellersReceiveAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
  "buyerSendAddress": "aaac0b3d64c12c6f164545545b2ba2ab4d80deff",
  "buyerChainDeadline": 17585,
  "sellersSendAddress": "bb43c46244cef15f2451a446cea011fc1a2eddfe"
}
```

## Orders

**Route:** `/v1/query/orders`

**Description**: view all sell orders for a counter-asset pair with optional filters and pagination

**HTTP Method**: `POST`

**Request**:

- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **committee**: `uint64` – the unique identifier of the committee to filter by (optional: use 0 to get all committees)
- **sellersSendAddress**: `hex-string` – the seller address to filter orders by (optional: use "" to get all seller addresses)
- **buyerSendAddress**: `hex-string` – the buyer address to filter locked orders by (optional: use "" to get all buyer addresses)
- **pageNumber**: `int` – the page number to retrieve (optional: starts at 1)
- **perPage**: `int` – the number of orders per page (optional: defaults to system default)

**Note**: `sellersSendAddress` and `buyerSendAddress` are mutually exclusive filters. You cannot use both in the same request.

**Response**:
- **pageNumber**: `int` - the current page number
- **perPage**: `int` - the number of items per page
- **results**: `sell order array` - the paginated list of sell orders
  - **id**: `hex string` - the 20 byte identifier of the order
  - **committee**: `uint64` - the id of the committee that is in-charge of escrow for the swap
  - **data**: `hex-string` - a generic data field which can allow a committee to execute specific functionality for the swap
  - **amountForSale**: `uint64` - amount of 'root-chain-asset' for sale
  - **requestedAmount**: `uint64` - amount of 'counter-asset' the seller of the 'root-chain-asset' receives
  - **sellerReceiveAddress**: `hex-string` - the external chain address to receive the 'counter-asset'
  - **buyerSendAddress**: `hex-string` - if reserved (locked): the address the buyer will be transferring the funds from
  - **buyerChainDeadline**: `hex-string` - the external chain height deadline to send the 'tokens' to SellerReceiveAddress
  - **sellersSendAddress**: `hex-string` - the signing address of seller who is selling the CNPY
- **type**: `string` - the type of paginated results ("orders")
- **count**: `int` - the number of items in this page
- **totalPages**: `int` - the total number of pages available
- **totalCount**: `int` - the total number of orders matching the filters

**Example 1: Basic pagination**
```
$ curl -X POST localhost:50002/v1/query/orders \
  -H "Content-Type: application/json" \
  -d '{
        "pageNumber": 1,
        "perPage": 10
      }'

> {
    "pageNumber": 1,
    "perPage": 10,
    "results": [
        {
            "id": "abb1f314f5f300d315a56581ccb0f10fe1665f90c8f09666f7c58abcabfbcedb",
            "committee": 1,
            "amountForSale": 1000000000000,
            "requestedAmount": 2000000000000,
            "sellerReceiveAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
            "sellersSendAddress": "bb43c46244cef15f2451a446cea011fc1a2eddfe"
        },
        {
            "id": "ccd2f425f6f411e426b67692ddc1f21gf2776ga1d9g1a777g8d69bcdbacddfec",
            "committee": 1,
            "amountForSale": 500000000000,
            "requestedAmount": 1000000000000,
            "sellerReceiveAddress": "613d1c4e7dde2d7g275bb6647c3cb3dc0f91d822",
            "buyerSendAddress": "aaac0b3d64c12c6f164545545b2ba2ab4d80deff",
            "buyerChainDeadline": 17585,
            "sellersSendAddress": "cc54d57355dfg26g3562b557dfb122gd2b3feegh"
        }
    ],
    "type": "orders",
    "count": 2,
    "totalPages": 1,
    "totalCount": 2
}
```

**Example 2: Filter by committee with pagination**
```
$ curl -X POST localhost:50002/v1/query/orders \
  -H "Content-Type: application/json" \
  -d '{
        "committee": 1,
        "pageNumber": 1,
        "perPage": 20
      }'
```

**Example 3: Filter by sellersSendAddress with pagination (uses indexed lookup)**
```
$ curl -X POST localhost:50002/v1/query/orders \
  -H "Content-Type: application/json" \
  -d '{
        "sellersSendAddress": "bb43c46244cef15f2451a446cea011fc1a2eddfe",
        "pageNumber": 1,
        "perPage": 10
      }'
```

**Example 4: Filter by both committee and sellersSendAddress with pagination (most efficient)**
```
$ curl -X POST localhost:50002/v1/query/orders \
  -H "Content-Type: application/json" \
  -d '{
        "committee": 1,
        "sellersSendAddress": "bb43c46244cef15f2451a446cea011fc1a2eddfe",
        "pageNumber": 1,
        "perPage": 10
      }'
```

**Example 5: Filter by buyerSendAddress with pagination (locked orders only)**
```
$ curl -X POST localhost:50002/v1/query/orders \
  -H "Content-Type: application/json" \
  -d '{
        "buyerSendAddress": "aaac0b3d64c12c6f164545545b2ba2ab4d80deff",
        "pageNumber": 1,
        "perPage": 10
      }'
```

## Dex Batch
**Route:** `/v1/query/dex-batch`
**Description**: view the locked dex batch for a committee or all dex batches
**HTTP Method**: `POST`
**Request**:
- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **id**: `uint64` – the unique identifier of the committee (optional: use 0 to get all committees)
- **points**: `bool` - return all point holders for the liquidity pool and total points
**Response**:
- **committee**: `uint64` - the id of the 'counter asset'
- **receiptHash**: `hex-string` - the hash of the counter batch this locked batch corresponds to
- **orders**: `dex limit order array` - the list of dex limit orders
  - **amountForSale**: `uint64` - amount of asset for sale
  - **requestedAmount**: `uint64` - the minimum requested amount of 'counter-asset' to receive
  - **address**: `hex string` - the address where the funds are transferred from and to
  - **orderId**: `hex string` - unique identifier for the order (20 bytes, derived from transaction hash)
- **deposits**: `dex deposit array` - the list of dex limit orders
  - **amount**: `uint64` - amount of asset being deposited
  - **address**: `hex string` - the address where the funds are transferred from
  - **orderId**: `hex string` - unique identifier for the deposit (20 bytes, derived from transaction hash)
- **withdraws**: `dex withdraw array` - the list of dex limit orders
  - **percent**: `uint64` - the percent of liquidity being withdrawn
  - **address**: `hex string` - the address where the funds are transferred from
  - **orderId**: `hex string` - unique identifier for the withdrawal (20 bytes, derived from transaction hash)
- **poolPoints**: `array` - a list of pool points
  - **address**: `hex-string` - the recipient address of the points
  - **points**: `uint64` the amount of points owned
- **totalPoolPoints**: `uint64` - the sum of all pool points
- **poolSize**: `uint64` - contains the current balance of the liquidity pool
- **counterPoolSize**: `uint64` - the last computed 'counter pool' size of the liquidity pool on the counter chain
- **receipts**: `uint64 array` - the amount distributed (dY) for each order
```
$ curl -X POST localhost:50002/v1/query/dex-batch \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000,
        "id": 1
      }'
> {
    "committee": 1,
    "receiptHash": "b5d54c39e66671c9731b9f471e585d8262cd4f54963f0c93082d8dcf334d4c78",
    "orders": [
        {
            "amountForSale": 1000000000000,
            "requestedAmount": 2000000000000,
            "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
            "orderId": "a1b2c3d4e5f67890abcd"
        }
    ],
    "deposits": [
        {
            "amount": 1000000000000,
            "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
            "orderId": "1234567890abcdef1234"
        }
    ],
    "withdraws": [
        {
            "amount": 100,
            "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
            "orderId": "fedcba0987654321fedc"
        }
    ],
    "poolPoints": [
      {
        "address": "551f21e333012027b81701a35023efc88b864975",
        "points": 100
      }
    ],
    "totalPoolPoints": 100,
    "poolSize": 5000000000000,
    "counterPoolSize": 3000000000000,
    "receipts": [1000000, 0, 950000]
}
```

## Next Dex Batch
**Route:** `/v1/query/next-dex-batch`
**Description**: view the up-next dex batch for a committee or all upcoming dex batches
**HTTP Method**: `POST`
**Request**:
- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **id**: `uint64` – the unique identifier of the committee (optional: use 0 to get all committees)
  **Response**:
- **committee**: `uint64` - the id of the 'counter asset'
- **receiptHash**: `hex-string` - the hash of the counter batch this locked batch corresponds to
- **orders**: `dex limit order array` - the list of dex limit orders
  - **amountForSale**: `uint64` - amount of asset for sale
  - **requestedAmount**: `uint64` - the minimum requested amount of 'counter-asset' to receive
  - **address**: `hex string` - the address where the funds are transferred from and to
  - **orderId**: `hex string` - unique identifier for the order (20 bytes, derived from transaction hash)
- **deposits**: `dex deposit array` - the list of dex limit orders
  - **amount**: `uint64` - amount of asset being deposited
  - **address**: `hex string` - the address where the funds are transferred from
  - **orderId**: `hex string` - unique identifier for the deposit (20 bytes, derived from transaction hash)
- **withdraws**: `dex withdraw array` - the list of dex limit orders
  - **percent**: `uint64` - the percent of liquidity being withdrawn
  - **address**: `hex string` - the address where the funds are transferred from
  - **orderId**: `hex string` - unique identifier for the withdrawal (20 bytes, derived from transaction hash)
- **poolPoints**: `array` - a list of pool points
  - **address**: `hex-string` - the recipient address of the points
  - **points**: `uint64` the amount of points owned
- **totalPoolPoints**: `uint64` - the sum of all pool points
- **poolSize**: `uint64` - contains the current balance of the liquidity pool
- **counterPoolSize**: `uint64` - the last computed 'counter pool' size of the liquidity pool on the counter chain
- **receipts**: `uint64 array` - the amount distributed (dY) for each order
```
$ curl -X POST localhost:50002/v1/query/next-dex-batch \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000,
        "id": 1
      }'
> {
    "committee": 1,
    "receiptHash": "b5d54c39e66671c9731b9f471e585d8262cd4f54963f0c93082d8dcf334d4c78",
    "orders": [
        {
            "amountForSale": 1000000000000,
            "requestedAmount": 2000000000000,
            "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
            "orderId": "a1b2c3d4e5f67890abcd"
        }
    ],
    "deposits": [
        {
            "amount": 1000000000000,
            "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
            "orderId": "1234567890abcdef1234"
        }
    ],
    "withdraws": [
        {
            "amount": 100,
            "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
            "orderId": "fedcba0987654321fedc"
        }
    ],
    "poolPoints": [
      {
        "address": "551f21e333012027b81701a35023efc88b864975",
        "points": 100
      }
    ],
    "totalPoolPoints": 100,
    "poolSize": 5000000000000,
    "counterPoolSize": 3000000000000,
    "receipts": [1000000, 0, 950000]
}
```

## Dex Price
**Route:** `/v1/query/dex-price`
**Description**: retrieves the latest dex price for a committee or all dex prices
**HTTP Method**: `POST`
**Request**:
- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **id**: `uint64` – the unique identifier of the committee (optional: use 0 to get all committees)
**Response**:
- **chainId**: `uint64` - the id of the local chain
- **remoteChainId**: `uint64` - the id of the remote chain
- **localPool**: `uint64` - the current balance of the local liquidity pool
- **remotePool**: `uint64` - the current balance of the remote liquidity pool
- **e6ScaledPrice**: `uint64` - the price scaled by 1e6 (1,000,000)
```
$ curl -X POST localhost:50002/v1/query/dex-price \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000,
        "id": 1
      }'
> {
    "chainId": 1,
    "remoteChainId": 2,
    "localPool": 5000000000000,
    "remotePool": 3000000000000,
    "e6ScaledPrice": 600000
}
```

## Pending Transactions (Mempool)

**Route:** `/v1/query/pending`

**Description**: view the transactions not yet confirmed in a block

**HTTP Method**: `POST`

**Request**:

- **perPage**: `int` - the number of elements per page (the default is 10 and max is 5,000)
- **pageNumber**: `int` - the number of the page (the default is 1)

**Response**:
- **perPage**: `int` - the number of elements per page
- **pageNumber**: `int` - the number of the page
- **results**: `array` - the list of result objects
  - **blockHeader**: `object` - the summary of the block
  - **transactions**: `array` - list of transactions in the block
    - **sender**: `hex-string` - the address of the user sending the transaction
    - **recipient**: `hex-string` - the address of the user receiving the transaction
    - **messageType**: `string` - the name of the of the message like 'send' or 'stake'
    - **height**: `uint64` - the block height at which the transaction was included (n/a)
    - **index**: `uint64` - the position of the transaction within the block
    - **transaction**: `object` - original transaction object
      - **messageType**: `string` - type of the transaction like 'send' or 'stake'
      - **msg**: `object` - the actual transaction message payload, which is encapsulated in a generic message format. (See `tx-by-hash`)
      - **signature**: `object` - the cryptographic signature used to verify the authenticity of the transaction
      - **createdHeight**: `uint64` - the height when the transaction was created - allows 'safe pruning'
      - **time**: `unix micro timestamp` - timestamp when the transaction was created - used as temporal entropy to prevent hash collisions in txs
      - **fee**: `uint64` - fee associated with processing the transaction in micro denomination
      - **memo**: `string` - an optional message or note attached to the transaction
      - **networkId**: `uint64` - the identity of the network the transaction is intended for
      - **chainId**: `uint64` - the identity of the committee the transaction is intended for
    - **txHash**: `hex-string` - the unique hash that identifies the transaction
- **type**: `string` - the type of results
- **count**: `int` - length of results
- **totalPages**: `int` - number of pages
- **totalCount**: `int` - total number of items that exist in all pages


```
$ curl -X POST localhost:50002/v1/query/pending \
  -H "Content-Type: application/json" \
  -d '{}'

>{
  "pageNumber": 1,
  "perPage": 10,
  "results": [
    {
      "sender": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
      "recipient": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
      "messageType": "send",
      "height": 0,
      "transaction": {
        "type": "send",
        "msg": {
          "fromAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
          "toAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
          "amount": 100000
        },
        "signature": {
          "publicKey": "b2947db37385bb43c46244cef15f2451a446cea011fc1a2e1d52b1cecc7a50a8924e0e062555793bbd55a91a685017ee",
          "signature": "8a427223eae305e0816d66c7940320f7fa17b3a084c1a27130fd4cd0e6fe68eae2edecf1ec197cac1ac049899cdf9e0e10ae7855b2b2aacd213950eb4856dd10b6bc0b77eb34bc3e49e82cabb21aaf85f588876b0568f1b4981961c3e9c525a9"
        },
        "time": 1749070106734452,
        "createdHeight": 17584,
        "fee": 10000,
        "networkID": 1,
        "chainID": 1
      },
      "txHash": "ff22f214f5f300d315a56581ccb0f10fe1665f90c8f09666f7c58abcabfbcedb"
    }
  ],
  "type": "pending-results-page",
  "count": 1,
  "totalPages": 1,
  "totalCount": 1
}
```

## Failed Transactions

**Route:** `/v1/query/failed-txs`

**Description**: view the transactions that failed locally - note: there is no global index of failed transactions because Canopy does not include failed transactions in blocks

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the address of the sender of the failed transactions
- **perPage**: `int` - the number of elements per page (the default is 10 and max is 5,000)
- **pageNumber**: `int` - the number of the page (the default is 1)

**Response**:
- **perPage**: `int` - the number of elements per page
- **pageNumber**: `int` - the number of the page
- **results**: `array` - the list of result objects
  - **blockHeader**: `object` - the summary of the block
  - **transactions**: `array` - list of transactions in the block
    - **sender**: `hex-string` - the address of the user sending the transaction
    - **recipient**: `hex-string` - the address of the user receiving the transaction
    - **messageType**: `string` - the name of the of the message like 'send' or 'stake'
    - **height**: `uint64` - the block height at which the transaction was included (n/a)
    - **index**: `uint64` - the position of the transaction within the block
    - **transaction**: `object` - original transaction object
      - **messageType**: `string` - type of the transaction like 'send' or 'stake'
      - **msg**: `object` - the actual transaction message payload, which is encapsulated in a generic message format. (See `tx-by-hash`)
      - **signature**: `object` - the cryptographic signature used to verify the authenticity of the transaction
      - **createdHeight**: `uint64` - the height when the transaction was created - allows 'safe pruning'
      - **time**: `unix micro timestamp` - timestamp when the transaction was created - used as temporal entropy to prevent hash collisions in txs
      - **fee**: `uint64` - fee associated with processing the transaction in micro denomination
      - **memo**: `string` - an optional message or note attached to the transaction
      - **networkId**: `uint64` - the identity of the network the transaction is intended for
      - **chainId**: `uint64` - the identity of the committee the transaction is intended for
    - **txHash**: `hex-string` - the unique hash that identifies the transaction
  - **address**: `hex-string` - the sender of the transaction
  - **error**: `object` - the failure reason
    - **code**: `uint64` - the softare code of the failure
    - **module**: `string` - the software module of the failure
    - **msg**: `string` - the failure message
- **type**: `string` - the type of results
- **count**: `int` - length of results
- **totalPages**: `int` - number of pages
- **totalCount**: `int` - total number of items that exist in all pages


```
$ curl -X POST localhost:50002/v1/query/failed-txs \
  -H "Content-Type: application/json" \
  -d '{
    "address":"502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711"
  }'

>{
  "pageNumber": 1,
  "perPage": 10,
  "results": [
    {
      "transaction": {
        "type": "send",
        "msg": {
          "fromAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
          "toAddress": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
          "amount": 341230502138
        },
        "signature": {
          "publicKey": "b2947db37385bb43c46244cef15f2451a446cea011fc1a2e1d52b1cecc7a50a8924e0e062555793bbd55a91a685017ee",
          "signature": "892b2467b96f82c986eddd584852713bddb49910e8f8b4fcce49713722c6fed3441271e329c275f6994e0ba02cfefda50cf13ade98c22fd140a2f5e2b27db527b11aaa013fb7422db479c61ba9fd95098975701f7f692fb1c9b633f03a15e979"
        },
        "time": 1749075548254250,
        "createdHeight": 18666,
        "fee": 10000,
        "networkID": 1,
        "chainID": 1
      },
      "txHash": "dd94d433b9da628444fb790515ddfc0434934ed7da9aca6793995f87691e8439",
      "address": "502c0b3d6ccd1c6f164aa5536b2ba2cb9e80c711",
      "error": {
        "code": 28,
        "module": "state_machine",
        "msg": "insufficient funds"
      }
    }
  ],
  "type": "failed-txs-page",
  "count": 1,
  "totalPages": 1,
  "totalCount": 1
}
```


## Governance Proposals

**Route:** `/v1/gov/proposals`

**Description**: view how your validator is voting for governance parameters changes and token distributions from the DAO Treasury Fund

**HTTP Method**: `GET`

**Request**: None

**Response**:
- `map[proposalHashHex] -> object`
  - **proposal**: `object` - the governance proposal structure in the form of a Transaction with a payload that is `oneOf`: `MessageChangeParameter` or `MessageDAOTransfer` see (see tx-by-hash)
  - **approve**: `bool` - is the local Validator voting yes or no on the proposal


```
$ curl -X GET localhost:50002/v1/gov/proposals

>{
  "93ba7a69f9be05257822c61be59bd80c1d5c5c9f56d6320f94a36d84c4c781f9": {
    "proposal": {
      "type": "changeParameter",
      "msg": {
        "parameterSpace": "cons|fee|val|gov",
        "parameterKey": "protocolVersion",
        "parameterValue": "example",
        "startHeight": 1,
        "endHeight": 1000,
        "signer": "4646464646464646464646464646464646464646"
      },
      "signature": {
        "publicKey": "a10665940f6dd5f8c5ca858b562afee13794a5055ffc289596b4fe9230aae83bd7e8b4dc1c9611793936d6dfd725e592",
        "signature": "8c586257852296498c056d9ea8090f69207d938d67375ea34fbc2af4fa06e97404588d3f67237c3bf847fe8c6d7a670d03680622aba6b3b7e72d3bc93f871bf98ba8a2e94d1ab6e52437746a0449bc2fbd9ccf42e8256f5ab541b95435f6c202"
      },
      "time": 1744837549307074,
      "createdHeight": 1,
      "fee": 10000,
      "memo": "example",
      "networkID": 1,
      "chainID": 1
    },
    "approve": true
  }
}
```

## Governance Poll

**Route:** `/v1/gov/poll`

**Description**: view community sentiment by on-chain votes of Accounts and Validators. This data is recalculated and updated periodically

**HTTP Method**: `GET`

**Request**: None

**Response**:
- `map[proposalHashHex] -> object`
  - **proposalHash**: `hex-string` - the SHA256 hash of the proposal that is voted on
  - **proposalURL**: `url-string` - the URL of the proposal information
  - **accounts**: `object` - the accounts voting data
    - **approveTokens**: `uint64` - approve count represented by liquid token balance
    - **rejectTokens**: `uint64` - reject count reprented by liquid token balance
    - **totalTokens**: `uint64` - total liquid tokens that voted
    - **approvePercent**: `uint64` - the percent of voters in this category who approve
    - **rejectPercent**: `uint64` - the percent of voters in this category who disapprove
    - **votedPercent**: `uint64` - the percent of voters in this category who voted
  - **validators**: `object` - the validators voting data
    - **approveTokens**: `uint64` - approve count represented by liquid token balance
    - **rejectTokens**: `uint64` - reject count reprented by liquid token balance
    - **totalTokens**: `uint64` - total liquid tokens that voted
    - **approvePercent**: `uint64` - the percent of voters in this category who approve
    - **rejectPercent**: `uint64` - the percent of voters in this category who disapprove
    - **votedPercent**: `uint64` - the percent of voters in this category who voted
  - **approve**: `bool` - is the local Validator voting yes or no on the proposal


```
$ curl -X GET localhost:50002/v1/gov/poll

>{
  "50d858e0985ecc7f60418aaf0cc5ab587f42c2570a884095a9e8ccacd0f6545c": {
    "proposalHash": "50d858e0985ecc7f60418aaf0cc5ab587f42c2570a884095a9e8ccacd0f6545c",
    "proposalURL": "https://forum.cnpy.network/something",
    "accounts": {
      "approveTokens": 0,
      "rejectTokens": 0,
      "totalVotedTokens": 0,
      "totalTokens": 12556191411824,
      "approvedPercent": 0,
      "rejectPercent": 0,
      "votedPercent": 0
    },
    "validators": {
      "approveTokens": 0,
      "rejectTokens": 0,
      "totalVotedTokens": 0,
      "totalTokens": 2223557946105,
      "approvedPercent": 0,
      "rejectPercent": 0,
      "votedPercent": 0
    }
  }
}
```

## Root Chain Info

**Route:** `/v1/gov/root-chain-info`

**Description**: responds with root-chain data needed for nested-chain consensus

**HTTP Method**: `POST`

**Request**:
- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **id**: `uint64` – the unique identifier of the committee

**Response**:
- **rootChainid**: `uint64` - the unique identifier of the chain responsible for the validator set
- **height**: `uint64` - the height of the query
- **validatorSet**: `object` - the consensus validators currently active according to the root-chain
  - **validatorSet**: `object array` - the list of consensus validators
    - **publicKey**: `hex-string` - the BLS public key of the validator
    - **votingPower**: `uint64` - the voting power of the validator (typically amount of staked tokens)
    - **netAddress**: `url string` - the TCP peer-to-peer address of the validator
- **lastValidatorSet**: `object` - the consensus validators active for the last root-chain height (see validatorSet)
- **lotteryWinner**: `object` - the selected delegate/pseudo-validator who receives rewards
  - **winner**: `hex-string` - the 20 byte address of the selected actor
  - **cut**: `uint64` - the percent cut of the rewards before normalization
- **orders**: `object` - the swap order book from the 'root chain' for the 'nested chain'
  - **chainId**: `uint64` - the unique identifier of the committee
  - **orders**: `sell order array` - the actual list of sell orders
    - **id**: `hex string` - the 20 byte identifier of the order
    - **committee**: `uint64` - the id of the committee that is in-charge of escrow for the swap
    - **data**: `hex-string` - a generic data field which can allow a committee to execute specific functionality for the swap
    - **amountForSale**: `uint64` - amount of 'root-chain-asset' for sale
    - **requestedAmount**: `uint64` - amount of 'counter-asset' the seller of the 'root-chain-asset' receives
    - **sellerReceiveAddress**: `hex-string` - the external chain address to receive the 'counter-asset'
    - **buyerSendAddress**: `hex-string` - if reserved (locked): the address the buyer will be transferring the funds from
    - **buyerChainDeadline**: `hex-string` - the external chain height deadline to send the 'tokens' to SellerReceiveAddress
    - **sellersSendAddress**: `hex-string` - the signing address of seller who is selling the CNPY


```
$ curl -X POST localhost:50002/v1/query/root-chain-info \
  -H "Content-Type: application/json" \
  -d '{
        "height": 17585,
        "id": 1
      }'

>{
  "rootChainId": 1,
  "height": 17585,
  "validatorSet": {
    "validatorSet": [
      {
        "publicKey": "b17d4eb3938957e710bacc9f09d2a9aa79a568fcdf1f8fc565bdb5de3f334295e929e4b086b9c8e9610654155fb0452b",
        "votingPower": 1,
        "netAddress": "tcp://canopyrocks.xyz"
      }
    ]
  },
  "lastValidatorSet": {
    "validatorSet": [
      {
        "publicKey": "b17d4eb3938957e710bacc9f09d2a9aa79a568fcdf1f8fc565bdb5de3f334295e929e4b086b9c8e9610654155fb0452b",
        "votingPower": 1,
        "netAddress": "tcp://canopyrocks.xyz"
      }
    ]
  },
  "lotteryWinner": {
    "winner": "",
    "cut": 0
  },
  "orders": {
    "chainID": 1
  }
}
```

## Last Proposers

**Route:** `/v1/query/last-proposers`

**Description**: responds with the last Proposer addresses saved in the state

**HTTP Method**: `POST`

**Request**:
- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)

**Response**:
- **addresses**: `hex-string array` - a list of addresses of the most recent previous proposers in fixed modulo order

```
$ curl -X POST localhost:50002/v1/query/last-proposers \
  -H "Content-Type: application/json" \
  -d '{
        "height": 17585
      }'

>{
  "addresses": [
    "fc5cdb5c0b6a6df41b92976bbdf2b6832855446f",
    "ac2c2eb9aa04b99ec9a523b09ca844a77826c291",
    "b9c6c2dfa9d049e480c8cec9c29463abf078a594",
    "b9c6c2dfa9d049e480c8cec9c29463abf078a594",
    "33e14ef6b87fb688b829c5e29618bb549dc7b4cd"
  ]
}
```

## Is Valid Double Signer

**Route:** `/v1/query/valid-double-signer`

**Description**: responds if the DoubleSigner is NOT already set for a height - this is useful to determine if byzantine evidence is stale.

**HTTP Method**: `POST`

**Request**:
- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **address**: `hex-string` - the address of the potential double signer

**Response**: `bool` - may be set as a double signer

```
$ curl -X POST localhost:50002/v1/query/valid-double-signer \
  -H "Content-Type: application/json" \
  -d '{
        "height": 17585,
        "address": "b9c6c2dfa9d049e480c8cec9c29463abf078a594"
      }'

>true
```

## Double Signer

**Route:** `/v1/query/double-signers`

**Description**: responds with all double signers in the indexer

**HTTP Method**: `POST`

**Request**:
- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)

**Response**:
- `array` - list of all double signers as of the query height
  - **id**: `uint64` - the address identifier of the malicious actor
  - **heights**: `array`

```
$ curl -X POST localhost:50002/v1/query/double-signers \
  -H "Content-Type: application/json" \
  -d '{
        "height": 17585
      }'

>[{
  "id": "b9c6c2dfa9d049e480c8cec9c29463abf078a594",
  "heights": [1000, 17585]
}]
```

## Minimum Evidence Height

**Route:** `/v1/query/minimum-evidence-height`

**Description**: responds with the minimum height the evidence must be to still be usable

**HTTP Method**: `POST`

**Request**: `None`

**Response**: `uint64` - the minimum evidence height

```
$ curl -X POST localhost:50002/v1/query/minimum-evidence-height \
  -H "Content-Type: application/json" \
  -d '{}'

> 165453
```

## Lottery

**Route:** `/v1/query/lottery`

**Description**: responds with the selected winner (nested-validator/nested-delegate/delegate) chosen pseudo-randomly based on their stake weight within a committee

**HTTP Method**: `POST`

**Request**:
- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **id**: `uint64` - the unique identifier of the committee

**Response**:
- **winner**: `hex-string` - the 20 byte address of the selected actor
- **cut**: `uint64` - the percent cut of the rewards before normalization

```
$ curl -X POST localhost:50002/v1/query/lottery \
  -H "Content-Type: application/json" \
  -d '{
        "height": 17585,
        "id" : 1
      }'

>{
  "winner": "b9c6c2dfa9d049e480c8cec9c29463abf078a594",
  "cut": 10
}
```

## Checkpoint

**Route:** `/v1/query/checkpoint`

**Description**: responds with the checkpoint block hash for a certain committee and height combination

**HTTP Method**: `POST`

**Request**:
- **height**: `uint64` – the block height to read data from (optional: use 0 to read from the latest block)
- **id**: `uint64` - the unique identifier of the committee / chain

**Response**: `hex-string` - the block hash of the committee / chain

```
$ curl -X POST localhost:50002/v1/query/checkpoint \
  -H "Content-Type: application/json" \
  -d '{
        "height": 1000,
        "id" : 1
      }'

> "cd9d3e487bce2918e306364fca286f473bd8cc92f150b3d97f9063b570a2b801"
```


## Subscribe Root Chain Info

**Route:** `/v1/query/subscribe-rc-info`

**Description**: ws: upgrades a http request to a websockets connection. Subscribes to the root chain info for when the root-chain publishes it.

**HTTP Method**: `WS`

**Request**:
- **chainId**: `uint64` - the unique identifier of the committee / chain

**Response**: Periodic updates with Root-Chain-Info sent over the WebSocket

```
$ websocat "ws://localhost:50002/v1/subscribe-rc-info?chainId=123"
```

## Ethereum

**Route:** `/v1/eth`

**Description**: implements [Ethereum JSON-RPC interface](https://ethereum.org/en/developers/docs/apis/json-rpc). For more details see the `Canopy Ethereum Wrapper` specification.

**HTTP Method**: `POST`

**Request**: (See Ethereum Specification)

**Response**: (See Ethereum Specification)

```
$ curl -X POST http://localhost:50002/v1/eth \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc":"2.0",
    "method":"eth_blockNumber",
    "params":[],
    "id":1
  }'

> {
  "id": 1,
  "jsonrpc": "2.0",
  "result": "0x2fc6c"
}
```

# Admin

🚨**Important: All admin commands assume secure https connection**

## Keystore

**Route:** `/v1/admin/keystore`

**Description**: responds with the local keystore

**HTTP Method**: `GET`

**Request**: `None`

**Response**:
- **addressMap**: `object` - the map of addresses to keystore entry `[hex-address] -> keystoreEntry`
  - **publicKey**: `hex-string` -  the public code that can cryptographically verify signatures from the private key
  - **salt**: `hex-string` - random 16 bytes salt
  - **encrypted**: `hex-string` - the private key encrypted with AES-GCM
  - **keyAddress**: `hex-string` - the address of the key
  - **keyNickname**: `string` - the nickname of the key
- **nicknameMap**: `object` the map of nicknames to address `[nickname] -> hex-address`


```
$ curl -X GET http://localhost:50003/v1/admin/keystore \
  -H "Content-Type: application/json" \
  -d '{}'

> {
  "addressMap": {
    "b0b4a45ca70104ecc943a49e4553f0e7e1135b01": {
      "publicKey": "8cb57bdcc0f90c36aea48a6edab31996400af55f623f676a3c64c04b826454aea227f14d1b70c150d5fa9fe5f6790f60",
      "salt": "aff12414025b261cc56fbeaeb5dcf5f4",
      "encrypted": "eb3d900e781a2de743bcc804737753cbe7440bffab18419409517db0a2bf98b8028f69aba271f482b838c6de8e8b8292",
      "keyAddress": "b0b4a45ca70104ecc943a49e4553f0e7e1135b01",
      "keyNickname": "my_key"
    }
  },
  "nicknameMap": {
    "my_key": "b0b4a45ca70104ecc943a49e4553f0e7e1135b01",
  }
}
```

## Keystore New Key

**Route:** `/v1/admin/keystore-new-key`

**Description**: create a new BLS key in the keystore

**HTTP Method**: `POST`

**Request**:
- **nickname**: `string` - the nickname associated with the key
- **password**: `string` - the plaintext password used to encrypt the key

**Response**: `hex-string` - the 20 byte address of the newly created key


```
$ curl -X POST http://localhost:50003/v1/admin/keystore-new-key \
  -H "Content-Type: application/json" \
  -d '{"nickname":"my_key_", "password":"plain-text"}'

> "7cd2a86894f4a23ea956dee1722c6776230bf552"
```


## Keystore Import

**Route:** `/v1/admin/keystore-import`

**Description**: import an encrypted private key to the node's keystore

**HTTP Method**: `POST`

**Request**:
- **nickname**: `string` - the nickname associated with the key
- **address**: `hex-string` - the address associated with the encrypted key
- **publicKey**: `hex-string` -  the public code that can cryptographically verify signatures from the private key
- **salt**: `hex-string` - random 16 bytes salt
- **encrypted**: `hex-string` - the private key encrypted with AES-GCM

**Response**: `hex-string` - the 20 byte address of the newly imported key


```
$ curl -X POST http://localhost:50003/v1/admin/keystore-import \
  -H "Content-Type: application/json" \
  -d '{
    "nickname":"my_key_", 
    "addresss":"b0b4a45ca70104ecc943a49e4553f0e7e1135b01",
    "publicKey": "8cb57bdcc0f90c36aea48a6edab31996400af55f623f676a3c64c04b826454aea227f14d1b70c150d5fa9fe5f6790f60",
      "salt": "aff12414025b261cc56fbeaeb5dcf5f4",
      "encrypted": "eb3d900e781a2de743bcc804737753cbe7440bffab18419409517db0a2bf98b8028f69aba271f482b838c6de8e8b8292"
    }'

> "b0b4a45ca70104ecc943a49e4553f0e7e1135b01"
```

## Keystore Import Raw

**Route:** `/v1/admin/keystore-import-raw`

**Description**: import a raw private key to the node's keystore

**HTTP Method**: `POST`

**Request**:
- **nickname**: `string` - the nickname associated with the key
- **password**: `string` - the plain-text password to encrypt the private key with
- **privateKey**: `hex-string` -  the secret code that is capable of producing digital signatures

**Response**: `hex-string` - the 20 byte address of the newly imported key


```
$ curl -X POST http://localhost:50003/v1/admin/keystore-import-raw \
  -H "Content-Type: application/json" \
  -d '{
    "nickname":"my_key_2", 
    "password":"plain-text",
    "privateKey": "ee62bbab22a26e34e5489bdb3a751533c34084975fadac4b5b23e15dbd0cff70"
    }'

> "b0b4a45ca70104ecc943a49e4553f0e7e1135b01"
```

## Keystore Delete

**Route:** `/v1/admin/keystore-delete`

**Description**: removes a key from the keystore using either the address or nickname

**HTTP Method**: `POST`

**Request**:
- **nickname**: `string` - the nickname associated with the key
- **address**: `string` - the address associated with the key

**Response**: `hex-string` - the 20 byte address of the newly imported key


```
$ curl -X POST http://localhost:50003/v1/admin/keystore-delete \
  -H "Content-Type: application/json" \
  -d '{
    "nickname":"my_key_2"
    }'

> "b0b4a45ca70104ecc943a49e4553f0e7e1135b01"
```

## Keystore Get

**Route:** `/v1/admin/keystore-get`

**Description**: responds with the key group associated with an address or nickname

**HTTP Method**: `POST`

**Request**:
- **nickname**: `string` - the nickname associated with the key
- **address**: `hex-string` - address associated with the key
- **password**: `hex-string` - the plain-text password to get the private key with

**Response**:
- **address**: `hex-string` - the 20 byte address of the newly imported key
- **publicKey**: `hex-string` - a cryptographic code shared openly, used to verify digital signatures of its paired private key
- **privateKey**: `hex-string` -  a secret cryptographic code that is used to produce digital signatures


```
$ curl -X POST http://localhost:50003/v1/admin/keystore-get \
  -H "Content-Type: application/json" \
  -d '{
    "nickname":"my_key",
    "password":"test"
    }'

> {
  "address": "b0b4a45ca70104ecc943a49e4553f0e7e1135b01",
  "publicKey": "acb57b23c0f90c16aea48a6edbb31996400af55f623f676a3c64c04b826454aea227f14d1b70c151d5fa9fe536790f6d",
  "privateKey": "ed3c9acd8fffa374756ba1c175af5228f9e4e96c0ecdfda0a9fd9afa09472e12"
}
```

## Txn Send

**Route:** `/v1/admin/tx-send`

**Description**: generates/submits a send transaction

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the from address
- **output**: `hex-string` - the recipient address
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **memo**: `string` - an arbitrary message encoded in the transaction
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction

**Response**: (See tx-by-hash and MessageSend)

```
$ curl -X POST http://localhost:50003/v1/admin/tx-send \
  -H "Content-Type: application/json" \
  -d '{
    "address":"b0b4a45ca70104ecc943a49e4553f0e7e1135b01",
    "output":"271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5",
    "amount": 1000000,
    "memo": "hello world",
    "submit": false,
    "password": "test"
    }'

> {
  "type": "send",
  "msg": {
  "fromAddress": "b0b4a45ca70104ecc943a49e4553f0e7e1135b01",
  "toAddress": "271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5",
  "amount": 1000000
  },
  "signature": {
  "publicKey": "8cb57bdcc0f90c36aea48a6edab31996400af55f623f676a3c64c04b826454aea227f14d1b70c150d5fa9fe5f6790f60",
  "signature": "8b33f205f3cfeae277ac9b363a442114117d67dfa2722c9396bd1f67f289a977b0495c252db9a693c8e4c659667461ea0e9928533f69376622572808b10a17bd9c005ccea3f798a138719f896b1d27e981d88ef0cb025a761f6e37a5a6fd6e49"
  },
  "time": 1749415924948935,
  "createdHeight": 196596,
  "fee": 10000,
  "memo": "hello world",
  "networkID": 1,
  "chainID": 1
}
```

## Txn Stake

**Route:** `/v1/admin/tx-stake`

**Description**: generates/submits a stake transaction

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the operator address associated with the validator
- **amount**: `uint64` - the amount of tokens to stake in smallest (micro) denomination
- **pubKey**: `hex-string` - the operator public key (must be BLS if non-delegate)
- **netAddress**: `url string` - the p2p url of the validator (n/a for delegate)
- **committees**: `string` - a comma separated list of committee ids
- **delegate**: `bool` - is this validator active (false) or delegating (true)
- **earlyWithdrawal**: `bool` - is this validator withdrawing its rewards early for a penalty (true) or auto-compounding (false)
- **output**: `hex-string` - the address where rewards and returned bonded tokens outputs to
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **memo**: `string` - an arbitrary message encoded in the transaction
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction
- **signer**: `hex-string` - the address associated with the key that is signing the transaction

**Response**: (See tx-by-hash and MessageStake)

```
$ curl -X POST http://localhost:50003/v1/admin/tx-stake \
  -H "Content-Type: application/json" \
  -d '{
    "address": "abb4a45ca70104ecc943a49e4553f0e7e1135b01",
    "netAddress": "tcp://cnpyrocks.com",
    "committees": "1,2,3",
    "amount": 1000000,
    "delegate": false,
    "earlyWithdrawal": false,
    "output": "ab1e0120ac7f11a6f60baa24b2b187eaf1e2e6f5",
    "signer": "b0b4a45ca70104ecc943a49e4553f0e7e1135b01",
    "memo": "hello world",
    "fee": 10000,
    "submit": false,
    "password": "test"
  }'

> {
  "type": "stake",
  "msg": {
    "publickey": "8cb57bdcc0f90c36aea48a6edab31996400af55f623f676a3c64c04b826454aea227f14d1b70c150d5fa9fe5f6790f60",
    "amount": 1000000,
    "committees": [
      1,2,3
    ],
    "netAddress": "tcp://cnpyrocks.com",
    "outputAddress": "ab1e0120ac7f11a6f60baa24b2b187eaf1e2e6f5",
    "delegate": false,
    "compound": true
  },
  "signature": {
    "publicKey": "8cb57bdcc0f90c36aea48a6edab31996400af55f623f676a3c64c04b826454aea227f14d1b70c150d5fa9fe5f6790f60",
    "signature": "a0473e839b23abc7d8448de5c58141827d86157263deaa79db0d9a1d1f0ffa9f7503c9ff52be7645b3ca0de50482651c1857d2af39c6953a68e386ca0901f668ef61dd706e4ed753548b5cb6737aceb06307e5894fbe49668fa2a1ff9036e971"
  },
  "time": 1749416582662716,
  "createdHeight": 196596,
  "fee": 10000,
  "memo": "hello world",
  "networkID": 1,
  "chainID": 1
}
```

## Txn Edit Stake

**Route:** `/v1/admin/tx-edit-stake`

**Description**: generates/submits an edit stake transaction

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the operator address associated with the validator
- **amount**: `uint64` - the amount of tokens to stake in smallest (micro) denomination - (cannot decrease amount, lower defaults to current stake amount)
- **netAddress**: `url string` - the p2p url of the validator (n/a for delegate)
- **committees**: `string` - a comma separated list of committee ids
- **earlyWithdrawal**: `bool` - is this validator withdrawing its rewards early for a penalty (true) or auto-compounding (false)
- **output**: `hex-string` - the address where rewards and returned bonded tokens outputs to (only the output address can change this)
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **memo**: `string` - an arbitrary message encoded in the transaction
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction
- **signer**: `hex-string` - the address associated with the key that is signing the transaction

**Response**: (See tx-by-hash and MessageEditStake)

```
$ curl -X POST http://localhost:50003/v1/admin/tx-edit-stake \
  -H "Content-Type: application/json" \
  -d '{
    "address": "abb4a45ca70104ecc943a49e4553f0e7e1135b01",
    "netAddress": "tcp://cnpyrocks.com",
    "committees": "1,2,3",
    "amount": 1000000,
    "earlyWithdrawal": false,
    "output": "ab1e0120ac7f11a6f60baa24b2b187eaf1e2e6f5",
    "signer": "b0b4a45ca70104ecc943a49e4553f0e7e1135b01",
    "memo": "hello world",
    "fee": 10000,
    "submit": false,
    "password": "test"
  }'

> {
  "type": "edit-stake",
  "msg": {
    "addresss": "abb4a45ca70104ecc943a49e4553f0e7e1135b01",
    "amount": 1000000,
    "committees": [
      1,2,3
    ],
    "netAddress": "tcp://cnpyrocks.com",
    "outputAddress": "ab1e0120ac7f11a6f60baa24b2b187eaf1e2e6f5",
    "compound": true
  },
  "signature": {
    "publicKey": "8cb57bdcc0f90c36aea48a6edab31996400af55f623f676a3c64c04b826454aea227f14d1b70c150d5fa9fe5f6790f60",
    "signature": "a0473e839b23abc7d8448de5c58141827d86157263deaa79db0d9a1d1f0ffa9f7503c9ff52be7645b3ca0de50482651c1857d2af39c6953a68e386ca0901f668ef61dd706e4ed753548b5cb6737aceb06307e5894fbe49668fa2a1ff9036e971"
  },
  "time": 1749416582662716,
  "createdHeight": 196596,
  "fee": 10000,
  "memo": "hello world",
  "networkID": 1,
  "chainID": 1
}
```

## Txn Pause

**Route:** `/v1/admin/tx-pause`

**Description**: generates/submits a pause transaction

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the operator address associated with the validator
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **memo**: `string` - an arbitrary message encoded in the transaction
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction
- **signer**: `hex-string` - the address associated with the key that is signing the transaction

**Response**: (See tx-by-hash and MessagePause)

```
$ curl -X POST http://localhost:50003/v1/admin/tx-pause \
  -H "Content-Type: application/json" \
  -d '{
    "address": "abb4a45ca70104ecc943a49e4553f0e7e1135b01",
    "signer": "b0b4a45ca70104ecc943a49e4553f0e7e1135b01",
    "memo": "hello world",
    "fee": 10000,
    "submit": false,
    "password": "test"
  }'

> {
  "type": "pause",
  "msg": {
    "addresss": "abb4a45ca70104ecc943a49e4553f0e7e1135b01",
  "signature": {
    "publicKey": "8cb57bdcc0f90c36aea48a6edab31996400af55f623f676a3c64c04b826454aea227f14d1b70c150d5fa9fe5f6790f60",
    "signature": "a0473e839b23abc7d8448de5c58141827d86157263deaa79db0d9a1d1f0ffa9f7503c9ff52be7645b3ca0de50482651c1857d2af39c6953a68e386ca0901f668ef61dd706e4ed753548b5cb6737aceb06307e5894fbe49668fa2a1ff9036e971"
  },
  "time": 1749416582662716,
  "createdHeight": 196596,
  "fee": 10000,
  "memo": "hello world",
  "networkID": 1,
  "chainID": 1
}
```


## Txn Unpause

**Route:** `/v1/admin/tx-unpause`

**Description**: generates/submits an unpause transaction

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the operator address associated with the validator
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **memo**: `string` - an arbitrary message encoded in the transaction
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction
- **signer**: `hex-string` - the address associated with the key that is signing the transaction

**Response**: (See tx-by-hash and MessageUnpause)

```
$ curl -X POST http://localhost:50003/v1/admin/tx-unpause \
  -H "Content-Type: application/json" \
  -d '{
    "address": "abb4a45ca70104ecc943a49e4553f0e7e1135b01",
    "signer": "b0b4a45ca70104ecc943a49e4553f0e7e1135b01",
    "memo": "hello world",
    "fee": 10000,
    "submit": false,
    "password": "test"
  }'

> {
  "type": "unpause",
  "msg": {
    "addresss": "abb4a45ca70104ecc943a49e4553f0e7e1135b01",
  "signature": {
    "publicKey": "8cb57bdcc0f90c36aea48a6edab31996400af55f623f676a3c64c04b826454aea227f14d1b70c150d5fa9fe5f6790f60",
    "signature": "a0473e839b23abc7d8448de5c58141827d86157263deaa79db0d9a1d1f0ffa9f7503c9ff52be7645b3ca0de50482651c1857d2af39c6953a68e386ca0901f668ef61dd706e4ed753548b5cb6737aceb06307e5894fbe49668fa2a1ff9036e971"
  },
  "time": 1749416582662716,
  "createdHeight": 196596,
  "fee": 10000,
  "memo": "hello world",
  "networkID": 1,
  "chainID": 1
}
```

## Txn Unstake

**Route:** `/v1/admin/tx-unstake`

**Description**: generates/submits an unstake transaction

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the operator address associated with the validator
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **memo**: `string` - an arbitrary message encoded in the transaction
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction
- **signer**: `hex-string` - the address associated with the key that is signing the transaction

**Response**: (See tx-by-hash and MessageUnstake)

```
$ curl -X POST http://localhost:50003/v1/admin/tx-unstake \
  -H "Content-Type: application/json" \
  -d '{
    "address": "abb4a45ca70104ecc943a49e4553f0e7e1135b01",
    "signer": "b0b4a45ca70104ecc943a49e4553f0e7e1135b01",
    "memo": "hello world",
    "fee": 10000,
    "submit": false,
    "password": "test"
  }'

> {
  "type": "unstake",
  "msg": {
    "addresss": "abb4a45ca70104ecc943a49e4553f0e7e1135b01",
  "signature": {
    "publicKey": "8cb57bdcc0f90c36aea48a6edab31996400af55f623f676a3c64c04b826454aea227f14d1b70c150d5fa9fe5f6790f60",
    "signature": "a0473e839b23abc7d8448de5c58141827d86157263deaa79db0d9a1d1f0ffa9f7503c9ff52be7645b3ca0de50482651c1857d2af39c6953a68e386ca0901f668ef61dd706e4ed753548b5cb6737aceb06307e5894fbe49668fa2a1ff9036e971"
  },
  "time": 1749416582662716,
  "createdHeight": 196596,
  "fee": 10000,
  "memo": "hello world",
  "networkID": 1,
  "chainID": 1
}
```

## Txn Create Order

**Route:** `/v1/admin/tx-create-order`

**Description**: generates/submits a create (sell) order transaction

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the address that is sending the 'root-chain' funds
- **receiveAddress**: `hex-string` - the address that is receiving the 'counter-asset' funds
- **committees**: `string` - the id of the committee that is responsible for the 'counter asset'
- **amount**: `uint64` - the amount of 'root-chain-asset' to lock in escrow to be swapped in smallest denomination
- **receiveAmount**: `uint64` - the amount of 'counter-asset' to receive in smallest denomination
- **data**: `hex-string` - an arbitrary string code associated with the order (can be used for sub-asset contract address)
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **memo**: `string` - an arbitrary message encoded in the transaction
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction

**Response**: (See tx-by-hash and MessageCreateOrder)

```
$ curl -X POST http://localhost:50003/v1/admin/tx-create-order \
  -H "Content-Type: application/json" \
  -d '{
    "address":"abb4a45ca70104ecc943a49e4553f0e7e1135b01",
    "committees":"1",
    "amount":100000000,
    "receiveAmount":100000000,
    "data": "0x...",
    "receiveAddress":"271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5",
    "memo":"",
    "fee":10000,
    "submit":false,
    "password":"test"
    }'

> {
  "type": "createOrder",
  "msg": {
    "chainId": 1,
    "amountForSale": 100000000,
    "data": "0x...",
    "requestedAmount": 100000000,
    "sellerReceiveAddress": "271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5",
    "sellersSendAddress": "abb4a45ca70104ecc943a49e4553f0e7e1135b01"
  },
  "signature": {
    "publicKey": "83e91c8cf692365efd9a99a5efbd0afcc3d93a1e88e9bfe7d5219f9f5cf50cb785dd8c9727a1618a92100e28d47f7bf1",
    "signature": "b4aaccade020f47abef8b3513bb2967aadf5a0b7a5bdfb238d91a6c56c01d6911c1dcc1c0158e58eca6cc006a3f3e3750f3ddc9a3fa8a5091c9e3a5a7cd02e40d632d57747c8fca3f2b46b8c1415e54d4bc8e367e4b78ad5a25403571f3e8a6f"
  },
  "time": 1749643857253846,
  "createdHeight": 196596,
  "fee": 10000,
  "networkID": 1,
  "chainID": 1
}
```
## Txn Edit Order

**Route:** `/v1/admin/tx-edit-order`

**Description**: generates/submits an edit (sell) order transaction. Note: can only go through. if order is not yet 'locked' by a buyer

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the address that originally created the sell order
- **receiveAddress**: `hex-string` - the address that is receiving the 'counter-asset' funds
- **committees**: `string` - the id of the committee that is responsible for the 'counter asset'
- **orderId**: `hex-string` - the unique id of the sell-order
- **amount**: `uint64` - the amount of 'root-chain-asset' to lock in escrow to be swapped in smallest denomination
- **receiveAmount**: `uint64` - the amount of 'counter-asset' to receive in smallest denomination
- **data**: `hex-string` - an arbitrary string code associated with the order (can be used for sub-asset contract address)
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **memo**: `string` - an arbitrary message encoded in the transaction
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction

**Response**: (See tx-by-hash and MessageEditOrder)

```
$ curl -X POST http://localhost:50003/v1/admin/tx-edit-order \
  -H "Content-Type: application/json" \
  -d '{
    "address":"abb4a45ca70104ecc943a49e4553f0e7e1135b01",
    "committees":"1",
    "orderId":"1516f1cdd23f7e9f89e13a96ddf86351775f130f",
    "amount":100000000,
    "data": "0x...",
    "receiveAmount":101000000,
    "receiveAddress":"271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5",
    "memo":"",
    "fee":10000,
    "submit":false,
    "password":"test"
    }'

> {
  "type": "editOrder",
  "msg": {
    "orderID": "1516f1cdd23f7e9f89e13a96ddf86351775f130f",
    "chainID": 1,
    "data": "0x...",
    "amountForSale": 100000000,
    "requestedAmount": 101000000,
    "sellerReceiveAddress": "271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5"
  },
  "signature": {
    "publicKey": "83e91c8cf692365efd9a99a5efbd0afcc3d93a1e88e9bfe7d5219f9f5cf50cb785dd8c9727a1618a92100e28d47f7bf1",
    "signature": "a2e993ba9d1516f1cdd23f7e9f89e13a96ddf86351775f130f179495a0c15116c3aa786f513c4efe9471412865c93541089dd330c8064be5c34772140e46672b8d58df086b908cfa6847bdb6efe37dae3e3ed7872ddd28fc9e20b1f2dd7edb70"
  },
  "time": 1749644414876391,
  "createdHeight": 196596,
  "fee": 10000,
  "networkID": 1,
  "chainID": 1
}
```
## Txn Delete Order

**Route:** `/v1/admin/tx-delete-order`

**Description**: generates/submits a delete (sell) order transaction. Note: can only go through. if order is not yet 'locked' by a buyer

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the address that originally created the sell order
- **committees**: `string` - the id of the committee that is responsible for the 'counter asset'
- **orderId**: `hex-string` - the unique id of the sell-order
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **memo**: `string` - an arbitrary message encoded in the transaction
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction

**Response**: (See tx-by-hash and MessageDeleteOrder)

```
$ curl -X POST http://localhost:50003/v1/admin/tx-delete-order \
  -H "Content-Type: application/json" \
  -d '{
    "address":"abb4a45ca70104ecc943a49e4553f0e7e1135b01",
    "committees":"1",
    "orderId":"1516f1cdd23f7e9f89e13a96ddf86351775f130f",
    "memo":"",
    "fee":10000,
    "submit":false,
    "password":"test"
    }'

> {
  "type": "deleteOrder",
  "msg": {
    "orderID": "1516f1cdd23f7e9f89e13a96ddf86351775f130f",
    "chainID": 1
  },
  "signature": {
    "publicKey": "83e91c8cf692365efd9a99a5efbd0afcc3d93a1e88e9bfe7d5219f9f5cf50cb785dd8c9727a1618a92100e28d47f7bf1",
    "signature": "8043f0c6827aea8449a238811af800dec35cc78222cd3f3b0c830323c22902920f7b19d963d6ff76165bf3754f6dfb6504ddd20368b32169720880fd928e7ccbbfd628cd4c2c065c280fa51bc2d4b0b23d88fe9ffa6088b5d19e1b3e80463783"
  },
  "time": 1749644810582870,
  "createdHeight": 196596,
  "fee": 10000,
  "networkID": 1,
  "chainID": 1
}
```

## Txn Dex Limit Order

**Route:** `/v1/admin/tx-dex-limit-order`

**Description**: generates/submits a DEX limit order transaction

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the from address
- **amount**: `uint64` - the amount to sell in smallest denomination
- **receiveAmount**: `uint64` - the amount to receive in smallest denomination
- **committees**: `string` - the committee id of the counter asset
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **memo**: `string` - an arbitrary message encoded in the transaction
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction

**Response**: (See tx-by-hash and MessageDexLimitOrder)

```
$ curl -X POST http://localhost:50003/v1/admin/tx-dex-limit-order \
  -H "Content-Type: application/json" \
  -d '{
    "address":"b0b4a45ca70104ecc943a49e4553f0e7e1135b01",
    "amount": 1000000,
    "receiveAmount": 2000000,
    "committees":"1",
    "fee":0,
    "submit":false,
    "password":"test"
    }'
  
> {
  "type": "dexLimitOrder",
  "msg": {
    "chainID": 1,
    "amountForSale": 1000000,
    "requestedAmount": 2000000,
    "sellerReceiveAddress": "b0b4a45ca70104ecc943a49e4553f0e7e1135b01"
  },
  "signature": {
    "publicKey": "83e91c8cf692365efd9a99a5efbd0afcc3d93a1e88e9bfe7d5219f9f5cf50cb785dd8c9727a1618a92100e28d47f7bf1",
    "signature": "835dfa9e5a370233369d1e955620a1512f9a5c31702e718e52d6cc60f40a91a0f142ae8e35c1dadd3213b17f881fbe6413c64103e27a7336296d963daf9a6e91cc9625d470248db009a9cde63c55cb6f4282f96b936bde547756e85e9ed84bb5"
  },
  "time": 1749644810582870,
  "createdHeight": 196596,
  "fee": 10000,
  "networkID": 1,
  "chainID": 1
}
```

## Txn Dex Liquidity Deposit

**Route:** `/v1/admin/tx-dex-liquidity-deposit`

**Description**: generates/submits a DEX liquidity deposit transaction

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the from address
- **amount**: `uint64` - the amount to deposit in smallest denomination
- **committees**: `string` -  the committee id of the counter asset
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **memo**: `string` - an arbitrary message encoded in the transaction
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction

**Response**: (See tx-by-hash and MessageDexLiquidityDeposit)

```
$ curl -X POST http://localhost:50003/v1/admin/tx-dex-liquidity-deposit \
  -H "Content-Type: application/json" \
  -d '{
    "address":"b0b4a45ca70104ecc943a49e4553f0e7e1135b01",
    "amount": 1000000,
    "committees":"1",
    "fee":0,
    "submit":false,
    "password":"test"
    }'
  
> {
  "type": "dexLiquidityDeposit",
  "msg": {
    "chainID": 1,
    "amount": 1000000,
    "address": "b0b4a45ca70104ecc943a49e4553f0e7e1135b01"
  },
  "signature": {
    "publicKey": "83e91c8cf692365efd9a99a5efbd0afcc3d93a1e88e9bfe7d5219f9f5cf50cb785dd8c9727a1618a92100e28d47f7bf1",
    "signature": "835dfa9e5a370233369d1e955620a1512f9a5c31702e718e52d6cc60f40a91a0f142ae8e35c1dadd3213b17f881fbe6413c64103e27a7336296d963daf9a6e91cc9625d470248db009a9cde63c55cb6f4282f96b936bde547756e85e9ed84bb5"
  },
  "time": 1749644810582870,
  "createdHeight": 196596,
  "fee": 10000,
  "networkID": 1,
  "chainID": 1
}
```

## Txn Dex Liquidity Withdraw

**Route:** `/v1/admin/tx-dex-liquidity-withdraw`

**Description**: generates/submits a DEX liquidity withdraw transaction

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the from address
- **percent**: `int` - the percentage of liquidity to withdraw (1-100)
- **committees**: `string` - the committee id of the counter asset
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **memo**: `string` - an arbitrary message encoded in the transaction
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction

**Response**: (See tx-by-hash and MessageDexLiquidityWithdraw)

```
$ curl -X POST http://localhost:50003/v1/admin/tx-dex-liquidity-withdraw \
  -H "Content-Type: application/json" \
  -d '{
    "address":"b0b4a45ca70104ecc943a49e4553f0e7e1135b01",
    "percent": 50,
    "committees":"1",
    "fee":0,
    "submit":false,
    "password":"test"
    }'
  
> {
  "type": "dexLiquidityWithdraw",
  "msg": {
    "chainID": 1,
    "percent": 50,
    "address": "b0b4a45ca70104ecc943a49e4553f0e7e1135b01"
  },
  "signature": {
    "publicKey": "83e91c8cf692365efd9a99a5efbd0afcc3d93a1e88e9bfe7d5219f9f5cf50cb785dd8c9727a1618a92100e28d47f7bf1",
    "signature": "835dfa9e5a370233369d1e955620a1512f9a5c31702e718e52d6cc60f40a91a0f142ae8e35c1dadd3213b17f881fbe6413c64103e27a7336296d963daf9a6e91cc9625d470248db009a9cde63c55cb6f4282f96b936bde547756e85e9ed84bb5"
  },
  "time": 1749644810582870,
  "createdHeight": 196596,
  "fee": 10000,
  "networkID": 1,
  "chainID": 1
}
```

## Txn Lock Order (Nested-Chain Only)

**Route:** `/v1/admin/tx-lock-order`

**Description**: generates/submits a lock (sell) order transaction.

Notes:
1. Can only go through if order is not yet 'locked' by a buyer.
2. This transaction is executed on the nested-chain but is reported back to the root-chain by the committee
3. This is only for nested chains trying to lock an order based on the root-chain
4. Embeds a 'lock order' command in a standard self-send transaction - this is a good model of how this could work in most chains like Ethereum or Bitcoin but can be `Nested-Chain` specific.
5. The default for Canopy chains carries a fee of 2x the Send Fee for extra spam protection on locks

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the address that is sending the nested-chain 'counter-asset'
- **receiveAddress**: `hex-string` - the address on the root-chain that is receiving the `sell order` funds
- **orderId**: `hex-string` - the unique id of the sell-order on the root-chain
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction

**Response**: (See tx-by-hash and MessageSend)

```
$ curl -X POST http://localhost:50003/v1/admin/tx-lock-order \
  -H "Content-Type: application/json" \
  -d '{
    "address":"271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5","receiveAddress":"abb4a45ca70104ecc943a49e4553f0e7e1135b01","orderId":"1516f1cdd23f7e9f89e13a96ddf86351775f130f",
    "fee":0,
    "submit":false,
    "password":"test"
    }'
  
> {
  "type": "send",
  "msg": {
    "fromAddress": "271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5",
    "toAddress": "271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5",
    "amount": 1
  },
  "signature": {
    "publicKey": "83e91c8cf692365efd9a99a5efbd0afcc3d93a1e88e9bfe7d5219f9f5cf50cb785dd8c9727a1618a92100e28d47f7bf1",
    "signature": "835dfa9e5a370233369d1e955620a1512f9a5c31702e718e52d6cc60f40a91a0f142ae8e35c1dadd3213b17f881fbe6413c64103e27a7336296d963daf9a6e91cc9625d470248db009a9cde63c55cb6f4282f96b936bde547756e85e9ed84bb5"
  },
  "time": 1749645076176475,
  "createdHeight": 196596,
  "fee": 20000,
  "memo": "{\"orderId\":\"1516f1cdd23f7e9f89e13a96ddf86351775f130f\",\"chain_id\":2,\"buyerSendAddress\":\"271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5\",\"buyerReceiveAddress\":\"abb4a45ca70104ecc943a49e4553f0e7e1135b01\"}",
  "networkID": 1,
  "chainID": 2
}
```

## Txn Close Order (Nested-Chain Only)

**Route:** `/v1/admin/tx-close-order`

**Description**: generates/submits a close (sell) order transaction.

Notes:
1. Can only go through if order is already 'locked' by this sender as the buyer.
2. This transaction is executed on the nested-chain but is reported back to the root-chain by the committee
3. This is only for nested chains trying to lock an order based on the root-chain
4. Embeds a 'lock order' command in a standard self-send transaction - this is a good model of how this could work in most chains like Ethereum or Bitcoin but can be `Nested-Chain` specific.
5. The default for Canopy chains carries a fee of 2x the Send Fee for extra spam protection on closes

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the address that is sending the nested-chain 'counter-asset'
- **orderId**: `hex-string` - the unique id of the sell-order on the root-chain
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction

**Response**: (See tx-by-hash and MessageSend)

```
$ curl -X POST http://localhost:50003/v1/admin/tx-close-order \
  -H "Content-Type: application/json" \
  -d '{
    "address":"271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5","orderId":"1516f1cdd23f7e9f89e13a96ddf86351775f130f",
    "fee":0,
    "submit":false,
    "password":"test"
  }'
  
> {
  "type": "send",
  "msg": {
    "fromAddress": "271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5",
    "toAddress": "271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5",
    "amount": 1
  },
  "signature": {
    "publicKey": "83e91c8cf692365efd9a99a5efbd0afcc3d93a1e88e9bfe7d5219f9f5cf50cb785dd8c9727a1618a92100e28d47f7bf1",
    "signature": "835dfa9e5a370233369d1e955620a1512f9a5c31702e718e52d6cc60f40a91a0f142ae8e35c1dadd3213b17f881fbe6413c64103e27a7336296d963daf9a6e91cc9625d470248db009a9cde63c55cb6f4282f96b936bde547756e85e9ed84bb5"
  },
  "time": 1749645076176475,
  "createdHeight": 196596,
  "fee": 20000,
  "memo": "{\"orderId\":\"1516f1cdd23f7e9f89e13a96ddf86351775f130f\",\"chain_id\":2,\"closeOrder\":\true"}",
  "networkID": 1,
  "chainID": 2
}
```

## Txn Subsidy

**Route:** `/v1/admin/tx-subsidy`

**Description**: generates/submits a subsidy transaction.

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the address that is sending the funds
- **id**: `uint64` - the id of the pool being subsidized
- **amount**: `hex-string` - the amount being sent in micro denominiation
- **opcode**: `hex-string` - an arbitrary instruction code that may be embedded in the message
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction

**Response**: (See tx-by-hash and MessageSubsidy)

```
$ curl -X POST http://localhost:50003/v1/admin/tx-subsidy \
  -H "Content-Type: application/json" \
  -d '{
    "address":"271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5",
    "amount": 1000,
    "id":"1",
    "fee":0,
    "submit":false,
    "password":"test"
  }'
  
> {
  "type": "subsidy",
  "msg": {
    "address": "271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5",
    "chainID": 0,
    "amount": 1000,
    "opcode": ""
  },
  "signature": {
    "publicKey": "83e91c8cf692365efd9a99a5efbd0afcc3d93a1e88e9bfe7d5219f9f5cf50cb785dd8c9727a1618a92100e28d47f7bf1",
    "signature": "b4d2793972505d56381c407a99c4d2bd05848ec433c4ce53f558ea8a2ad4014c657b46c056e5519a43fe4c322aaa227011f2199c314b0c8a4e29f850e092449ebbf5d9ee94984d96266b8b2246b2eaa1b720a046eff16e97636196bf707074ce"
  },
  "time": 1749647821374907,
  "createdHeight": 65,
  "fee": 10000,
  "networkID": 1,
  "chainID": 1
}
```

## Txn Start Poll

**Route:** `/v1/admin/tx-start-poll`

**Description**: generates/submits a start on-chain poll transaction.

Note: Embeds a 'start-poll' command in a standard self-send transaction - this is a good model of how this could work in most chains like Ethereum or Bitcoin but can be `Nested-Chain` specific.

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the address that is initiating the poll
- **pollJSON**: `json string` - the poll json object (proposal, endBlock, url)
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction

**Response**: (See tx-by-hash and MessageSend) with poll embedded in the memo

```
$ curl -X POST http://localhost:50003/v1/admin/tx-start-poll \
  -H "Content-Type: application/json" \
  -d '{
    "address":"271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5",
    "pollJSON":{"proposal":"canopy network is the best","endBlock":100,"URL":"https://discord.com/link-to-thread"},
    "password":"test",
    "submit":false
    }'
  
> {
  "type": "send",
  "msg": {
    "fromAddress": "271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5",
    "toAddress": "271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5",
    "amount": 1
  },
  "signature": {
    "publicKey": "83e91c8cf692365efd9a99a5efbd0afcc3d93a1e88e9bfe7d5219f9f5cf50cb785dd8c9727a1618a92100e28d47f7bf1",
    "signature": "8e154daa1909ec4b3a60df1f0b7f222c226be0b9057d243eb7ba6979447832b78535980324b564cd880086109db4e09c0c24b04a6aa2ebd883f78a4e05412334fe5f45652a0405e613c7d71a4118e2e4e865c19f47f06177fc1773c9410e0dac"
  },
  "time": 1749646707973281,
  "createdHeight": 10,
  "fee": 10000,
  "memo": "{\"startPoll\":\"04729495295b8fb37cd82562d0876afc838637daf2024670bc49a8b241926661\",\"url\":\"https://discord.com/link-to-thread\",\"endHeight\":100}",
  "networkID": 1,
  "chainID": 1
}
```


## Txn Vote Poll

**Route:** `/v1/admin/tx-vote-poll`

**Description**: generates/submits a vote on-chain poll transaction.

Note: Embeds a 'vote-poll' command in a standard self-send transaction - this is a good model of how this could work in most chains like Ethereum or Bitcoin but can be `Nested-Chain` specific.

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the address that is initiating the poll
- **pollJSON**: `json string` - the poll json object (proposal, endBlock, url)
- **pollApprove**: `bool` - vote yes or not on the poll
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction

**Response**: (See tx-by-hash and MessageSend) with poll vote embedded in the memo

```
$ curl -X POST http://localhost:50003/v1/admin/tx-vote-poll \
  -H "Content-Type: application/json" \
  -d '{
    "address":"271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5",
    "pollJSON":{"proposal":"canopy network is the best","endBlock":100,"URL":"https://discord.com/link-to-thread"},
    "pollApprove":true,
    "password":"test",
    "submit":false
    }'
  
> {
  "type": "send",
  "msg": {
    "fromAddress": "271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5",
    "toAddress": "271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5",
    "amount": 1
  },
  "signature": {
    "publicKey": "83e91c8cf692365efd9a99a5efbd0afcc3d93a1e88e9bfe7d5219f9f5cf50cb785dd8c9727a1618a92100e28d47f7bf1",
    "signature": "8b21bce5c260e556107647cc085174bc3353f022116005972655a7c3c2484a5b5b379bf949e4ef464fac3c8659090c5e188db778d29790612be91e537567725ebfad9e7257757e4a116f46bc21da404eed8910e965edc79679f87d6e93292d5d"
  },
  "time": 1749646925994608,
  "createdHeight": 21,
  "fee": 10000,
  "memo": "{\"votePoll\":\"04729495295b8fb37cd82562d0876afc838637daf2024670bc49a8b241926661\",\"approve\":true}",
  "networkID": 1,
  "chainID": 1
```


## Txn DAO Transfer

**Route:** `/v1/admin/tx-dao-transfer`

**Description**: generates/submits a dao transfer transaction. Note: this transaction type is a proposal and requires approval from +2/3rds of the validator stake

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the address where the funds will be received
- **amount**: `uint64` - the amount requested from the DAO
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **startHeight**: `uint64` - is the beginning height where the transaction must. be sent, this field locks in a block-range when it's converted to JSON, allowing Validators a deadline to vote, and creating a valid window when this transaction may be submitted
- **endHeight**: `uint64` -  is the ending height counterpart to `startHeight`
- **memo**: `string` - an arbitrary message encoded in the transaction
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction
- **signer**: `hex-string` - the address associated with the key that is signing the transaction

**Response**: (See tx-by-hash and MessageDAOTransfer)

```
$ curl -X POST http://localhost:50003/v1/admin/tx-dao-transfer \
  -H "Content-Type: application/json" \
  -d '{
    "address":"abb4a45ca70104ecc943a49e4553f0e7e1135b01",
    "amount":100000000,
    "startBlock":1,
    "endBlock":100,
    "memo":"hello world",
    "fee":0,
    "submit":false,
    "password":"test"
    }'

> {
  "type": "daoTransfer",
  "msg": {
  "address": "abb4a45ca70104ecc943a49e4553f0e7e1135b01",
  "amount": 1000000000,
  "startHeight": 1,
  "endHeight": 100
  },
  "signature": {
  "publicKey": "8cb57bdcc0f90c36aea48a6edab31996400af55f623f676a3c64c04b826454aea227f14d1b70c150d5fa9fe5f6790f60",
  "signature": "a0473e839b23abc7d8448de5c58141827d86157263deaa79db0d9a1d1f0ffa9f7503c9ff52be7645b3ca0de50482651c1857d2af39c6953a68e386ca0901f668ef61dd706e4ed753548b5cb6737aceb06307e5894fbe49668fa2a1ff9036e971"
  },
  "time": 1749642868861970,
  "createdHeight": 196596,
  "fee": 10000,
  "memo": "hello world",
  "networkID": 1,
  "chainID": 1
}
```

## Txn Change Parameter

**Route:** `/v1/admin/tx-change-param`

**Description**: generates/submits a change parameter transaction. Note: this transaction type is a proposal and requires approval from +2/3rds of the validator stake

**HTTP Method**: `POST`

**Request**:
- **address**: `hex-string` - the address where the funds will be received
- **paramSpace**: `string` - the sub-space where the parameter exists (fee, val, cons, gov)
- **paramKey**: `string` - the unique name of the parameter
- **paramValue**: `uint64/string` - the updated value of the parameter
- **startHeight**: `uint64` - is the beginning height where the transaction must. be sent, this field locks in a block-range when it's converted to JSON, allowing Validators a deadline to vote, and creating a valid window when this transaction may be submitted
- **endHeight**: `uint64` -  is the ending height counterpart to `startHeight`
- **fee**: `uint64` - the transaction fee in micro denomination (optional - minimum fee filled if 0)
- **memo**: `string` - an arbitrary message encoded in the transaction
- **submit**: `bool` - submit this transaction or not (returns the tx-hash if true)
- **password**: `string` - the password associated to decrypt the private key to sign the transaction
- **signer**: `hex-string` - the address associated with the key that is signing the transaction

**Response**: (See tx-by-hash and MessageChangeParameter)

```
$ curl -X POST http://localhost:50003/v1/admin/tx-change-param \
  -H "Content-Type: application/json" \
  -d '{
    "address":"abb4a45ca70104ecc943a49e4553f0e7e1135b01",
    "paramSpace":"fee",
    "paramKey":"sendFee",
    "paramValue":"20000",
    "startBlock":1,
    "endBlock":100,
    "memo":"hello world",
    "fee":10000,
    "submit":false,
    "password":"test"
    }'

> {
  "type": "changeParameter",
  "msg": {
    "parameterSpace": "fee",
    "parameterKey": "sendFee",
    "parameterValue": 20000,
    "startHeight": 1,
    "endHeight": 100,
    "signer": "abb4a45ca70104ecc943a49e4553f0e7e1135b01"
  },
  "signature": {
    "publicKey": "8cb57bdcc0f90c36aea48a6edab31996400af55f623f676a3c64c04b826454aea227f14d1b70c150d5fa9fe5f6790f60",
    "signature": "882b6b3ede14a6ca6cc1a228c21f73ece1db2e4b28b51d10b3d2c3bac8bade80342a358cff65f496127803469438f9dd0d1c49c7231f0a4861ab45c6a9941e0724eba59afc8210dfd64acf05770375957a85a98ccecd798b349dc26125d6472b"
  },
  "time": 1749643492341200,
  "createdHeight": 196596,
  "fee": 10000,
  "memo": "hello world",
  "networkID": 1,
  "chainID": 1
}
```

## Add Proposal Vote

**Route:** `/v1/gov/add-vote`

**Description**: configures local validator to vote yes or no on a proposal transaction when included in a block

**HTTP Method**: `POST`

**Request**:
- **approve**: `bool` - approve the proposal or not
- **proposal**: `object` - the proposal transaction

**Response**: Echo back the request

```
$ curl -X POST http://localhost:50003/v1/gov/add-vote \
  -H "Content-Type: application/json" \
  -d '{
  "approve": true,
  "proposal": {
    "type": "changeParameter",
    "msg": {
      "parameterSpace": "fee",
      "parameterKey": "sendFee",
      "parameterValue": 1000,
      "startHeight": 1,
      "endHeight": 100,
      "signer": "271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5"
    },
    "signature": {
      "publicKey": "83e91c8cf692365efd9a99a5efbd0afcc3d93a1e88e9bfe7d5219f9f5cf50cb785dd8c9727a1618a92100e28d47f7bf1",
      "signature": "b3ed9ba4988a1209af9871ebf086ff7d8ae81efc968144e65d9f37abecf51070c53e6d2bbbb9c91e79ea8de9ee4a1cda1611b908ecf6dd63625ef5fa6d048cb618d871cdf83cc93000b5126c3aa7fe658af8e3f71eb1e2860aa32fed6892dd7b"
    },
    "time": 1749647127495605,
    "createdHeight": 31,
    "fee": 10000,
    "networkID": 1,
    "chainID": 1
  }
}'
  
> {
  "approve": true,
  "proposal": {
    "type": "changeParameter",
    "msg": {
      "parameterSpace": "fee",
      "parameterKey": "sendFee",
      "parameterValue": 1000,
      "startHeight": 1,
      "endHeight": 100,
      "signer": "271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5"
    },
    "signature": {
      "publicKey": "83e91c8cf692365efd9a99a5efbd0afcc3d93a1e88e9bfe7d5219f9f5cf50cb785dd8c9727a1618a92100e28d47f7bf1",
      "signature": "b3ed9ba4988a1209af9871ebf086ff7d8ae81efc968144e65d9f37abecf51070c53e6d2bbbb9c91e79ea8de9ee4a1cda1611b908ecf6dd63625ef5fa6d048cb618d871cdf83cc93000b5126c3aa7fe658af8e3f71eb1e2860aa32fed6892dd7b"
    },
    "time": 1749647127495605,
    "createdHeight": 31,
    "fee": 10000,
    "networkID": 1,
    "chainID": 1
  }
}
```

## Remove Proposal Vote

**Route:** `/v1/gov/del-vote`

**Description**: removes a proposal vote from a local validator

**HTTP Method**: `POST`

**Request**:
- **proposal**: `object` - the proposal transaction

**Response**: Echo back the request

```
$ curl -X POST http://localhost:50003/v1/gov/del-vote \
  -H "Content-Type: application/json" \
  -d '{
  "proposal": {
    "type": "changeParameter",
    "msg": {
      "parameterSpace": "fee",
      "parameterKey": "sendFee",
      "parameterValue": 1000,
      "startHeight": 1,
      "endHeight": 100,
      "signer": "271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5"
    },
    "signature": {
      "publicKey": "83e91c8cf692365efd9a99a5efbd0afcc3d93a1e88e9bfe7d5219f9f5cf50cb785dd8c9727a1618a92100e28d47f7bf1",
      "signature": "b3ed9ba4988a1209af9871ebf086ff7d8ae81efc968144e65d9f37abecf51070c53e6d2bbbb9c91e79ea8de9ee4a1cda1611b908ecf6dd63625ef5fa6d048cb618d871cdf83cc93000b5126c3aa7fe658af8e3f71eb1e2860aa32fed6892dd7b"
    },
    "time": 1749647127495605,
    "createdHeight": 31,
    "fee": 10000,
    "networkID": 1,
    "chainID": 1
  }
}'
  
> {
  "approve": false,
  "proposal": {
    "type": "changeParameter",
    "msg": {
      "parameterSpace": "fee",
      "parameterKey": "sendFee",
      "parameterValue": 1000,
      "startHeight": 1,
      "endHeight": 100,
      "signer": "271e0120ac7f11a6f60ba124b2b187eaf1e2e6f5"
    },
    "signature": {
      "publicKey": "83e91c8cf692365efd9a99a5efbd0afcc3d93a1e88e9bfe7d5219f9f5cf50cb785dd8c9727a1618a92100e28d47f7bf1",
      "signature": "b3ed9ba4988a1209af9871ebf086ff7d8ae81efc968144e65d9f37abecf51070c53e6d2bbbb9c91e79ea8de9ee4a1cda1611b908ecf6dd63625ef5fa6d048cb618d871cdf83cc93000b5126c3aa7fe658af8e3f71eb1e2860aa32fed6892dd7b"
    },
    "time": 1749647127495605,
    "createdHeight": 31,
    "fee": 10000,
    "networkID": 1,
    "chainID": 1
  }
}
```

## Peer Info

**Route:** `/v1/admin/peer-info`

**Description**: gets the resource usage for the node

**HTTP Method**: `GET`

**Request**: `none`

**Response**:
- **id**: `object` - the id of the this node as seen by peers
  - **publicKey**: `hex-string` - the peer-to-peer public key
  - **netAddress**: `hex-string` - the external net address as configured
  - **peerMeta**: `object` - the meta data about the peer
    - **networkID**: `uint64` - the identifier of the network (1=mainnet, 2=testnet)
    - **chainID**: `uint64` - the identifier of the chain (1=Canopy, 2=Canary)
- **numPeers**: `uint64` - the total number of peers
- **numInbound**: `uint64` - the total number of peers that dialed this node and connected
- **numOutbound**: `uint64` - the total number of peers that this node dialed and connected to
- **peers**: `array` - the list of peers
  - **address**: `object` - the identifier of the peer
    - **publicKey**: `hex-string` - the peer-to-peer public key
    - **netAddress**: `hex-string` - the external net address as configured
    - **peerMeta**: `object` - the meta data about the peer
    - **networkID**: `uint64` - the identifier of the network (1=mainnet, 2=testnet)
    - **chainID**: `uint64` - the identifier of the chain (1=Canopy, 2=Canary)
  - **isOutbound**: `bool` - was this peer dialed or did this peer dial our node?
  - **isValidator**: `bool` - is this peer an active validator for our chainId?
  - **isMustConnect**: `bool` - must this node connect to this peer?
  - **isTrusted**: `bool` - is this peer trusted by configuration?
  - **reputation**: `int` - what is the reputation of this peer according to our node?
```
$ curl http://localhost:50003/v1/admin/peer-info
  
> {
  "id": {
    "publicKey": "b88a5928e54cbf0a36e0b98f5bcf02de9a9a1deba6994739f9160181a609f516eb702936a0cbf4c1f2e7e6be5b8272f2",
    "netAddress": "127.125.17.207",
    "peerMeta": {
      "networkID": 2,
      "chainID": 1
    }
  },
  "numPeers": 1,
  "numInbound": 0,
  "numOutbound": 1,
  "peers": [
    {
      "address": {
        "publicKey": "98d45087a99bcbfde91993502e77dde869d4485c3778fe46513958320da560823d56a0108f4cf3513393f4d561bc489b",
        "netAddress": "159.89.181.58:9001",
        "peerMeta": {
          "networkID": 2,
          "chainID": 1,
        }
      },
      "isOutbound": true,
      "isValidator": true,
      "isMustConnect": true,
      "isTrusted": false,
      "reputation": 10
    }
  ]
}
```


## Peer Book

**Route:** `/v1/admin/peer-book`

**Description**: gets the peer address book for this local node

**HTTP Method**: `GET`

**Request**: `none`

**Response**: `array` - the list of peers in the book
- **address**: `object` - the identifier of the peer
  - **publicKey**: `hex-string` - the peer-to-peer public key
  - **netAddress**: `hex-string` - the external net address as configured
  - **peerMeta**: `object` - the meta data about the peer
  - **networkID**: `uint64` - the identifier of the network (1=mainnet, 2=testnet)
  - **chainID**: `uint64` - the identifier of the chain (1=Canopy, 2=Canary)
- **consecutiveFailedDial**: `uint64` - how many consecutive failed dial attempts are there in churn management process?

```
$ curl http://localhost:50003/v1/admin/peer-book
  
> [
  {
    "address": {
      "publicKey": "abd45087acdbcbfde91993502e77dde869d4485c3778fe46513958320da56082fd56a0108f4cf3513393f4d561bf48eb",
      "netAddress": "132.23.81.38:9001",
      "peerMeta": {
        "networkID": 2,
        "chainID": 1,
      }
    },
    "consecutiveFailedDial": 0
  },
]
```

## Consensus Info

**Route:** `/v1/admin/consensus-info`

**Description**: gets the consensus dump from this node (if a validator)

**HTTP Method**: `GET`

**Request**: `none`

**Response**:

- **isSyncing**: `bool` - is the node syncing or not?
- **view**: `object` - the current view of BFT
- **blockHash**: `hex-string` - the hash of the proposal block (if any)
- **resultsHash**: `hex-string` - the hash of the proposal results (if any)
- **locked**: `bool` - is the node 'locked' on a proposal
- **address**: `hex-string` - the address of this validator
- **publicKey**: `hex-string` - the pubKey of this validator
- **proposerAddress**: `hex-string` - the address of the current proposer
- **proposer**: `hex-string` - the pubKey of the proposer
- **proposals**: `object` - `map[round]` -> 'proposals' received from the Leader Validator(s)
- **partialQcs**: `object` - double sign evidence
- **pacemakerVotes**: `object` -  view messages from the current ValidatorSet allowing the node to synchronize to the highest +2/3 seen Round
- **minimumPowerFor23Maj**: `uint64` - minimum amount of voting power needed to acheive a +2/3rds majority
- **votes**: `object` - `map[round]` -> 'votes' received from Replica (non-leader) Validators
- **status**: `string` - useful message about the current BFT status of the node

```
$ curl http://localhost:50003/v1/admin/consensus-info
  
> {
  "isSyncing": false,
  "view": {
    "height": 15672,
    "committeeHeight": 15672,
    "round": 0,
    "phase": "PROPOSE_VOTE",
    "networkID": 2,
    "chainId": 1
  },
  "blockHash": "",
  "resultsHash": "",
  "locked": false,
  "address": "851e90eaef1fa27debaee2c2591503bdeec1d123",
  "publicKey": "b88a5928e54cbf0a36e0b98f5bcf02de9a9a1deba6994739f9160181a609f516eb702936a0cbf4c1f2e7e6be5b8272f2",
  "proposerAddress": "",
  "proposer": "",
  "proposals": {
    "0": {
      "1_ELECTION": null,
      "3_PROPOSE": [
        {
          "header": {
            "height": 15672,
            "committeeHeight": 15672,
            "round": 0,
            "phase": "PROPOSE",
            "networkID": 2,
            "chainId": 1
          },
          "qc": {
            "header": {
              "height": 15672,
              "committeeHeight": 15672,
              "round": 0,
              "phase": "ELECTION_VOTE",
              "networkID": 2,
              "chainId": 1
            },
            "block": "...",
            "blockHash": "513fb23628627a424fdc4b85b41283df2e0110637bc780cd78423d1be170f096",
            "resultsHash": "0a8d6752d87bc3a2de60daaba14e2a235dd1e2ef53fe864bbfe5a20881ce196d",
            "results": {
              "rewardRecipients": {
                "paymentPercents": [
                  {
                    "address": "02cd4e5eb53ea665702042a6ed6d31d616054dc5",
                    "percents": 100,
                    "chainId": 1
                  }
                ]
              },
              "slashRecipients": {},
              "orders": {
                "lockOrders": null,
                "resetOrders": null,
                "closeOrders": null
              }
            },
            "proposerKey": "98d45087a99bcbfde91993502e77dde869d4485c3778fe46513958320da560823d56a0108f4cf3513393f4d561bc489b",
            "signature": {
              "signature": "8332e70fd02c193d9dd7248d27a79608ad1159f694656850dac4f0668fa65f42ebfdac8714a4f680ea448ebf858d0519046cbcec3922c6a1305e479856cbcaafdf4e5b1fd6087a86f792d98a32ce9039cfdff6fab07151f65d9db27c20d1f7f5",
              "bitmap": "07"
            }
          },
          "highQC": null,
          "lastDoubleSignEvidence": null,
          "signature": {
            "publicKey": "98d45087a99bcbfde91993502e77dde869d4485c3778fe46513958320da560823d56a0108f4cf3513393f4d561bc489b",
            "signature": "af7d8c09c361344e7ce9459aaf597e9c5628afadf8f1cfa969631d9c9f0f5509ef5b510fe8dd3eee9ab23cd9e986cb9f186c00d92c551cd3215bf54fac5519c14bd1638f3059992a1c07883fbbbed1f82c7d2ebc87b68e70d306c6273afc0670"
          }
        }
      ]
    }
  },
  "partialQCs": {},
  "pacemakerVotes": {},
  "minimumPowerFor23Maj": 2000000001,
  "votes": {},
  "status": "voting on proposal"
}
```

## Resource Usage

**Route:** `/v1/admin/resource-usage`

**Description**: gets the resource usage for the node

**HTTP Method**: `GET`

**Request**: `none`

**Response**: See below

```
$ curl http://localhost:50003/v1/admin/resource-usage
  
> {
  "process": {
    "name": "___go_build_github_com_canopy_network_canopy_cmd_cli",
    "status": "S",
    "createTime": "18 Feb 14 05:40 AST",
    "fdCount": 46,
    "threadCount": 22,
    "usedMemoryPercent": 4.719185829162598,
    "usedCPUPercent": 55.01593610361968
  },
  "system": {
    "totalRAM": 17179869184,
    "availableRAM": 5899804672,
    "usedRAM": 11280064512,
    "usedRAMPercent": 65.65861701965332,
    "freeRAM": 205660160,
    "usedCPUPercent": 7.31242895399022,
    "userCPU": 369023.99,
    "systemCPU": 140800.73,
    "idleCPU": 6470812.52,
    "totalDisk": 1000240963584,
    "usedDisk": 641660645376,
    "usedDiskPercent": 64.1506065775233,
    "freeDisk": 358580318208,
    "ReceivedBytesIO": 38328565621,
    "WrittenBytesIO": 9078489286
  }
}
```

## Logs Stream

**Route:** `/v1/admin/logs`

**Description**: gets the stream of node logs (up to last log rotation)

**HTTP Method**: `GET`

**Request**: `none`

**Response**: See below

```
$ curl http://localhost:50003/v1/admin/log
  
> 
Jun 11 09:47:15.533 INFO: Self is the proposer
Jun 11 09:47:15.532 INFO: (rH:155, H:155, R:0, P:PROPOSE)
Jun 11 09:47:13.538 INFO: Voting SELF as the proposer
Jun 11 09:47:13.532 INFO: (rH:155, H:155, R:0, P:ELECTION_VOTE)
Jun 11 09:47:11.537 INFO: Self is a leader candidate 🗳️
Jun 11 09:47:11.531 INFO: (rH:155, H:155, R:0, P:ELECTION)
Jun 11 09:47:09.525 INFO: Reset BFT (NEW_COMMITTEE)
Jun 11 09:47:09.521 INFO: Reset BFT (NEW_HEIGHT)
...
```

## Golang Profiling Debug

**Route:**
- DebugBlockedRoutePath = "/debug/blocked"
- DebugHeapRoutePath    = "/debug/heap"
- DebugCPURoutePath     = "/debug/cpu"
- DebugRoutineRoutePath = "/debug/routine"

**Description**: returns an HTTP handler that serves the named profile. Available profiles can be found in [runtime/pprof.Profile]. See https://pkg.go.dev/net/http/pprof

**HTTP Method**: `GET`
