#!/bin/bash
cd "$(dirname "$0")"
GO111MODULE=on GOMOD="$(pwd)/go.mod" go build -mod=mod -o generate_exchange_rates
