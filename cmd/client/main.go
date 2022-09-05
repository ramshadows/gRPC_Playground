package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"gRPC-Playground/client"
	sampledata "gRPC-Playground/sample-data"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	username        = "admin1"
	password        = "secret"
	refreshDuration = 30 * time.Second
)

/*
// loadServerSideTLSCredentials function load the certificate of the CA who signed
// the server’s certificate.
// Implements Server-side TLS
// For Server-side TLS, only the server shares it certificates with the client
// This verifys the authenticity of the certificate the client gets from
// the server to make sure that it’s the right server it wants to talk to.
func loadServerSideTLSCredentials() (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := os.ReadFile("cert/ca-cert.pem")

	// check for errors
	if err != nil {
		return nil, err
	}

	// create a new x509 cert pool
	certPool := x509.NewCertPool()

	// append the CA’s pem to that pool
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate: ")
	}

	// create the credentials
	config := &tls.Config{
		// Note that we only need to set the RootCAs field,
		// which contains the trusted CA’s certificate.
		RootCAs: certPool,
	}

	// return the credentials
	return credentials.NewTLS(config), nil

}
*/

// loadMutualTLSCredentials function load server and client certificates
// Implements Mutual TLS
// For mutual TLS, the client also has to share its certificate with the server.
func loadMutualTLSCredentials() (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := os.ReadFile("cert/ca-cert.pem")

	// check for errors
	if err != nil {
		return nil, err
	}

	// create a new x509 cert pool
	certPool := x509.NewCertPool()

	// append the CA’s pem to that pool
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate: ")
	}

	// Load client's certificate and private key
	clientCert, err := tls.LoadX509KeyPair("cert/client-cert.pem", "cert/client-key.pem")
	if err != nil {
		return nil, err
	}

	// create the credentials
	config := &tls.Config{
		// set the RootCAs field, which contains the trusted CA’s certificate.
		RootCAs: certPool,
		// add the client certificate to the TLS config
		Certificates: []tls.Certificate{clientCert},
	}

	// return the credentials
	return credentials.NewTLS(config), nil

}

func main() {
	serverAddress := flag.String("address", "", "the server address")

	// enable TLS on our gRPC server or not.
	enableTLS := flag.Bool("tls", false, "enable SSL/TLS")

	flag.Parse()
	log.Printf("dial server %s", *serverAddress)

	// call loadMutualTLSCredentials() to get the Mutual TLS credential object.
	// Note: To load Server-Side TLS, use loadServerSideTLSCredentials function
	//tlsCredentials, err := loadMutualTLSCredentials()

	// log error
	//if err != nil {
	//	log.Fatal("cannot load TLS Credentials: ", err)
	//}

	// transportOption variable with the default value grpc.WithInsecure().
	transportOption := grpc.WithTransportCredentials(insecure.NewCredentials())

	// Only when the enableTLS flag value is true, we load the TLS credentials 
	// from PEM files and change the transportOption to grpc.WithTransportCredentials(tlsCredentials).
	if *enableTLS {
		// call loadMutualTLSCredentials() to get the Mutual TLS credential object.
		// Note: To load Server-Side TLS, use loadServerSideTLSCredentials function
		tlsCredentials, err := loadMutualTLSCredentials()

	   // Log errors
		if err != nil {
			log.Fatal("cannot load TLS credentials: ", err)
		}

		// load the Mutual/Server-Side TLS credential to the gRPC Client 
		// by using the grpc.WithTransportCredentials
		transportOption = grpc.WithTransportCredentials(tlsCredentials)
	}

	// create separate connection for the auth client because it will be
	// used to create an auth interceptor, which will be used to create another
	// connection for the laptop client.
	conn1, err := grpc.Dial(
		*serverAddress,
		transportOption,
		//grpc.WithTransportCredentials(tlsCredentials),
	)
	if err != nil {
		log.Fatal("cannot dial server: ", err)
	}

	defer conn1.Close()

	authClient := client.NewAuthClient(conn1, username, password)

	// create a new interceptor with the auth client
	interceptor, err := client.NewAuthInterceptor(authClient, authMethods(), refreshDuration)
	if err != nil {
		log.Fatal("cannot create auth interceptor: ", err)
	}

	// dial server to create another connection. But this time,
	// we also add 2 dial options: the unary interceptor and the stream interceptor.
	conn2, err := grpc.Dial(
		*serverAddress,
		//grpc.WithTransportCredentials(tlsCredentials),
		transportOption,
		grpc.WithUnaryInterceptor(interceptor.Unary()),
		grpc.WithStreamInterceptor(interceptor.Stream()),
	)
	if err != nil {
		log.Fatal("cannot dial server: ", err)
	}

	// Once the gRPC channel is setup, we need a client stub to perform RPCs. We get it using
	// the NewLaptopServiceClient method provided by the pb package generated from the .proto file.
	laptopClient := client.NewLaptopClient(conn2)

	// call our unary RPC CreateLaptop remote method
	testCreateLaptop(laptopClient)

	// call our unary RPC GetLaptopByID remote method
	testGetLaptopByID(laptopClient)

	// call our server-streaming RPC SearchLaptop remote method
	testSearchLaptop(laptopClient)

	// *** Calling our bidirectional remote method gRPC RateLaptop()
	testRateLaptop(laptopClient)

	// call our client-side streaming UploadImage RPC remote method
	testUploadImage(laptopClient)

}

func authMethods() map[string]bool {
	const laptopServicePath = "/ecommerce.LaptopService/"

	return map[string]bool{
		laptopServicePath + "CreateLaptop": true,
		laptopServicePath + "UploadImage":  true,
		laptopServicePath + "RateLaptop":   true,
	}
}

func testCreateLaptop(laptopClient *client.LaptopClient) {
	laptop := sampledata.NewLaptop()
	laptopClient.CreateLaptopClient(laptop)
}

func testSearchLaptop(laptopClient *client.LaptopClient) {
	laptop := sampledata.NewLaptop()

	// create 10 laptops to search from
	for i := 0; i < 10; i++ {
		laptopClient.CreateLaptopClient(laptop)
	}
	laptopClient.SearchLaptopClient()

}

func testGetLaptopByID(laptopClient *client.LaptopClient) {
	laptop := sampledata.NewLaptop()
	laptopClient.CreateLaptopClient(laptop)
	laptopClient.GetLaptopByIDClient(laptop.GetId())

}

func testUploadImage(laptopClient *client.LaptopClient) {
	laptop := sampledata.NewLaptop()
	laptopClient.CreateLaptopClient(laptop)
	laptopClient.UploadImageClient(laptop.GetId(), "tmp/laptop.jpg")
}

func testRateLaptop(laptopClient *client.LaptopClient) {
	// Let’s say we want to rate 3 laptops,
	// so we declare a slice to keep the laptop IDs.
	n := 3
	laptopIDs := make([]string, n)

	// use a for loop to generate a random laptop,
	// save its ID to the slice, and call createLaptop() function
	// to create it on the server.
	for i := 0; i < n; i++ {
		laptop := sampledata.NewLaptop()
		laptopIDs[i] = laptop.GetId()
		laptopClient.CreateLaptopClient(laptop)
	}

	// Then we also make a slice to keep the scores. I want to rate these 3 laptops
	// in multiple rounds, so I will use a for loop here and
	// ask if we want to do another round of rating or not.
	scores := make([]float64, n)
	for {
		fmt.Print("rate laptop (y/n)? ")
		var answer string
		fmt.Scan(&answer)

		// If the answer is no, we break the loop.
		if strings.ToLower(answer) != "y" {
			break
		}

		// Else we generate a new set of scores for the laptops
		for i := 0; i < n; i++ {
			scores[i] = sampledata.RandomLaptopScore()
		}

		// call RateLaptopClient() function to rate them with the generated scores.
		err := laptopClient.RateLaptopClient(laptopIDs, scores)

		// If an error occurs, we write a fatal log.
		if err != nil {
			log.Fatal(err)
		}
	}
}
