package sampledata

import (
	pb "gRPC-Playground/ecommerce"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// This is a special function that will be called exactly once before
// any other code in the package is executed
func init() {
	// tell rand to use the current unix nano as the seed value.
	rand.Seed(time.Now().UnixNano())
}

func randomBool() bool {
	// give us a random integer, which is either 0 or 1
	// since n=2. So we just return true if the value is 1.
	return rand.Intn(2) == 1

}

func randomKeyboardLayout() pb.Keyboard_Layout {
	switch rand.Intn(3) {
	// If the value is 1 then return QWERTY
	case 1:
		return pb.Keyboard_QWERTY

		// If the value is 2, then return QWERTZ
	case 2:
		return pb.Keyboard_QWERTZ

		// Otherwise, return AZERTY
	default:
		return pb.Keyboard_AZERTY

	}
}

// randomCPUBrand returns a select a random value from a predefined set of brands
func randomCPUBrand() string {
	return randomStringFromSet("Intel", "AMD")
}

// randomCPUName returns a random CPU name based on the brand
// Since we know there are only 2 brands, a simple if is enough.
func randomCPUName(brand string) string {
	if brand == "Intel" {
		return randomStringFromSet(
			"Xeon E-2286M",
			"Core i9-9980HK",
			"Core i7-9750H",
			"Core i5-9400F",
			"Core i3-1005G1",
		)
	}

	// Otherwise
	return randomStringFromSet(
		"Ryzen 7 PRO 2700U",
		"Ryzen 5 PRO 3500U",
		"Ryzen 3 PRO 3200GE",
	)
}

// randomStringFromSet takes a set of variable number of strings as input,
// and return 1 random string from that set.
func randomStringFromSet(brands ...string) string {
	n := len(brands)

	if n == 0 {
		return ""
	}

	return brands[rand.Intn(n)]
}

// randomInt function uses the rand.Int() to generate from
// zero to (max-min). If we add min to it, we will get a value from min to max
// randomInt() function can be used to set the number of cores and the number of threads
// Cores would be between 2 cores and 8 cores
// The number of threads will be a random integer between the number of cores and 12
func randomInt(min, max int) int {
	return min + rand.Int()%(max-min+1)
}

// randomFloat64() uses the rand.Float64() func to generate a random float between 0 and 1
// We then multiply it with (max - min) to get a value between 0 and (max - min)
// We add min to this value, we will get a number from min to max
// randomFloat64 can be used to generate random cpu frequencies between 2.0 and 3.5
func randomFloat64(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func randomGPUBrand() string {
	return randomStringFromSet("Nvidia", "AMD")
}

func randomGPUName(brand string) string {
	if brand == "Nvidia" {
		return randomStringFromSet(
			"RTX 2060",
			"RTX 2070",
			"GTX 1660-Ti",
			"GTX 1070",
		)
	}

	return randomStringFromSet(
		"RX 590",
		"RX 580",
		"RX 5700-XT",
		"RX Vega-56",
	)
}

// randomFloat32() uses the rand.Float32() func to generate a random float between 0 and 1
// We then multiply it with (max - min) to get a value between 0 and (max - min)
// We add min to this value, we will get a number from min to max
// randomFloat32 can be used to generate random screen height between 13 and 17
func randomFloat32(min, max float32) float32 {
	return min + rand.Float32()*(max-min)
}

// randomScreenResolution() returns a new screen Resolution
func randomScreenResolution() *pb.Screen_Resolution {
	height := randomInt(1080, 4320)
	width := height * 16 / 9

	resolution := &pb.Screen_Resolution{
		Width:  uint32(width),
		Height: uint32(height),
	}
	return resolution
}

// Then the screen panel. There are only 2 types of panel: either IPS or OLED.
// So we just use rand.Intn(2) here, and a simple if would do the job.
func randomScreenPanel() pb.Screen_Panel {
	if rand.Intn(2) == 1 {
		return pb.Screen_IPS
	}
	return pb.Screen_OLED
}

// randomID() uses google UUID to generate a random id
func randomID() string {
	return uuid.New().String()
}

func randomLaptopBrand() string {
    return randomStringFromSet("Apple", "Dell", "Lenovo")
}

func randomLaptopName(brand string) string {
    switch brand {
    case "Apple":
        return randomStringFromSet("Macbook Air", "Macbook Pro")
    case "Dell":
        return randomStringFromSet("Latitude", "Vostro", "XPS", "Alienware")
    default:
        return randomStringFromSet("Thinkpad X1", "Thinkpad P1", "Thinkpad P53")
    }
}

// RandomLaptopScore generates a random laptop score between 1 and 10.
func RandomLaptopScore() float64 {
    return float64(randomInt(1, 10))
}


