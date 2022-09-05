package client

import (
	"bufio"
	"context"
	"fmt"
	pb "gRPC-Playground/ecommerce"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LaptopClient is a client to call laptop service RPCs
type LaptopClient struct {
	service pb.LaptopServiceClient
}

// NewLaptopClient returns a new laptop client
func NewLaptopClient(cc *grpc.ClientConn) *LaptopClient {
	service := pb.NewLaptopServiceClient(cc)
	return &LaptopClient{service}
}

func (laptopClient *LaptopClient) CreateLaptopClient(laptop *pb.Laptop) string {
	// generate a new laptop
	//laptop := sampledata.NewLaptop()

	// make a new request object,
	req := &pb.CreateLaptopRequest{
		Laptop: laptop,
	}

	// set timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	// *** Implementing our CreateLaptop unary rpc remote method
	// call laptopClient.Createlaptop() our unary RPC remote method with the request and a context
	res, err := laptopClient.service.CreateLaptop(ctx, req)
	// If error is not nil, we convert it into a status object.
	if err != nil {
		st, ok := status.FromError(err)
		// If the status code is AlreadyExists then it’s not a big deal, just write a normal log
		if ok && st.Code() == codes.AlreadyExists {
			// not a big deal
			log.Print("laptop already exists")
		} else {
			// Else, we write a fatal log.
			log.Fatal("cannot create laptop: ", err)
		}
		return res.Id
	}

	log.Printf("created laptop with id: %s", res.Id)
	return res.Id
}

func (laptopClient *LaptopClient) GetLaptopByIDClient(laptopID string) {
	// set timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	// *** Implementing our GetLaptopByID unary rpc remote method
	// call laptopClient.GetLaptopByID our unary RPC remote method with the request and a context
	retrievedLaptop, err := laptopClient.service.GetLaptopByID(
		ctx,
		&pb.GetLaptopByIDRequest{
			Id: laptopID,
		},
	)

	if err != nil {
		log.Fatalf("Could not get Laptop with id: %v", err)
	}

	log.Println("*******Requested Laptop By ID ******. ID : ", retrievedLaptop.Laptop.Id)

	log.Printf("Laptop Brand: %s\n", retrievedLaptop.Laptop.GetBrand())
	log.Printf("Laptop Name: %s\n", retrievedLaptop.Laptop.GetName())
	log.Printf("CPU Details: %s\n", retrievedLaptop.Laptop.GetCpu())
	log.Printf("Ram Size: %s\n", retrievedLaptop.Laptop.GetRam())
	log.Printf("Screen Details: %s\n", retrievedLaptop.Laptop.GetScreen())

	

}

func (laptopClient *LaptopClient) SearchLaptopClient() {
	//log.Print("search filter: ", filter)
	// set timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	// call laptopClient.SearchLaptop server streaming RPC remote method with the request
	// and a context
	searchedStream, err := laptopClient.service.SearchLaptop(
		ctx,
		&pb.SearchLaptopRequest{
			Filter: &pb.Filter{
				MaxPriceUsd: 3000,
				MinCpuCores: 4,
				MinCpuGhz:   2.5,
				MinRam: &pb.Memory{
					Value: 8,
					Unit:  pb.Memory_GIGABYTE,
				},
			},
		},
	)

	if err != nil {
		log.Fatal("cannot search laptop: ", err)
	}

	// we send messages streams from the server-side stream to the client
	// using the client Recv() method which retrieves stream messages from the server-side
	// and keep doing so until we reach the end of the stream.
	for {
		res, err := searchedStream.Recv()

		// When the end of the stream is found Recv returns an io.EOF.
		if err == io.EOF {
			log.Print("EOF")
			return
		}

		// Otherwise, if error is not nil, we write a fatal log.
		if err != nil {
			log.Fatal("cannot receive response: ", err)
		}

		// Otherwise. print search result
		log.Println("*****   Search Results *******")

		laptop := res.GetLaptop()
		log.Print("- found: ", laptop.GetId())
		log.Print("  + brand: ", laptop.GetBrand())
		log.Print("  + name: ", laptop.GetName())
		log.Print("  + cpu cores: ", laptop.GetCpu().GetNumberCores())
		log.Print("  + cpu min ghz: ", laptop.GetCpu().GetMinGhz())
		log.Print("  + ram: ", laptop.GetRam())
		log.Print("  + price: ", laptop.GetPriceUsd())

	}
}

// uploadImageClient() function uploads an image of a laptop to the server.
func (laptopClient *LaptopClient) UploadImageClient(laptopID string, imagePath string) {
	// call os.Open() to open the image file
	file, err := os.Open(imagePath)

	// If there’s an error, we write a fatal log.
	if err != nil {
		log.Fatal("cannot open image file: ", err)
	}

	// Else, we use defer() to close the file afterward.
	defer file.Close()

	// create a context with timeout of 5 seconds,
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// call our UploadImage client-side streaming RPC remote method with the context
	// It will return a stream object, and an error.
	stream, err := laptopClient.service.UploadImage(ctx)

	// If error is not nil, we write a fatal log.
	if err != nil {
		log.Fatal("cannot upload image: ", err)
	}

	// Otherwise, we create the first request to send some image information to the server,
	// which includes the laptop ID, and the image type, or the extension of the image file
	req := &pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_Info{
			Info: &pb.ImageInfo{
				LaptopId:  laptopID,
				ImageType: filepath.Ext(imagePath),
			},
		},
	}

	// we call stream.Send() to send the first request to the server.
	err = stream.Send(req)

	// If we get an error, write a fatal log.
	if err != nil {
		log.Fatal("cannot send image info to server: ", err, stream.RecvMsg(nil))
	}

	// Then we will create a buffer reader to read the content of the image file in chunks.
	reader := bufio.NewReader(file)

	// Make each chunk to be 1 KB, or 1024 bytes.
	buffer := make([]byte, 1024)

	// read the image data chunks sequentially in a for loop:
	for {
		// call reader.Read() to read the data to the buffer.
		// It will return the number of bytes read and an error.
		n, err := reader.Read(buffer)

		// If the error is EOF, then it’s the end of the file, we simply break the loop.
		if err == io.EOF {
			break
		}

		// Otherwise, we create a new request with the chunk data.
		// Make sure that the chunk only contains the first n bytes of the buffer,
		// since the last chunk might contain less than 1024 bytes.
		req := &pb.UploadImageRequest{
			Data: &pb.UploadImageRequest_ChunkData{
				ChunkData: buffer[:n],
			},
		}

		// Then we call stream.Send() to send it to the server.
		err = stream.Send(req)
		if err != nil {
			// use stream.RecvMsg(nil) to get more info on the error
			log.Fatal("cannot send chunk to server: ", err, stream.RecvMsg(nil))
		}

	}

	// Finally, after the for loop, We call stream.CloseAndRecv() to receive a
	// response from the server:
	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatal("cannot receive response: ", err)
	}

	// If there's no error, we write a log saying that the image is successfully uploaded,
	// and the server returns this ID and size.
	log.Printf("image uploaded with id: %s, size: %d", res.GetId(), res.GetSize())
}

// rateLaptopClient() function with 3 input parameters: a laptop client,
// a list of laptop IDs and their corresponding scores.
func (laptopClient *LaptopClient) RateLaptopClient(laptopIDs []string, scores []float64) error {
	// create a context with timeout of 5 seconds,
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// call laptopClient.RateLaptop() with the created context.
	// The output is a stream or an error.
	stream, err := laptopClient.service.RateLaptop(ctx)

	// If error is not nil, we just return it.
	if err != nil {
		return fmt.Errorf("cannot rate laptop: %v", err)
	}

	// Otherwise, make a channel to wait for the responses from the server.
	// The waitResponse channel will receive an error when it occurs, or a nil
	// if all responses are received successfully.
	waitResponse := make(chan error)

	// Since the requests and responses are sent concurrently, we start a new go routine
	// to receive the responses
	go func() {
		// use a for loop, and call stream.Recv() to get a response from the server.
		for {
			// call stream.Recv() to get a response from the server.
			res, err := stream.Recv()

			// If error is EOF, it means there’s no more responses,
			// so we send nil to the waitResponse channel, and return.
			if err == io.EOF {
				log.Print("no more responses")
				waitResponse <- nil
				return
			}
			// Else, if error is not nil, we send the error to the waitResponse channel,
			// and return as well.
			if err != nil {
				waitResponse <- fmt.Errorf("cannot receive stream response: %v", err)
				return
			}

			// If no errors occur, we just write a simple log.
			log.Print("received response: ", res)
		}

	}()

	// now after this go routine, we can start sending requests to the server.
	// Use a for loop to iterate through the list of the laptops and
	// create a new request for each of them with the input laptop ID
	// and the corresponding input scores.

	// send requests
	for i, laptopID := range laptopIDs {
		req := &pb.RateLaptopRequest{
			LaptopId: laptopID,
			Score:    scores[i],
		}

		// call stream.Send() to send the request to the server.
		err := stream.Send(req)

		// If we get an error, just return it.
		// Note that here we call stream.RecvMsg() to get the real error
		if err != nil {
			return fmt.Errorf("cannot send stream request: %v - %v", err, stream.RecvMsg(nil))
		}

		// If no error occurs, we write a log saying the request is sent.
		log.Print("sent request: ", req)

	}

	// call stream.CloseSend() to tell the server that we won’t send any more data.
	err = stream.CloseSend()
	if err != nil {
		return fmt.Errorf("cannot close send: %v", err)
	}

	// finally read from the waitResponse channel and return the received error.
	err = <-waitResponse
	return err

}
