package main

import (
	"context"
	"log"
	"time"

	pb "gRPC-Playground/ecommerce"

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
	c := pb.NewProductInfoClient(conn)

	// Manually defining product values to pass to our RPC method AddProduct()
	name := "Apple iPhone 11"
	description := `Meet Apple iPhone 11. All-new dual-camera system
	                with Ultra Wide and Night mode.`

	price := float32(1000.0)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)

	defer cancel()

	// Calling the simple RPC AddProduct
	r, err := c.AddProduct(
		ctx,
		&pb.Product{
			Name:        name,
			Description: description,
			Price:       price,
		},
	)

	if err != nil {
		log.Fatalf("Could not add product: %v", err)
	}

	log.Printf("Product ID: %s added successfully", r.Value)

	// // Calling the simple RPC GetProduct
	product, err := c.GetProduct(
		ctx, 
		&pb.ProductID{
			Value: r.Value,
		},
	)

	if err != nil {
		log.Fatalf("Could not get product: %v", err)
	}

	log.Printf("Product ID: %s\n", product.Id)
	log.Printf("Product Name: %s\n", product.Name)
	log.Printf("Product Desc: %s\n", product.Description)
	log.Printf("Product Price: %.2f\n", product.Price)

}
