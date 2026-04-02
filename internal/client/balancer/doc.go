// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

// Package balancer registers gRPC load-balancing policies for use with
// [github.com/H0llyW00dzZ/grpc-template/internal/client.WithLoadBalancing].
//
// Importing this package as a blank import registers every built-in
// gRPC balancer policy so that any of them can be selected at runtime:
//
//	import _ "github.com/H0llyW00dzZ/grpc-template/internal/client/balancer"
//
// If binary size matters and you only need a subset of policies,
// import the upstream gRPC balancer packages directly instead:
//
//	import _ "google.golang.org/grpc/balancer/roundrobin"         // round_robin
//	import _ "google.golang.org/grpc/balancer/weightedroundrobin" // weighted_round_robin
//
// # Available Policies
//
// The following policies are registered by this package:
//
//   - "pick_first" — default gRPC policy; sends all RPCs to a single backend
//   - "round_robin" — distributes RPCs across all resolved backends in order
//   - "weighted_round_robin" — distributes RPCs proportionally based on backend-reported weights
//   - "least_request_experimental" — sends RPCs to the backend with fewest outstanding requests
//   - "ring_hash_experimental" — consistent-hashing ring; useful for cache-friendly routing
package balancer
