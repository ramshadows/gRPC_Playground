package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	pb "gRPC-Playground/ecommerce"

	wrapper "github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	address = "localhost:50051"
)

func main() {

	// To call service methods, we first need to create a
	// gRPC channel to communicate with the server. We create
	// this by passing the server address and port number to grpc.Dial() as follows:
	// Note: You can use DialOptions to set the auth credentials
	// (for example, TLS, GCE credentials, or JWT credentials) in grpc.Dial
	// when a service requires them. The ProductInfo service doesn’t require any credentials.

	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer conn.Close()

	// Once the gRPC channel is setup, we need a client stub to perform RPCs. We get it using
	// the NewProductInfoClient method provided by the pb package generated from the .proto file.
	c := pb.NewOrderManagementClient(conn)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)

	defer cancel()

	// Calling the simple RPC GetOrder
	retrievedOrder, err := c.GetOrder(
		ctx,
		&wrapper.StringValue{
			Value: "102",
		},
	)

	if err != nil {
		log.Fatalf("Could not get order with order id: %v", err)
	}

	log.Printf("Order ID: %s\n", retrievedOrder.Id)
	log.Printf("Order Items: %q\n", retrievedOrder.Items[:])
	log.Printf("Order Description: %s\n", retrievedOrder.Description)
	log.Printf("Order Price: %.2f\n", retrievedOrder.Price)
	log.Printf("Order Destination: %s\n", retrievedOrder.Destination)

	// searchOrder rpc client
	searchStream, _ := c.SearchOrders(
		ctx,
		&wrapper.StringValue{
			Value: "Google",
		},
	)

	// we retrieve messages from the client-side stream using the Recv() method
	// and keep doing so until we reach the end of the stream.
	for {

		// Calling the client stream’s Recv() method to retrieve Order responses one by one.
		searchOrder, err := searchStream.Recv()

		// When the end of the stream is found Recv returns an io.EOF.
		if err == io.EOF {
			log.Print("EOF")
			break
		}

		// Otherwise
		log.Print("Search Result : ", searchOrder)
	}

	// UpdateOrders rpc client
	// Invoking UpdateOrders remote method.

	updateStream, err := c.UpdateOrders(ctx)

	if err != nil {
		log.Fatalf("%v.UpdateOrders(_) = _, %v", c, err)
	}

	// *** Define Orders to Update ***
	updOrder1 := pb.Order{Id: "102", Items: []string{"Google Pixel 3A", "Google Pixel Book"}, Destination: "Mountain View, CA", Price: 1100.00}

	updOrder2 := pb.Order{Id: "103", Items: []string{"Apple Watch S4", "Mac Book Pro", "iPad Pro"}, Destination: "San Jose, CA", Price: 2800.00}

	updOrder3 := pb.Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub", "iPad Mini"}, Destination: "Mountain View, CA", Price: 2200.00}

	// Updating order 1
	if err := updateStream.Send(&updOrder1); err != nil {
		log.Fatalf("%v.Send(%v) = %v", updateStream, &updOrder1, err)
	}

	// the client can send multiple messages via the client-side streamreference
	// using the updateStream.Send method.

	// Updating order 2
	if err := updateStream.Send(&updOrder2); err != nil {
		log.Fatalf("%v.Send(%v) = %v", updateStream, &updOrder2, err)
	}

	// Updating order 3
	if err := updateStream.Send(&updOrder3); err != nil {
		log.Fatalf("%v.Send(%v) = %v", updateStream, &updOrder3, err)
	}

	// Once all the messages are streamed the client can mark the
	// end of the stream and receive the response from the service.
	// This is done using the CloseAndRecv method of the stream reference.

	updateRes, err := updateStream.CloseAndRecv()

	if err != nil {
		log.Fatalf("%v.CloseAndRecv() got error %v, want %v", updateStream, err, nil)
	}

	// Otherwise
	log.Printf("Update Orders Res : %s", updateRes)

	// =========================================
	// Process Order : Bi-di streaming scenario

	// Invoke the remote method processOrders and obtain the stream reference for
	// writing and reading from the client side.
	streamProcessOrder, err := c.ProcessOrders(ctx)

	if err != nil {
		fmt.Println("Am inside here")
		log.Fatalf("%v.ProcessOrders(_) = _, %v", c, err)
	}

	// If no error, begin sending order messages/streams for processing
	if err := streamProcessOrder.Send(
		&wrapper.StringValue{Value: "102"}); err != nil {
		log.Fatalf("%v.Send(%v) = %v", c, "102", err)
	}

	if err := streamProcessOrder.Send(
		&wrapper.StringValue{Value: "103"}); err != nil {
		log.Fatalf("%v.Send(%v) = %v", c, "103", err)
	}

	if err := streamProcessOrder.Send(
		&wrapper.StringValue{Value: "104"}); err != nil {
		log.Fatalf("%v.Send(%v) = %v", c, "104", err)
	}

	// Create a channel for use with goroutines
	channel := make(chan bool, 1)

	// Invoke the function using Goroutines to read the messages in parallel from the service.
	go asyncClientBidirectionalRPC(streamProcessOrder, channel)

	// Mimic a delay when sending some messages to the service.
	time.Sleep(time.Millisecond * 1000)

	// Mark the end of stream for the client stream (order IDs).
	if err := streamProcessOrder.CloseSend(); err != nil {
		log.Fatal(err)
	}

	<-channel

}

func asyncClientBidirectionalRPC(streamProcessOrder pb.OrderManagement_ProcessOrdersClient, c chan bool)  {
	for {
		combinedShipment, err := streamProcessOrder.Recv()

		if err == io.EOF {
			break

		}

		log.Printf("Combined shipment : %s", combinedShipment.OrdersList)
	}

	c <- true
}
