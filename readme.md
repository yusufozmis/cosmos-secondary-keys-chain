# example: Cosmos SDK Blockchain
**example** is a blockchain built using Cosmos SDK and Tendermint and created with [Ignite CLI](https://ignite.com/cli).

## Get started

```
ignite chain serve --reset-once 
```
OR if you wanna see the logs

```
ignite chain serve --reset-once -v 
```

`serve` command installs dependencies, builds, initializes, and starts your blockchain in development.

```--reset-once``` flag builds the chain using config.yml file. For some reason, without this flag chain does not enable vote-extensions.

## Custom module secondaryKeys

This module defines 2 maps to store the secondary public keys. One map is used in vote extensions, map[validator address] = validator's secondary public key

And the second map is used in Ante Handler, and stores user's secondary public keys. This allows for a secondary signature scheme to be implemented.

This module also defines a new transaction type: ```BroadcastData```. Users submit this transaction to register their secondary public key into state.

## Benchmarking

Benchmark tests currently generates 10 random accounts, creates 10k transactions, and broadcasts asyncrounosly. 

Previous benchmarking results are

- 416 TPS
- 585 TPS
- 630 TPS
- 626 TPS
- 522 TPS
- 552 TPS

Based on the previous benchmarking tests, chain's TPS is estimated to be 550 TPS

