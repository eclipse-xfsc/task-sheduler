#!/bin/bash

# The script is used to generate Goa service code from
# Goa DSL definitions when the project uses dependencies from
# the `vendor` directory.
# Goa doesn't work well with `vendor` by default.

set -e

# preserve the value of GOFLAGS
STORED_GOFLAGS=$(go env GOFLAGS)

# force goa not to use vendor deps during generation
go env -w GOFLAGS=-mod=mod

# execute goa code generation
goa gen github.com/eclipse-xfsc/task-sheduler/design

# restore the value of GOFLAGS
go env -w GOFLAGS=$STORED_GOFLAGS
