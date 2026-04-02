// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package balancer

import (
	// Register all built-in gRPC load-balancing policies.
	// Each package's init() calls balancer.Register, making the policy
	// available to [client.WithLoadBalancing] at runtime.
	_ "google.golang.org/grpc/balancer/leastrequest"      // least_request_experimental
	_ "google.golang.org/grpc/balancer/pickfirst"          // pick_first
	_ "google.golang.org/grpc/balancer/ringhash"           // ring_hash_experimental
	_ "google.golang.org/grpc/balancer/roundrobin"         // round_robin
	_ "google.golang.org/grpc/balancer/weightedroundrobin" // weighted_round_robin
)
