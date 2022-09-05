package service

import (
	"bytes"
	"fmt"
	"os"
	"sync"

	"github.com/google/uuid"
)

// ImageStore saves the uploaded image file somewhere on the server or on the cloud
type ImageStore interface {
	Save(laptopID string, imageType string, imageData bytes.Buffer) (string, error)
}

// ImageInfo contains an extra field Path since we 
// cannot request an image path from the client
type ImageInfo struct {
	LaptopID string
	Type     string
	Path     string
}

// DiskImageStore, implements the ImageStore interface.
// It saves image files to the disk, and store its information in memory.
type DiskImageStore struct {
	mutex       sync.RWMutex // mutex to handle concurrency
	imageFolder string       // path of the folder to save laptop images
	// map with the key is image ID and the value is some information of the image.
	images map[string]*ImageInfo
}

// NewDiskImageStore returns a new instance of DiskImageStore
func NewDiskImageStore(imageFolder string) *DiskImageStore {
	return &DiskImageStore{
		imageFolder: imageFolder,
		images:      make(map[string]*ImageInfo),
	}
}

// implement the Save() function, which is required by the ImageStore interface.
func (store *DiskImageStore) Save(laptopID string, imageType string, imageData bytes.Buffer) (string, error) {

	// generate a new random UUID for the image
	imageID, err := uuid.NewRandom()

	if err != nil {
		return "", fmt.Errorf("cannot generate image id: %w", err)
	}

	// make the path to store the image by joining the image folder, image ID, and image type.
	imagePath := fmt.Sprintf("%s/%s%s", store.imageFolder, imageID, imageType)

	// call os.Create() to create the file
	file, err := os.Create(imagePath)

	if err != nil {
		return "", fmt.Errorf("cannot create image file: %w", err)
	}

	// call imageData.WriteTo() to write the image data to the created file
	_, err = imageData.WriteTo(file)

	if err != nil {
		return "", fmt.Errorf("cannot write image to file: %w", err)
	}

	// If the file is written successfully, we need to save its information to
	// the in-memory map. So we have to acquire the write lock of the store.
	store.mutex.Lock()
	defer store.mutex.Unlock()

	// save the image information to the map with key is the ID of the image
	store.images[imageID.String()] = &ImageInfo{
		LaptopID: laptopID,
		Type:     imageType,
		Path:     imagePath,
	}

	// Finally we return the image ID to the caller
	return imageID.String(), nil
}
