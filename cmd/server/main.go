package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"gRPC-Playground/service"
	"log"
	"net"
	"os"
	"time"

	pb "gRPC-Playground/ecommerce"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

const (
	secretKey     = "secret"
	tokenDuration = 15 * time.Minute
)

/*
// Fuction to implement Server-Side TLS
// loadServerSideTLSCredentials function returns a TranportCredentials object or an error
// It is implements a server side TLS
// Note: In Server-Side TLS, only the server shares its certificate with the client.
func loadServerSideTLSCredentials() (credentials.TransportCredentials, error) {
	//For sever side TLS, load server’s certificate and private key.
	// use tls.LoadX509KeyPair() function to load the server-cert.pem and
	// server-key.pem files from the cert folder.
	serverCert, err := tls.LoadX509KeyPair("cert/server-cert.pem", "cert/server-key.pem")

	// check for errors
	if err != nil {
		return nil, err
	}

	// Else, we will create the transport credentials from them.
	// Make a tls.Config object with the server certificate, and
	// we set the ClientAuth field to NoClientCert since we’re just using server-side TLS.
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.NoClientCert,
	}

	// call credentials.NewTLS() with that config and return it to the caller.
	return credentials.NewTLS(config), nil

}

*/
// Function to implement Mutual TLS
// loadMutualTLSCredentials function returns a TranportCredentials object or an error
// It is implements Mutual TLS
// For mutual TLS, the client also has to share its certificate with the server.
func loadMutualTLSCredentials() (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed client's certificate
	// In our case: It is just one single CA that signs both the client and server
	pemClientCA, err := os.ReadFile("cert/ca-cert.pem")
	if err != nil {
		return nil, err
	}

	// create a new certificate pool.
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemClientCA) {
		return nil, fmt.Errorf("failed to add client CA's certificate")
	}

	// load server’s certificate and private key.
	// use tls.LoadX509KeyPair() function to load the server-cert.pem and
	// server-key.pem files from the cert folder.
	serverCert, err := tls.LoadX509KeyPair("cert/server-cert.pem", "cert/server-key.pem")

	// check for errors
	if err != nil {
		return nil, err
	}

	// Else, we will create the transport credentials from them.
	// Make a tls.Config object with the server certificate, and
	// we set the ClientAuth field to NoClientCert since we’re just using server-side TLS.
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}

	// call credentials.NewTLS() with that config and return it to the caller.
	return credentials.NewTLS(config), nil

}

func main() {
	// use the flag.Int() function to get port from command line arguments.
	port := flag.Int("port", 0, "the server port")

	// enable TLS on our gRPC server or not.
	enableTLS := flag.Bool("tls", false, "enable SSL/TLS")

	flag.Parse()
	log.Printf("start server on port %d", *port)

	// Create a new user InMemoryStore
	userStore := service.NewInMemoryUserStore()
	// Call SeedUser to create a new user and store in the InMemoryUserStore
	err := service.SeedUsers(userStore)
	if err != nil {
		log.Fatal("cannot seed users: ", err)
	}

	// create a new laptop store with an in-memory laptop store.
	laptopStore := service.NewInMemoryLaptopStore()

	// create a new ImageStore store with an NewDiskImageStore.
	imageStore := service.NewDiskImageStore("assets")

	// create a new rating store with an in-memory rating store.
	ratingStore := service.NewInMemoryRatingStore()

	// Create a new JWTManager
	jwtManager := service.NewJWTManager(secretKey, tokenDuration)

	// Create a new auth server
	authServer := service.NewAuthServer(userStore, jwtManager)

	// create a new laptop server with an in-memory laptop store.
	laptopServer := service.NewLaptopServer(laptopStore, imageStore, ratingStore)

	// Retrieve the accessible roles list
	accessibleRoles := service.AccessibleRoles()

	// create a new interceptor object with the jwt manager and a map of accessible roles.
	interceptor := service.NewAuthInterceptor(jwtManager, accessibleRoles)

	// call loadMutualTLSCredentials() to get the Mutual TLS credential object.
	// Note: To load Server-Side TLS, use loadServerSideTLSCredentials function
	//tlsCredentials, err := loadMutualTLSCredentials()

	// log error
	//if err != nil {
	//	log.Fatal("cannot load TLS Credentials: ", err)
	//}

	// serverOptions slice to hold the our interceptors
	serverOptions := []grpc.ServerOption{
		grpc.UnaryInterceptor(interceptor.Unary()),
		grpc.StreamInterceptor(interceptor.Stream()),
	}

	if *enableTLS {
		// call loadMutualTLSCredentials() to get the Mutual TLS credential object.
		// Note: To load Server-Side TLS, use loadServerSideTLSCredentials function
		tlsCredentials, err := loadMutualTLSCredentials()

		// log error
		if err != nil {
			log.Fatal("cannot load TLS Credentials: ", err)
		}

		// append the Mutual/Server-Side TLS credential to the gRPC server 
		// by using the grpc.Creds() option.
		serverOptions = append(serverOptions, grpc.Creds(tlsCredentials))

	}

	// create the gRPC server by calling grpc.NewServer() function.
	// pass the our interceptors via serverOptions variable 
	//grpcServer := grpc.NewServer(
		// add the server-side TLS credential to the gRPC server by using the grpc.Creds() option.
		//grpc.Creds(tlsCredentials),
		// Registering the unary interceptor with the gRPC server.
		//grpc.UnaryInterceptor(interceptor.Unary()),
		// Registering the stream interceptor with the gRPC server.
		//grpc.StreamInterceptor(interceptor.Stream()),
	//)

	// create the gRPC server by calling grpc.NewServer() function.
	// pass the our interceptors via serverOptions variable 
	grpcServer := grpc.NewServer(serverOptions...)

	// call pb.RegisterAuthServiceServer to add it to the gRPC server.
	pb.RegisterAuthServiceServer(grpcServer, authServer)
	// register the laptop service server on that gRPC server.
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)

	// Enable gRPC Reflection on the server
	/*
		gRPC reflection is an optional extension for the server to assist clients
		to construct requests without having to generate stubs beforehand.
		This is very useful for the clients to explore the gRPC API before actually going into implementation.

		It is used by gRPC CLI, which can be used to introspect server protos and send/receive test RPCs.
	*/
	reflection.Register(grpcServer)

	// create an address string with the port
	address := fmt.Sprintf("0.0.0.0:%d", *port)

	// listen for TCP connections on this server address.
	listener, err := net.Listen("tcp", address)

	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Printf("Starting gRPC listener on address %v", address)

	// Finally we call grpcServer.Serve() to start the server.
	err = grpcServer.Serve(listener)
	// If any error occurs, just write a fatal log and exit.
	if err != nil {
		log.Fatal("cannot start server: ", err)
	}

}
