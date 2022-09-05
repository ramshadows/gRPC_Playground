package service_test

import (
	"context"
	pb "gRPC-Playground/ecommerce"
	sampledata "gRPC-Playground/sample-data"
	"gRPC-Playground/service"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Note: This tests does not use any kind of network call yet.
// They're basically just a direct call on server side.
func TestCreateLaptopServer(t *testing.T) {
	t.Parallel()

	laptopNoID := sampledata.NewLaptop()
	laptopNoID.Id = ""

	laptopInvalidID := sampledata.NewLaptop()
	laptopInvalidID.Id = "invalid-uuid"

	laptopDuplicateID := sampledata.NewLaptop()
	storeDuplicateID := service.NewInMemoryLaptopStore()
	//err := storeDuplicateID.Save(laptopDuplicateID)
	//require.Nil(t, err)

	testCases := []struct {
		name   string
		laptop *pb.Laptop
		store  service.LaptopStore
		code   codes.Code
	}{
		{
			name:   "success_with_id",
			laptop: sampledata.NewLaptop(),
			store:  service.NewInMemoryLaptopStore(),
			code:   codes.OK,
		},
		{
			name:   "success_no_id",
			laptop: laptopNoID,
			store:  service.NewInMemoryLaptopStore(),
			code:   codes.OK,
		},
		{
			name:   "failure_invalid_id",
			laptop: laptopInvalidID,
			store:  service.NewInMemoryLaptopStore(),
			code:   codes.InvalidArgument,
		},
		{
			name:   "failure_duplicate_id",
			laptop: laptopDuplicateID,
			store:  storeDuplicateID,
			code:   codes.AlreadyExists,
		},
	}

	for i := range testCases {
		// save the current test case to a local variable.
		// This is very important to avoid concurrency issues,
		// because we want to create multiple parallel subtests.
		tc := testCases[i]

		// call t.Run() to create a subset and use tc.name for the name of the subtest.
		t.Run(tc.name, func(t *testing.T) {
			// call t.Parallel() to make it run in parallel with other tests.
			t.Parallel()

			// build a new CreateLaptopRequest object with the input tc.laptop.
			req := &pb.CreateLaptopRequest{
				Laptop: tc.laptop,
			}

			testImageFolder := "../tmp"

			imageStore := service.NewDiskImageStore(testImageFolder)
			ratingStore := service.NewInMemoryRatingStore()

			// create a new LaptopServer with the in-memory laptop store.
			server := service.NewLaptopServer(tc.store, imageStore, ratingStore)

			// call server.CreateLaptop() function with a background context and the request object.
			res, err := server.CreateLaptop(context.Background(), req)

			// Now there are 2 cases:
			// The successful case, i.e when tc.code is OK.
			// In this case, we should check there's no error. The response should be not nil.
			// The returned ID should be not empty. And if the input laptop already has ID, then
			// the returned ID should equal to it.
			if tc.code == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.NotEmpty(t, res.Id)
				if len(tc.laptop.Id) > 0 {
					require.Equal(t, tc.laptop.Id, res.Id)
				}
			} else {
				// The failure case, when tc.code is not OK. We check there should be an
				// error and the response should be nil.
				require.Error(t, err)
				require.Nil(t, res)
				// To check the status code, we call status.FromError() to get the status object.
				st, ok := status.FromError(err)
				// Check that ok should be true and st.Code() should equal to tc.code.
				require.True(t, ok)
				require.Equal(t, tc.code, st.Code())
			}

		})
	}

}
