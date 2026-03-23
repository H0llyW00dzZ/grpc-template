<?php
// GENERATED CODE -- DO NOT EDIT!

// Original file comments:
// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.
//
namespace Helloworld\V1;

/**
 * GreeterService is a sample gRPC service demonstrating unary and server-streaming RPCs.
 */
class GreeterServiceClient extends \Grpc\BaseStub {

    /**
     * @param string $hostname hostname
     * @param array $opts channel options
     * @param \Grpc\Channel $channel (optional) re-use channel object
     */
    public function __construct($hostname, $opts, $channel = null) {
        parent::__construct($hostname, $opts, $channel);
    }

    /**
     * SayHello sends a single greeting (unary RPC).
     * @param \Helloworld\V1\SayHelloRequest $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     * @return \Grpc\UnaryCall<\Helloworld\V1\SayHelloResponse>
     */
    public function SayHello(\Helloworld\V1\SayHelloRequest $argument,
      $metadata = [], $options = []) {
        return $this->_simpleRequest('/helloworld.v1.GreeterService/SayHello',
        $argument,
        ['\Helloworld\V1\SayHelloResponse', 'decode'],
        $metadata, $options);
    }

    /**
     * SayHelloServerStream sends multiple greetings over a server stream.
     * @param \Helloworld\V1\SayHelloServerStreamRequest $argument input argument
     * @param array $metadata metadata
     * @param array $options call options
     * @return \Grpc\ServerStreamingCall
     */
    public function SayHelloServerStream(\Helloworld\V1\SayHelloServerStreamRequest $argument,
      $metadata = [], $options = []) {
        return $this->_serverStreamRequest('/helloworld.v1.GreeterService/SayHelloServerStream',
        $argument,
        ['\Helloworld\V1\SayHelloServerStreamResponse', 'decode'],
        $metadata, $options);
    }

}
