package service_test

import (
	"bufio"
	"context"
	"fmt"
	pb "gRPC-Playground/ecommerce"
	sampledata "gRPC-Playground/sample-data"
	"gRPC-Playground/serializer"
	"gRPC-Playground/service"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

//Test the Unary RPC with a real connection
func TestClientCreateLaptop(t *testing.T) {
	t.Parallel()

	testImageFolder := "../tmp"

	laptopStore := service.NewInMemoryLaptopStore()
	imageStore := service.NewDiskImageStore(testImageFolder)
	ratingStore := service.NewInMemoryRatingStore()
	serverAddress := startTestLaptopServer(t, laptopStore, imageStore, ratingStore)
	laptopClient := newTestLaptopClient(t, serverAddress)

	laptop := sampledata.NewLaptop()

	expectedID := laptop.Id

	req := &pb.CreateLaptopRequest{
		Laptop: laptop,
	}

	res, err := laptopClient.CreateLaptop(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, expectedID, res.Id)

	// check that the laptop is saved to the store
	laptopCopy, err := laptopStore.Find(res.Id)
	require.NoError(t, err)
	require.NotNil(t, laptopCopy)

	// check that the saved laptop is the same as the one we send
	// Note: if we just use require.Equal() function to compare these 2 objects,
	// the test will fail.
	requireSameLaptop(t, laptop, laptopCopy)

}

func TestClientSearchLaptop(t *testing.T) {
	t.Parallel()

	// First I will create a search filter and an
	filter := &pb.Filter{
		MaxPriceUsd: 2000,
		MinCpuCores: 4,
		MinCpuGhz:   2.2,
		MinRam:      &pb.Memory{Value: 8, Unit: pb.Memory_GIGABYTE},
	}

	// in-memory laptop store to insert some laptops for searching.
	store := service.NewInMemoryLaptopStore()

	// make an expectedIDs map that will contain all laptop IDs that we expect
	// to be found by the server.
	expectedIDs := make(map[string]bool)

	/*
			We use a for loop to create 6 laptops:

		    Case 0: unmatched laptop with a too high price.
		    Case 1: unmatched because it has only 2 cores.
		    Case 2: doesn’t match because the min frequency is too low.
		    Case 3: doesn’t match since it has only 4 GB of RAM.
		    Case 4 + 5: matched.

	*/

	for i := 0; i < 6; i++ {
		laptop := sampledata.NewLaptop()

		switch i {
		case 0:
			laptop.PriceUsd = 2500
		case 1:
			laptop.Cpu.NumberCores = 2
		case 2:
			laptop.Cpu.MinGhz = 2.0
		case 3:
			laptop.Ram = &pb.Memory{Value: 4096, Unit: pb.Memory_MEGABYTE}
		case 4:
			laptop.PriceUsd = 1999
			laptop.Cpu.NumberCores = 4
			laptop.Cpu.MinGhz = 2.5
			laptop.Cpu.MaxGhz = laptop.Cpu.MinGhz + 2.0
			laptop.Ram = &pb.Memory{Value: 16, Unit: pb.Memory_GIGABYTE}
			expectedIDs[laptop.Id] = true
		case 5:
			laptop.PriceUsd = 2000
			laptop.Cpu.NumberCores = 6
			laptop.Cpu.MinGhz = 2.8
			laptop.Cpu.MaxGhz = laptop.Cpu.MinGhz + 2.0
			laptop.Ram = &pb.Memory{Value: 64, Unit: pb.Memory_GIGABYTE}
			expectedIDs[laptop.Id] = true
		}

		// call store.Save() to save the laptop to the store, and require that
		// there’s no error returned.
		err := store.Save(laptop)
		require.NoError(t, err)

		// Next we have to add this store to the test laptop server.
	}

	// call startTestLaptopServe function to start the test server
	serverAddress := startTestLaptopServer(t, store, nil, nil)

	// create a laptop client object with that server address:
	laptopClient := newTestLaptopClient(t, serverAddress)

	// create a new SearchLaptopRequest with the filter
	req := &pb.SearchLaptopRequest{Filter: filter}

	// call laptopCient.SearchLaptop() with the created request to get back the stream.
	// There should be no errors returned.
	stream, err := laptopClient.SearchLaptop(context.Background(), req)
	require.NoError(t, err)

	// use the found variable to keep track of the number of laptops found.
	found := 0

	// use a for loop to receive multiple responses from the stream.
	for {
		res, err := stream.Recv()
		// If we got an end-of-file error, then break.
		if err == io.EOF {
			break
		}

		// Else we check that there’s no error
		require.NoError(t, err)
		// laptop ID should be in the expectedIDs map
		require.Contains(t, expectedIDs, res.GetLaptop().GetId())

		// increase the number of laptops found
		found += 1
	}

	// Finally we require that number to equal to the size of the expectedIDs.
	require.Equal(t, len(expectedIDs), found)
}

// requireSameLaptop serialises the objects to JSON, and compares the 2 output JSON strings
func requireSameLaptop(t *testing.T, laptop1 *pb.Laptop, laptop2 *pb.Laptop) {
	json1, err := serializer.ProtobufToJSON(laptop1)
	require.NoError(t, err)

	json2, err := serializer.ProtobufToJSON(laptop2)
	require.NoError(t, err)

	require.Equal(t, json1, json2)
}

// startTestLaptopServer func to start the gRPC server.
// It will take a testing.T as an argument, and return the
// network address string of the server.
func startTestLaptopServer(t *testing.T, laptopStore service.LaptopStore, imageStore service.ImageStore, ratingStore service.RatingStore) string {
	// create a new laptop server with an in-memory laptop store.
	laptopServer := service.NewLaptopServer(laptopStore, imageStore, ratingStore)

	// create the gRPC server by calling grpc.NewServer() function.
	grpcServer := grpc.NewServer()

	// register the laptop service server on that gRPC server.
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)

	// create a new listener that will listen to tcp connection.
	// The number 0 here means that we want it to be assigned any random available port.
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	// call grpcServer.Serve() to start listening to the request.
	// This is a blocking call, so we have to run it in a separate go-routine,
	// since we want to send requests to this server after that.
	go grpcServer.Serve(listener)

	// Finally we just return the the address string of the listener.
	return listener.Addr().String()

}

// newTestLaptopClient returns a new laptop-client.
// Takes the testing.T object, and the server address as its arguments, then return a
// pb.LaptopServiceClient.
func newTestLaptopClient(t *testing.T, serverAddress string) pb.LaptopServiceClient {
	conn, err := grpc.Dial(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	// return a new laptop service client with the created connection.
	return pb.NewLaptopServiceClient(conn)

}

// Test client upload image
func TestClientUploadImage(t *testing.T) {
	t.Parallel()

	// use tmp as the folder to store images
	testImageFolder := "../tmp"

	// create a new in-memory laptop store
	laptopStore := service.NewInMemoryLaptopStore()

	// create a new disk image store with the tmp image folder.
	imageStore := service.NewDiskImageStore(testImageFolder)

	// generate a sample laptop, and save it to the laptop store.
	laptop := sampledata.NewLaptop()
	err := laptopStore.Save(laptop)
	require.NoError(t, err)

	// start the test server and make a new laptop client.
	serverAddress := startTestLaptopServer(t, laptopStore, imageStore, nil)
	laptopClient := newTestLaptopClient(t, serverAddress)

	imagePath := fmt.Sprintf("%s/laptop.jpg", testImageFolder)
	// open the file
	file, err := os.Open(imagePath)
	// check that there’s no error
	require.NoError(t, err)
	// defer closing it.
	defer file.Close()

	// call laptopClient.UploadImage() to get the stream and require that no error should occur.
	stream, err := laptopClient.UploadImage(context.Background())
	require.NoError(t, err)

	// send the first request that contains only the metadata of the laptop image.
	imageType := filepath.Ext(imagePath)
	req := &pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_Info{
			Info: &pb.ImageInfo{
				LaptopId:  laptop.GetId(),
				ImageType: imageType,
			},
		},
	}

	err = stream.Send(req)
	require.NoError(t, err)

	//  use a for loop to send the content of the image files in chunks:

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)
	// size keeps track of the total image size,
	size := 0

	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}

		require.NoError(t, err)
		size += n

		req := &pb.UploadImageRequest{
			Data: &pb.UploadImageRequest_ChunkData{
				ChunkData: buffer[:n],
			},
		}

		err = stream.Send(req)
		require.NoError(t, err)
	}

	// call stream.CloseAndRecv() to get the response from the server,
	// and we that, the returned ID should not be a zero-value,
	// and that the value of the returned image size should equal to size.

	res, err := stream.CloseAndRecv()
	require.NoError(t, err)
	require.NotZero(t, res.GetId())
	require.EqualValues(t, size, res.GetSize())

	// check that the image is saved to the correct folder on the server.
	// It should be inside the test image folder, with file name is the image ID
	// and file extension is the image type.
	savedImagePath := fmt.Sprintf("%s/%s%s", testImageFolder, res.GetId(), imageType)
	require.FileExists(t, savedImagePath)
	// remove the file at the end of the test.
	require.NoError(t, os.Remove(savedImagePath))

}

func TestClientRateLaptop(t *testing.T) {
	t.Parallel()

	// create a new laptop store
	laptopStore := service.NewInMemoryLaptopStore()
	//create a new rating store
	ratingStore := service.NewInMemoryRatingStore()

	// generate a new random laptop
	laptop := sampledata.NewLaptop()

	// save it to the store.
	err := laptopStore.Save(laptop)
	require.NoError(t, err)

	// start the test laptop server to get the server adress, and use it
	// to create a test laptop client.
	serverAddress := startTestLaptopServer(t, laptopStore, nil, ratingStore)
	laptopClient := newTestLaptopClient(t, serverAddress)

	// call laptopClient.RateLaptop() with a background context to get the stream,
	// and require no error
	stream, err := laptopClient.RateLaptop(context.Background())
	require.NoError(t, err)

	/*
		For simplicity, we just rate 1 single laptop, but we will rate it 3 times with a
		score of 8, 7.5 and 10 respectively. So the expected average score after each time
		should be 8, 7.75 and 8.5.
	*/
	scores := []float64{8, 7.5, 10}
	averages := []float64{8, 7.75, 8.5}

	// number of rated times
	n := len(scores)
	// use a for loop to send multiple requests.
	for i := 0; i < n; i++ {
		// Each time we will create a new request with the same laptop ID and a new score.
		req := &pb.RateLaptopRequest{
			LaptopId: laptop.GetId(),
			Score:    scores[i],
		}

		// We call stream.Send() to send the request to the server,
		// and require no errors to be returned.
		err := stream.Send(req)
		require.NoError(t, err)
	}

	// After sending all the rate laptop requests, we call stream.CloseSend()
	// and require no errors to be returned.
	err = stream.CloseSend()
	require.NoError(t, err)

	// Note: You can use a separate goroutine to receive the responses
	// Or use a for loop
	// use an idx variable to count how many responses we have received.
	for idx := 0; ; idx++ {
		// call stream.Recv() to receive a new response.
		res, err := stream.Recv()
		//  If error is EOF, then it’s the end of the stream
		if err == io.EOF {
			// we just require that the number of responses we received must be equal to n, 
			// which is the number of requests we sent, and we return immediately.
			require.Equal(t, n, idx)
			return
		}

		// Else, there should be no error.
		require.NoError(t, err)
		// The response laptop ID should be equal to the input laptop ID. 
		require.Equal(t, laptop.GetId(), res.GetLaptopId())
		// The rated count should be equal to idx + 1. 
		require.Equal(t, uint32(idx+1), res.GetRatedCount())
		// And the average score should be equal to the expected value.
		require.Equal(t, averages[idx], res.GetAverageScore())
	}

}
