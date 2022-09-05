package serializer

import (
	sampledata "gRPC-Playground/sample-data"
	"testing"

	pb "gRPC-Playground/ecommerce"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestFileSerializer(t *testing.T) {
	// call t.Parallel() so that all the unit tests can be run in parallel,
	// and any racing condition can be easily detected.
	t.Parallel()

	binaryFile := "../tmp/laptop.bin"
	jsonFile := "../tmp/laptop.json"

	// use the NewLaptop() function to make a new laptop1
	laptop1 := sampledata.NewLaptop()

	// call the WriteProtobufToBinaryFile() function to save it to the laptop.bin file.
	// in the tmp folder
	err := WriteProtobufToBinaryFile(laptop1, binaryFile)
	require.NoError(t, err)

	err = WriteProtobufToJSONFile(laptop1, jsonFile)
	require.NoError(t, err)

	// define a new laptop2 object
	laptop2 := &pb.Laptop{}

	// call ReadProtobufFromBinaryFile() to read the file data into that object.
	err = ReadProtobufFromBinaryFile(binaryFile, laptop2)

	// check that there's no errors.
	require.NoError(t, err)

	// check that laptop2 contains the same data as laptop1 by calling the
	// proto.Equal function provided by the golang/protobuf package
	// This function must return true, so we use require.True()
	require.True(t, proto.Equal(laptop1, laptop2))

}
