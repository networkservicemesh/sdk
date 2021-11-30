package trace_test

// type myChainClient struct {
// 	saved *networkservice.NetworkServiceRequest
// }

// func NewClient() networkservice.NetworkServiceClient {
// 	return &myChainClient{}
// }

// func (c *myChainClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (conn *networkservice.Connection, err error) {
// 	return next.Client(ctx).Request(ctx, request, opts...)
// }

// func (c *myChainClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (emp *emptypb.Empty, err error) {
// 	return next.Client(ctx).Close(ctx, conn, opts...)
// }

// func TestTrace(t *testing.T) {
// 	t.Skip()
// 	ctx := context.Background()

// 	client1 := NewClient()
// 	client2 := NewClient()
// 	nextClient := next.NewNetworkServiceClient(client1)
// 	chainClient := chain.NewNetworkServiceClient(client2)

// 	request := &networkservice.NetworkServiceRequest{
// 		Connection: newConnection().Connection,
// 	}

// 	logrus.SetLevel(logrus.TraceLevel)
// 	log.EnableTracing(true)

// 	start := time.Now()
// 	nextClient.Request(ctx, request)
// 	elapsed := time.Since(start)
// 	fmt.Printf("Next request. Elapsed time: %v\n", elapsed)

// 	start = time.Now()
// 	chainClient.Request(ctx, request)
// 	elapsed = time.Since(start)
// 	fmt.Printf("Chain request. Elapsed time: %v\n", elapsed)
// }

// func TestTrace2(t *testing.T) {
// 	t.Skip()
// 	ctx := context.Background()

// 	client1 := NewClient()
// 	client2 := NewClient()
// 	nextClient := next.NewNetworkServiceClient(client1, client1, client1, client1, client1, client1)
// 	chainClient := chain.NewNetworkServiceClient(client2, client2, client2, client2, client2, client2)

// 	request := &networkservice.NetworkServiceRequest{
// 		Connection: newConnection().Connection,
// 	}

// 	logrus.SetLevel(logrus.TraceLevel)
// 	log.EnableTracing(true)

// 	start := time.Now()
// 	nextClient.Request(ctx, request)
// 	elapsed := time.Since(start)
// 	fmt.Printf("Next request. Elapsed time: %v\n", elapsed)

// 	start = time.Now()
// 	chainClient.Request(ctx, request)
// 	elapsed = time.Since(start)
// 	fmt.Printf("Chain request. Elapsed time: %v\n", elapsed)
// }
