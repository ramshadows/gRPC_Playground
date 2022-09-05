package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	pb "gRPC-Playground/ecommerce"
	"io"
	"log"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxImageSize = 1 << 20

// // LaptopServer is the server that provides laptop services
type LaptopServer struct {
	pb.UnimplementedLaptopServiceServer
	laptopStore LaptopStore
	imageStore  ImageStore
	ratingStore RatingStore
}

// NewLaptopServer returns a new LaptopServer
func NewLaptopServer(laptopStore LaptopStore, imageStore ImageStore, ratingStore RatingStore) *LaptopServer {
	return &LaptopServer{
		laptopStore: laptopStore,
		imageStore:  imageStore,
		ratingStore: ratingStore,
	}
}

// CreateLaptop is a unary RPC to create a new laptop
// It implement the CreateLaptop function, which is required by the
// LaptopServiceServer interface.
// It takes a context and a CreateLaptopRequest object as input,
// and returns a CreateLaptopResponse or an error.
func (server *LaptopServer) CreateLaptop(ctx context.Context, req *pb.CreateLaptopRequest) (*pb.CreateLaptopResponse, error) {
	// First we call GetLaptop function to get the laptop object from the request.
	laptop := req.GetLaptop()

	log.Printf("received a create-laptop request with id: %s", laptop.Id)

	// If the client has already generated the laptop ID, we must check if
	// it is a valid UUID or not.
	if len(laptop.Id) > 0 {
		// check if it is a valid id
		_, err := uuid.Parse(laptop.Id)

		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "laptop ID is not a valid UUID: %v", err)
		}
	} else {
		// Generate a new uuid for the laptop id
		id, err := uuid.NewRandom()

		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot generate a new laptop ID: %v", err)
		}

		// set laptop id to the generated id
		laptop.Id = id.String()

	}

	//  check if the request is timeout or cancelled by the client or not,
	// because if it is then there's no reason to continue processing the request.

	if ctx.Err() == context.Canceled {
		log.Print("request is canceled")
		return nil, status.Error(codes.Canceled, "request is canceled")
	}

	if ctx.Err() == context.DeadlineExceeded {
		log.Print("deadline is exceeded")
		return nil, status.Error(codes.DeadlineExceeded, "deadline is exceeded")

	}

	// call server.Store.Save() to save the input laptop to the store
	err := server.laptopStore.Save(laptop)

	// If there's an error, return codes.Internal with the error to the client.
	if err != nil {
		code := codes.Internal

		// make it clearer to the client to handle, by checking if the error is
		// already-exists-record or not by call to call errors.Is() function.
		// If it's true, we return AlreadyExists status code instead of Internal.
		if errors.Is(err, ErrAlreadyExists) {
			code = codes.AlreadyExists
		}
		return nil, status.Errorf(code, "cannot save laptop to the store: %v", err)
	}

	log.Printf("saved laptop with id: %s", laptop.Id)

	// Finally, if no errors occur, we can create a new response object with the laptop ID
	// and return it to the caller.
	res := &pb.CreateLaptopResponse{
		Id: laptop.Id,
	}

	return res, nil

}

// SearchLaptop is a server-streaming RPC to search for laptops
func (server *LaptopServer) SearchLaptop(req *pb.SearchLaptopRequest,
	stream pb.LaptopService_SearchLaptopServer,
) error {
	// The first thing we do is to get the filter from the request.
	filter := req.GetFilter()
	log.Printf("receive a search-laptop request with filter: %v", filter)

	// Then we call server.Store.Search(), pass in the stream context, the filter,
	// and the callback function.
	err := server.laptopStore.Search(
		stream.Context(),
		filter,
		func(laptop *pb.Laptop) error {
			// create a new response object with that laptop and send it to the
			// client by calling stream.Send().
			res := &pb.SearchLaptopResponse{Laptop: laptop}
			err := stream.Send(res)
			// If an error occurs, we return it with the Internal status code,
			if err != nil {
				return err
			}

			log.Printf("sent laptop with id: %s", laptop.GetId())
			// else we return nil.
			return nil
		},
	)

	if err != nil {
		return status.Errorf(codes.Internal, "unexpected error: %v", err)
	}

	return nil
}

func (server *LaptopServer) GetLaptopByID(ctx context.Context, reqID *pb.GetLaptopByIDRequest) (*pb.GetLaptopByIDResponse, error) {
	// First we call GetId() function to get the laptop id from the request.
	laptopID := reqID.GetId()
	fmt.Println("Received a get laptop by id. ID: ", laptopID)

	// use the store.Find function to search for the given laptop id
	laptop, err := server.laptopStore.Find(laptopID)

	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Laptop with ID: %s not Ffound.", laptopID)

	}

	// create a new response object with that laptop
	res := &pb.GetLaptopByIDResponse{
		Laptop: laptop,
	}

	return res, status.New(codes.OK, "").Err()

}

func (server *LaptopServer) UploadImage(stream pb.LaptopService_UploadImageServer) error {
	// First we call stream.Recv() to receive the first request, which contains the metadata
	// information of the image
	req, err := stream.Recv()

	// If there’s an error, we write a log and return the status code Unknown to the client.
	if err != nil {
		return logError(status.Errorf(codes.Unknown, "cannot receive image info"))
	}

	// Next we can get the laptop ID and the image type from the request.
	laptopID := req.GetInfo().GetLaptopId()
	imageType := req.GetInfo().GetImageType()

	// write a log here saying that we have received the upload-image request with this
	// laptop ID and image type.
	log.Printf("receive an upload-image request for laptop %s with image type %s", laptopID, imageType)

	// Before saving the laptop image, we have to make sure that the laptop ID really exists.
	// So we call server.laptopStore.Find() to find the laptop by ID.
	laptop, err := server.laptopStore.Find(laptopID)

	// If we get an error, just log and return it with the Internal status code.
	if err != nil {
		return logError(status.Errorf(codes.Internal, "cannot find laptop: %v", err))
	}

	// Else, if the laptop is nil, which means it is not found, we log and return an
	// error status code NotFound to be more precise.
	if laptop == nil {
		return logError(status.Errorf(codes.NotFound, "laptop id %s doesn't exist", laptopID))
	}

	// Now if everything goes well and the laptop is found, we can start receiving the
	// image chunks data. So let’s create a new byte buffer to store them, and also a
	// variable to keep track of the total image size.
	imageData := bytes.Buffer{}
	imageSize := 0

	for {
		// checking the context error on server side before calling receive on the stream
		err := contextError(stream.Context())
		if err != nil {
			return err
		}

		log.Print("waiting to receive more data")

		// call stream.Recv() to get the request.
		req, err := stream.Recv()

		// check if the error is EOF or not. If it is, this means that no
		// more data will be sent, and we can safely break the loop.
		if err == io.EOF {
			log.Print("no more data")
			break
		}

		// Else, if the error is still not nil, we return it with Unknown
		// status code to the client.
		if err != nil {
			return logError(status.Errorf(codes.Unknown, "cannot receive chunk data: %v", err))
		}

		// Otherwise, if there’s no error, we can get the chunk data from the request.
		chunk := req.GetChunkData()

		// We get its size using the len() function.
		size := len(chunk)

		log.Printf("received a chunk with size: %d", size)

		// add this size to the total image size.
		imageSize += size

		// we don’t want the client to send too large image, so we check if the
		// image size is greater than the maximum size, let's say 1 MB as defined
		// by the constant maxImageSize (1 MB = 2^20 bytes = 1 << 20 bytes).
		if imageSize > maxImageSize {
			return logError(status.Errorf(
				codes.InvalidArgument,
				"image is too large: %d > %d", imageSize, maxImageSize,
			),
			)
		}

		// Else, we can append the chunk to the image data with the Write() function.
		_, err = imageData.Write(chunk)

		if err != nil {
			return logError(status.Errorf(codes.Internal, "cannot write chunk data: %v", err))
		}
	}

	//call imageStore.Save() to save the image data to the store and get back the image ID:
	imageID, err := server.imageStore.Save(laptopID, imageType, imageData)

	if err != nil {
		return logError(status.Errorf(codes.Internal, "cannot save image to the store: %v", err))
	}

	// If the image is saved successfully, we create a response object with the
	// image ID and image size.
	res := &pb.UploadImageResponse{
		Id:   laptopID,
		Size: uint32(imageSize),
	}

	// Then we call stream.SendAndClose() to send the response to client.
	err = stream.SendAndClose(res)

	if err != nil {
		return logError(status.Errorf(codes.Unknown, "cannot send response: %v", err))
	}

	// And finally we can write a log saying that the image is successfully saved
	// with this ID and size.
	log.Printf("saved image with id: %s, size: %d", imageID, imageSize)

	return nil

}

// RateLaptop is a bidirectional remote gRPC method
func (server *LaptopServer) RateLaptop(stream pb.LaptopService_RateLaptopServer) error {
	// Since we will receive multiple requests from the stream, we must use a for loop here.
	for {
		// But first, check the context error to see if it’s already canceled
		// or deadline exceeded or not.
		err := contextError(stream.Context())
		if err != nil {
			return err
		}

		// call stream.Recv() to get a request from the stream.
		req, err := stream.Recv()

		// If error is end of file (EOF), then there’s no more data, we simply break the loop.
		if err == io.EOF {
			log.Print("no more data")
			break
		}
		// Else if error is not nil, we log it and return the error with status
		// code unknown to the client.
		if err != nil {
			return logError(status.Errorf(codes.Unknown, "cannot receive stream request: %v", err))
		}

		// Otherwise, we can get the laptop ID and the score from the request.
		laptopID := req.GetLaptopId()
		score := req.GetScore()

		// write a log here saying that we have received a request with this laptop ID and score.
		log.Printf("received a rate-laptop request: id = %s, score = %.2f", laptopID, score)

		// check if this laptop ID really exists or not by using the laptopStore.Find() function.
		found, err := server.laptopStore.Find(laptopID)

		// If an error occurs, we return it with the status code Internal
		if err != nil {
			return logError(status.Errorf(codes.Internal, "cannot find laptop: %v", err))
		}

		// If the laptop is not found, we return the status code NotFound to the client.
		if found == nil {
			return logError(status.Errorf(codes.NotFound, "laptopID %s is not found", laptopID))
		}

		// If everything goes well, we call ratingStore.Add() to add the new laptop score 
		// to the store and get back the updated rating object.
		rating, err := server.ratingStore.Add(laptopID, score)
		// If there’s an error, we return Internal status code.
		if err != nil {
			return logError(status.Errorf(codes.Internal, "cannot add rating to the store: %v", err))
		}

		// Else, we create a RateLaptopResponse with laptop ID is the input laptop ID, 
		// rated count taken from the rating object, and average score is computed 
		// using the sum and count of the rating.
		res := &pb.RateLaptopResponse{
			LaptopId:     laptopID,
			RatedCount:   rating.Count,
			AverageScore: rating.Sum / float64(rating.Count),
		}

		// Then we call stream.Send() to send the response to the client.
		err = stream.Send(res)

		// If error is not nil, we log it and return status code Unknown.
		if err != nil {
			return logError(status.Errorf(codes.Unknown, "cannot send stream response: %v", err))
		}

	}

	return nil
}

func logError(err error) error {
	if err != nil {
		log.Print(err)
	}
	return err
}

//contexError is extracted from the RPC
func contextError(ctx context.Context) error {
	switch ctx.Err() {
	// In case the context error is Canceled, we log it and return the error with Canceled status code.
	case context.Canceled:
		return logError(status.Error(codes.Canceled, "request is canceled"))

	// In case DeadlineExceeded, we do the same thing, but with DeadlineExceeded status code.
	case context.DeadlineExceeded:
		return logError(status.Error(codes.DeadlineExceeded, "deadline is exceeded"))

	// And for default case, we just return nil.
	default:
		return nil
	}
}
