package main

type Order struct {
	Id          string
	Items       []string
	Price       float32
	Destination string
}

var orderMap = make(map[string]*Order)

func initSampleData() {

	orderMap["102"] = &Order{Id: "102", Items: []string{"Google Pixel 3A", "Mac Book Pro"}, Destination: "Mountain View, CA", Price: 1800.00}
	orderMap["103"] = &Order{Id: "103", Items: []string{"Apple Watch S4"}, Destination: "San Jose, CA", Price: 400.00}
	orderMap["104"] = &Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub"}, Destination: "Mountain View, CA", Price: 400.00}
	orderMap["105"] = &Order{Id: "105", Items: []string{"Amazon Echo"}, Destination: "San Jose, CA", Price: 30.00}
	orderMap["106"] = &Order{Id: "106", Items: []string{"Amazon Echo", "Apple iPhone XS"}, Destination: "Mountain View, CA", Price: 300.00}
}

func main() {
	initSampleData()
/*
	myOrder := []struct {
		Id          string
		Items       []string
		Price       float32
		Destination string
	}{
		{
			Id:          orderMap["102"].Id,
			Items:       orderMap["102"].Items,
			Price:       orderMap["102"].Price,
			Destination: orderMap["102"].Destination,
		},

		{
			Id:          orderMap["103"].Id,
			Items:       orderMap["103"].Items,
			Price:       orderMap["103"].Price,
			Destination: orderMap["103"].Destination,
		},

		{
			Id:          orderMap["104"].Id,
			Items:       orderMap["104"].Items,
			Price:       orderMap["104"].Price,
			Destination: orderMap["104"].Destination,
		},

		{
			Id:          orderMap["105"].Id,
			Items:       orderMap["105"].Items,
			Price:       orderMap["105"].Price,
			Destination: orderMap["105"].Destination,
		},

		{
			Id:          orderMap["106"].Id,
			Items:       orderMap["106"].Items,
			Price:       orderMap["106"].Price,
			Destination: orderMap["106"].Destination,
		},
	}

	/*
		fmt.Printf("Order id: %s\n", myOrder.Id)
		fmt.Printf("Order Items: %v\n", myOrder.Items[:])
		fmt.Printf("Order Price: %.2f\n", myOrder.Price)
		fmt.Printf("Order Destination: %s\n", myOrder.Destination)
	*/

}
