package main

import (
	"context"
	"fmt"
	pb "gRPC-Playground/ecommerce"
	"io"
	"log"
	"net"
	"strings"

	wrapper "github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	port           = "50051"
	orderBatchSize = 3
)

var (
	orderMap            = make(map[string]*pb.Order)
	combinedShipmentMap = make(map[string]*pb.CombinedShipment)
)

func main() {
	// initialize our sample data
	initSampleData()

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%v", port))

	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Create an instance of the gRPC server using grpc.NewServer(...)
	grpcServer := grpc.NewServer(
		// Registering the unary interceptor with the gRPC server.
		grpc.UnaryInterceptor(orderUnaryServerInterceptor),
	)

	// Register our service implementation with the gRPC server.
	pb.RegisterOrderManagementServer(grpcServer, &orderManagementServer{})

	log.Printf("Starting gRPC listener on port " + port)

	// Call Serve() on the server with our port details to do a blocking wait until
	// the process is killed or Stop() is called
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

type orderManagementServer struct {
	pb.UnimplementedOrderManagementServer
	//orderMap map[string]*pb.Order
}

func (server *orderManagementServer) GetOrder(ctx context.Context, orderId *wrapper.StringValue) (*pb.Order, error) {

	// check if product exist not the default zero value
	requestedOrder, exist := orderMap[orderId.Value]

	if exist {
		return requestedOrder, status.New(codes.OK, "").Err()

	}

	return nil, status.Errorf(codes.NotFound, "Order with %s not found.", orderId.Value)
}

// The searchOrders method takes a searchQuery as string and a OrderManagement_SearchOrdersServer
// a reference object to the stream that we can write multiple responses to.
// The business logic here is to ﬁnd the matching orders and send them one by one
// via the stream. When a new order is found, it is written to the stream using the Send(…) method of the
// stream reference object. Once all the responses are written to the stream you can mark the end of the
// stream by returning nil, and the server status and other trailing metadata will be sent to the client.
func (server *orderManagementServer) SearchOrders(searchQuery *wrapper.StringValue, stream pb.OrderManagement_SearchOrdersServer) error {

	for key, order := range orderMap {
		log.Print(key, order)

		for _, itemStr := range order.Items {
			log.Print(itemStr)

			// if the itemStr == searchQuery
			if strings.Contains(itemStr, searchQuery.Value) {
				// Send the matching orders in a stream using the Send() method
				err := stream.Send(order)

				if err != nil {
					return fmt.Errorf("error sending message through the stream: %v", err)
				}

				// Otherwise
				log.Print("Matching order found: " + key)
				break
			}

		}

	}

	return nil

}

// The UpdateOrders method takes a OrderManagement_UpdateOrdersServer
// a reference object to the incoming message stream from the client.
// The business logic here read a few messages or all the messages until the end of the stream
// The service can send its response simply by calling the SendAndClose method of the
// OrderManagement_UpdateOrdersServer object, which also marks the end of the stream for server-side messages.
// If the server decides to prematurely stop reading from the client’s stream, the server should cancel the client
// stream so the client knows to stop producing messages.
func (server *orderManagementServer) UpdateOrders(stream pb.OrderManagement_UpdateOrdersServer) error {

	ordersStr := "Updated Orders IDs: "

	for {
		// read incoming message stream from the client.
		order, err := stream.Recv()

		// Check for end of stream.
		if err == io.EOF {

			// The service marks the end of the stream for server-side messages.
			// by calling the SendAndClose method of the OrderManagement_UpdateOrdersServer object
			return stream.SendAndClose(
				&wrapper.StringValue{
					Value: "Orders Processed: " + ordersStr + "\n",
				},
			)

		}

		// Update changes to our orderMap
		orderMap[order.Id] = order

		log.Println("Order ID: ", order.Id, ": Updated")
		ordersStr += order.Id + ", "
	}

}

// processOrders has an OrderManagement_ProcessOrdersServer parameter, which is
// the reference object to the message stream between the client and the service.
// Using this stream object, the service can read the client’s messages that are
// streamed to the server as well as write the stream server’s messages back to
// the client. Using that stream reference object, the incoming message stream
// can be read using the Recv() method. In the processOrders method, the service
// can keep on reading the incoming message stream while writing to the
// same stream using Send.
func (server *orderManagementServer) ProcessOrders(stream pb.OrderManagement_ProcessOrdersServer) error {
	batchMarker := 1

	for {

		// Read order IDs from the incoming stream.
		orderId, err := stream.Recv()

		log.Printf("Reading process order : %s", orderId)

		// Keep reading until the end of the stream is found.
		if err == io.EOF {
			log.Print("no more data")

		}

		for _, comb := range combinedShipmentMap {
			// When the end of the stream is found send all the
			// remaining combined shipments to the client.
			stream.Send(comb)
		}

		if err != nil {
			return status.Errorf(codes.Unknown, "cannot receive stream request: %v", err)
		}

		//Logic to organize orders into shipments, based on the destination.
		destination := orderMap[orderId.GetValue()].Destination

		// check if the destination exist, not the default zero value
		shipment, found := combinedShipmentMap[destination]

		if found {
			ord := orderMap[orderId.GetValue()]
			fmt.Println("Order is: ", ord)
			shipment.OrdersList = append(shipment.OrdersList, ord)
			combinedShipmentMap[destination] = shipment
		} else {
			comShip := pb.CombinedShipment{
				Id: "cmb - " + (orderMap[orderId.GetValue()].Destination), Status: "Processed!",
			}
			ord := orderMap[orderId.GetValue()]
			comShip.OrdersList = append(shipment.OrdersList, ord)
			combinedShipmentMap[destination] = &comShip
			log.Print(len(comShip.OrdersList), comShip.GetId())
		}

		if batchMarker == orderBatchSize {
			for _, comb := range combinedShipmentMap {
				log.Printf("Shipping : %v -> %v", comb.Id, len(comb.OrdersList))
				if err := stream.Send(comb); err != nil {
					return err
				}
			}
			batchMarker = 0
			combinedShipmentMap = make(map[string]*pb.CombinedShipment)
		} else {
			batchMarker++
		}
	}

}

// orderUnaryServerInterceptor func is server-side unary interceptor
// Note: Interceptors in gRPC’s helps to implement certain requirements such as
// logging, authentication, authorization, metrics, tracing,
// Method Signature to implement
// *****************
// func(ctx context.Context, req interface{}, info *UnaryServerInfo, handler UnaryHandler) (resp
// interface{}, err error)
/// ********************
func orderUnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	// *** Phase 1: Preprocessing logic
	// Preprocessing phase: this is where you can intercept the message prior to invoking the
	// respective RPC. You can implement your logging, authorizatio, metrics e.tc from here
	// Example case: Gets info about the current RPC call by examining the args passed in
	// by calling the info.fullMethod
	log.Println("+++  Server-Side Unary Interceptor +++", info.FullMethod)
	log.Printf(" Pre Process Message : %s", req)

	// *** Phase 2: Invoking the handler to complete the normal execution of a unary RPC
	// via UnaryHandler

	m, err := handler(ctx, req)

	// *** Phase 3: Post Processing logic - Here you can process the response from the RPC invocation.
	// This means that the response for the RPC call goes through the postprocessor phase.
	// In the phase, you can deal with the returned reply (m) and error (err) when required.
	// Once the postprocessor phase is completed, you need to return the message (m) and the error (err)
	// as the return parameters of your interceptor function. If no postprocessing is required,
	// you can simply return the handler call - return handler(ctx, req).
	// Example case: Sending back the RPC response.
	log.Printf(" Post Proc Message : %s", m)
	return m, err

	// *** Phase 4: Registering the unary interceptor with the gRPC server. - see main()

}

func initSampleData() {

	orderMap["102"] = &pb.Order{Id: "102", Items: []string{"Google Pixel 3A", "Mac Book Pro"}, Destination: "Mountain View, CA", Price: 1800.00}
	orderMap["103"] = &pb.Order{Id: "103", Items: []string{"Apple Watch S4"}, Destination: "San Jose, CA", Price: 400.00}
	orderMap["104"] = &pb.Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub"}, Destination: "Mountain View, CA", Price: 400.00}
	orderMap["105"] = &pb.Order{Id: "105", Items: []string{"Amazon Echo"}, Destination: "San Jose, CA", Price: 30.00}
	orderMap["106"] = &pb.Order{Id: "106", Items: []string{"Amazon Echo", "Apple iPhone XS"}, Destination: "Mountain View, CA", Price: 300.00}

}
