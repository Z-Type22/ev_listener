# ev_listener

> A set of microservices for receiving EVM network events, delivering them through Kafka, and exposing a transaction API.

The project contains Go services that subscribe to smart-contract events, publish them to Kafka, and store and expose the data through a REST API. Authentication is handled by a dedicated gRPC SSO service using JWT.

---

## Structure

```
ev_listener/                  # Project root
├── protos/                   # Protobuf contracts and generated Go code
│   ├── proto/sso/             # SSO gRPC definition
│   └── gen/go/                # Generated protobuf files
├── sso/                       # Authentication gRPC service
│   ├── cmd/sso/               # Service entry point
│   ├── cmd/migrator/          # Database migration command
│   ├── config/local.yaml      # Local configuration
│   ├── migrations/            # SSO SQL migrations
│   └── tools/certs/           # JWT public and private PEM keys
├── rest/                      # Transactions REST API
│   ├── cmd/rest/              # HTTP server entry point
│   ├── cmd/migrator/          # Database migration command
│   ├── config/local.yaml      # Local configuration
│   ├── docs/                  # Swagger specification
│   ├── migrations/            # REST API SQL migrations
│   └── tools/certs/           # JWT public PEM key
├── pusher/                    # EVM contract event listener
│   ├── cmd/pusher/            # Service entry point
│   ├── config/local.yaml      # Local configuration
│   └── tools/                 # Contract ABI files
├── .gitignore                 # Ignored local JSON and PEM files
└── README.md                  # Project documentation
```

## Requirements

- Go 1.26.3;
- PostgreSQL for `sso` and `rest`;
- Apache Kafka;
- An EVM node with WebSocket RPC access for `pusher`;
- JWT keys and contract ABI files.
