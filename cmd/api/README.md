# Quick Block Explorer

An explorer to the depths of IOV networks.

## Requirements

- go 1.13+
- direnv (_optionally_)
- docker

You must also `export GO111MODULE=on` in your environment to use the go modules feature.

## Run local

For local development you can use a postgres instance and any Tendermint
node address.

Before the steps below, make sure you created a docker persistent volume: `mkdir -p $HOME/docker/volumes/postgres`

### DB

1. Run postgres `docker run --rm --name pg-docker -e POSTGRES_PASSWORD='postgres' -d -p 5432:5432 -v $HOME/docker/volumes/postgres:/var/lib/postgresql/data  postgres`
2. Export env variables

### Environment 

```sh
TENDERMINT_WS_URI="wss://rpc-private-a-vip-mainnet.iov.one/websocket"
DATABASE_URL="postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable"
HRP="iov"
ALLOWED_ORIGIN="*"
PORT="8000"
```

### Deployment

Heroku deployments are done from `heroku-deployment` branch. Auto deployment is enable so just merge your PR to the
 repo.


### Collector

Run `make run-collector`

### API

Run `make run-api`

#### Endpoints

- `/api/blocks/latest`: returns the latest `Block`

```json
{
    "fee_frac": 500000000,
    "hash": "1ea2ae0a9612510625904674bc820be25b515d91b7e45f17e2603d2df51627e7",
    "height": 72,
    "messages": [
        "cash/send"
    ],
    "proposer_name": "StakeWith.Us",
    "time": "2019-10-10T11:35:35.499602Z",
    "transactions": [
        {
            "block_height": 72,
            "hash": "6a751b1f730faabfeebc6b70f0b0ec7daf16d7c78e88189ecad4bbc7ef991dac",
            "message": {
                "details": {
                    "amount": {
                        "ticker": "IOV",
                        "whole": 10
                    },
                    "destination": "iov1cwjx9wrd3tndzj45x90hfc4u7vjph3etycrmqx",
                    "memo": "bitlion",
                    "source": "iov1vr7zdpxh4mg0tzy7lf4p0r2rsmln3ltefuwswk"
                },
                "path": "cash/send"
            }
        }
    ]
}
```

- `/api/blocks/last/<number>`: returns the last `number` of `Blocks`

```json
[
    {
        "fee_frac": 0,
        "hash": "0825bdadd1a6a31011813f2da726e7d7c2c2f5c8fe438525e08863380f3e4eb8",
        "height": 255,
        "proposer_name": "Cosmostation",
        "time": "2019-10-11T01:46:17.448255Z",
        "transactions": null
    },
    {
        "fee_frac": 0,
        "hash": "42270ea70a45ed13d0cd971aa152c5c649e1efc7f1278f14a5b7b658235284b8",
        "height": 254,
        "proposer_name": "Bianjie",
        "time": "2019-10-11T01:41:11.062501Z",
        "transactions": null
    }
]
```

- `/api/blocks/hash/<hash>`: returns the `Block` by `BlockHash`

```json
{
    "fee_frac": 0,
    "hash": "42270ea70a45ed13d0cd971aa152c5c649e1efc7f1278f14a5b7b658235284b8",
    "height": 254,
    "proposer_name": "Bianjie",
    "time": "2019-10-11T01:41:11.062501Z",
    "transactions": null
}
```

- `/api/txs/<hash>` returns the `Transaction` by `TransactionHash`

```json
{
    "block_height": 74,
    "hash": "7a2dadb0aa7c7cce31a7c26c6797fd3120175bc8f192b688e17b25b9ab0cc705",
    "message": {
        "details": {
            "metadata": {
                "schema": 1
            },
            "targets": [
                {
                    "address": "iov1cwjx9wrd3tndzj45x90hfc4u7vjph3etycrmqx",
                    "blockchain_id": "iov-mainnet"
                },
                {
                    "address": "17747612735376126537L",
                    "blockchain_id": "lisk-ed14889723"
                },
                {
                    "address": "0x522BE285D2B406d78a18C01c21141576aF17E2Cb",
                    "blockchain_id": "ethereum-eip155-1"
                }
            ],
            "username": "bit999*iov"
        },
        "path": "username/register_token"
    }
}
```

# Sample queries

First run the above command to fill the database with all the sample hugnet data, then:

Find active validators at height h:

```sql
SELECT v.address 
    FROM validators v 
    INNER JOIN block_participations p ON v.id = p.validator_id 
    WHERE p.block_id = 57;
```

Find all missing precommits (over all validators and blocks)

```sql
SELECT * FROM block_participations WHERE validated = false;

SELECT b.block_height, b.proposer_id, p.validator_id, b.block_time 
    FROM block_participations p 
    INNER JOIN blocks b ON p.block_id = b.block_height 
    WHERE p.validated = false;
```

Find total counts for each validator:

```sql
SELECT COUNT(NULLIF(validated, false)) as signed, COUNT(NULLIF(validated, true)) as missed, validator_id 
    FROM block_participations 
    GROUP BY validator_id 
    ORDER BY validator_id;

SELECT v.address, COUNT(NULLIF(p.validated, false)) as signed, COUNT(NULLIF(p.validated, true)) as missed 
    FROM block_participations p 
    INNER JOIN validators v ON p.validator_id = v.id 
    GROUP BY v.address 
    ORDER BY v.address;
```

Find missed by block proposer:

(by validator id)
```sql
SELECT b.proposer_id, COUNT(*) 
    FROM blocks b 
    INNER JOIN block_participations p ON b.block_height = p.block_id 
    WHERE p.validated = false 
    GROUP BY b.proposer_id
    ORDER BY count DESC;
```

(or with full address)
```sql
SELECT v.address, COUNT(*) 
    FROM validators v 
    INNER JOIN blocks b ON b.proposer_id = v.id 
    INNER JOIN block_participations p ON b.block_height = p.block_id 
    WHERE p.validated = false 
    GROUP BY v.address
    ORDER BY count DESC;
```

Find misses by proposer and signer:

```sql
SELECT b.proposer_id, p.validator_id, COUNT(*) 
    FROM blocks b 
    INNER JOIN block_participations p ON b.block_height = p.block_id 
    WHERE p.validated = false 
    GROUP BY b.proposer_id, p.validator_id;
```

Find misses by **next** proposer and signer: 
(next proposer makes the canonical commits, and note how this ensures no more self-censorship)


```sql
SELECT b.proposer_id, p.validator_id, COUNT(*) 
    FROM blocks b 
    INNER JOIN block_participations p ON b.block_height = p.block_id + 1
    WHERE p.validated = false 
    GROUP BY b.proposer_id, p.validator_id;
```
