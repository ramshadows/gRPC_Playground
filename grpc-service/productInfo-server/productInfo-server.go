package main

import (
	"context"
	"fmt"
	pb "gRPC-Playground/ecommerce"
	"log"
	"net"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"google.golang.org/grpc/status"
)

const (
	port = ":50051"
)

func main() {

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%v", port))

	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Create an instance of the gRPC server using grpc.NewServer(...)
	grpcServer := grpc.NewServer()

	// Register our service implementation with the gRPC server.
	pb.RegisterProductInfoServer(grpcServer, &productInfoServer{})

	log.Printf("Starting gRPC listener on port " + port)

	// Call Serve() on the server with our port details to do a blocking wait until
	// the process is killed or Stop() is called
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

type productInfoServer struct {
	pb.UnimplementedProductInfoServer
	productMap map[string]*pb.Product
}

func (server *productInfoServer) AddProduct(ctx context.Context, product *pb.Product) (*pb.ProductID, error) {
	// generate the product id
	prodId, err := uuid.NewUUID()

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error while generating product id", err)
	}

	// assign the generated id as the product id
	product.Id = prodId.String()

	// check to verify that the product is not empty
	// if so generate a new map to hold the product

	if server.productMap == nil {
		server.productMap = make(map[string]*pb.Product)
	}

	server.productMap[product.Id] = product

	return &pb.ProductID{Value: product.Id}, status.New(codes.OK, "").Err()

}

func (server *productInfoServer) GetProduct(ctx context.Context, productID *pb.ProductID) (*pb.Product, error) {
	// check if product exist not the default zero value
	value, exist := server.productMap[productID.Value]

	if exist {
		return value, status.New(codes.OK, "").Err()

	}

	return nil, status.Errorf(codes.NotFound, "Product not found.", productID.Value)
}
