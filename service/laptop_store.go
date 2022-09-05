package service

import (
	"context"
	"errors"
	"fmt"
	pb "gRPC-Playground/ecommerce"
	"log"
	"sync"

	"github.com/jinzhu/copier"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrAlreadyExists is returned when a record with the same ID already exists in the store
var ErrAlreadyExists = errors.New("record already exists")

// LaptopStore is an interface to store laptop
// As we might have different types of store, define
// LaptopStore as an interface.
// It has a Save() function to save a laptop to the store.
// Note: we can always implement another DBLaptopStore to save
// laptop to a database
type LaptopStore interface {
	// Save saves the laptop to the store
	Save(laptop *pb.Laptop) error

	// Find finds a laptop by ID
	Find(id string) (*pb.Laptop, error)

	// Search() function takes a filter as input, and also a callback function to
	// report whenever a laptop is found.
	// The context is used to control the deadline/timeout of the request.
	Search(ctx context.Context, filter *pb.Filter, found func(laptop *pb.Laptop) error) error
}

// InMemoryLaptopStore to implement this interface
// InMemoryLaptopStore stores laptop in memory
type InMemoryLaptopStore struct {
	// Use the read-write mutex to handle the multiple concurrent
	// requests to save laptops.
	mutex sync.RWMutex
	// key is the laptop ID, and the value is the laptop object.
	data map[string]*pb.Laptop
}

// RatingStore interface saves the laptop ratings.
type RatingStore interface{
	Add(laptopID string, score float64) (*Rating, error)
} 

// Rating struct
type Rating struct {
	// No of times a laptop is rated
	Count uint32
	// sum of rated scores
    Sum   float64

}

// InMemoryRatingStore implements the RatingStore interface
type InMemoryRatingStore struct {
	// mutex to handle concurrent access.
	mutex sync.RWMutex
	// rating map with key is the laptop ID, and value is the rating object.
	rating map[string]*Rating
}

// NewInMemoryLaptopStore returns a new InMemoryLaptopStore
func NewInMemoryLaptopStore() *InMemoryLaptopStore {
	return &InMemoryLaptopStore{
		data: make(map[string]*pb.Laptop),
	}
}

// NewInMemoryRatingStore() returns a new InMemoryRatingStore
func NewInMemoryRatingStore() *InMemoryRatingStore {
	return &InMemoryRatingStore{
		rating: make(map[string]*Rating),
	}
}

// implement the Save laptop function as required by the interface.
// Save saves the laptop to the store
func (store *InMemoryLaptopStore) Save(laptop *pb.Laptop) error {
	// First we need to acquire a write lock before adding new objects
	store.mutex.Lock()

	// defer the unlock command.
	defer store.mutex.Unlock()

	// check if the laptop ID already exists in the map or not.
	// If it does, just return an error to the caller.
	if store.data[laptop.Id] != nil {
		log.Println(ErrAlreadyExists)

	}

	// If the laptop doesn't exist, we can save it to the store.
	// However, to be safe, we should do a deep-copy of the laptop object.
	laptopCopy, err := deepCopy(laptop)
	if err != nil {
		log.Fatal("failed to do a deepCopy: ", err)
	}

	store.data[laptopCopy.Id] = laptopCopy

	return nil

}

// Find finds a laptop by ID
func (store *InMemoryLaptopStore) Find(id string) (*pb.Laptop, error) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	// get the laptop from the store.data map by its id.
	laptop, exist := store.data[id]
	
	if exist {
		
		return deepCopy(laptop)

	}

	return nil, status.Errorf(codes.NotFound, "Laptop with id:  %s not Found.", id)

}

func deepCopy(laptop *pb.Laptop) (*pb.Laptop, error) {
	laptopCopy := &pb.Laptop{}

	err := copier.Copy(laptopCopy, laptop)
	if err != nil {
		return nil, fmt.Errorf("cannot copy laptop data: %w", err)
	}

	return laptopCopy, nil
}

// Search searches for laptops with filter, returns one by one via the found function
func (store *InMemoryLaptopStore) Search(ctx context.Context, filter *pb.Filter,
	found func(laptop *pb.Laptop) error,
) error {
	// acquire a read lock, and unlock it afterward.
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	// iterate through all laptops in the store, and check which one is qualified to the filter.
	for _, laptop := range store.data {
		// before checking if a laptop is qualified or not, we check if the context error is
		// Cancelled or DeadlineExceeded or not.
		if ctx.Err() == context.Canceled || ctx.Err() == context.DeadlineExceeded {
			log.Print("context is cancelled")
			// If it is, we should return immediately because the request is either
			// already timed out or cancelled by client
			return nil
		}

		// time.Sleep(time.Second)
		log.Println("searrching laptop id: ", laptop.GetId())

		if isQualified(filter, laptop) {
			// When the laptop is qualified, we have to deep-copy it before sending it
			// to the caller via the callback function found()
			laptopCopy, err := deepCopy(laptop)
			if err != nil {

				return err
			}

			err = found(laptopCopy)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Implement the Add method
func (store *InMemoryRatingStore) Add(laptopID string, score float64) (*Rating, error) {
	// Acquire write lock
	store.mutex.Lock()
	defer store.mutex.Unlock()

	// get the rating of the laptop ID from the map. 
	rating := store.rating[laptopID]

	// If the rating is not found, we just create a new object with count is 1 
	// and sum is the input score. Else, we increase the rating count by 1 
	// and add the score to the sum.
	if rating == nil {
		rating = &Rating{
			Count: 1,
			Sum: score,
		}
	} else {
		rating.Count++
		rating.Sum += score
	}

	// put the updated rating back to the map 
	store.rating[laptopID] = rating

	// and return it to the caller. 
	return rating, nil
	
}

// isQualified() function takes a filter and a laptop as input, and returns true if the
// laptop satisfies the filter.
func isQualified(filter *pb.Filter, laptop *pb.Laptop) bool {
	if laptop.GetPriceUsd() > filter.GetMaxPriceUsd() {
		return false
	}

	if laptop.GetCpu().GetNumberCores() < filter.GetMinCpuCores() {
		return false
	}

	if laptop.GetCpu().GetMinGhz() < filter.GetMinCpuGhz() {
		return false
	}

	if toBit(laptop.GetRam()) < toBit(filter.GetMinRam()) {
		return false
	}

	return true
}

func toBit(memory *pb.Memory) uint64 {
	value := memory.GetValue()

	switch memory.GetUnit() {
	case pb.Memory_BIT:
		return value
	case pb.Memory_BYTE:
		return value << 3 // 8 = 2^3
	case pb.Memory_KILOBYTE:
		return value << 13 // 1024 * 8 = 2^10 * 2^3 = 2^13
	case pb.Memory_MEGABYTE:
		return value << 23
	case pb.Memory_GIGABYTE:
		return value << 33
	case pb.Memory_TERABYTE:
		return value << 43
	default:
		return 0
	}
}
