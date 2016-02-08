# Session

The [Go](https://golang.org/) standard library includes a nice [http server](https://golang.org/pkg/net/http/), but unfortunately it lacks a very basic and important feature: _HTTP session management_.

This package provides an easy-to-use, extensible and secure session implementation and management.

## Overview

There are 3 key _players_ in the package:

- **`Session`** is the (HTTP) session interface. We can use it to store and retrieve constant and variable attributes from it.
- **`Store`** is a session store interface which is responsible to store sessions and make them retrievable by their IDs at the server side.
- **`Manager`** is a session manager interface which is responsible to acquire a `Session` from an (incoming) HTTP request, and to add a `Session` to an HTTP response to let the client know about the session. A `Manager` has a backing `Store` which is responsible to manage `Session` values at server side.

_Players_ of this package are represented by interfaces, and various implementations are provided for all these players.
You are not bound by the provided implementations, feel free to provide your own implementations for any of the players.
