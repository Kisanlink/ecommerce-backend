package responses

import (
	"fmt"
	"io"
	"net/http"
)

// Example function to demonstrate Delhivery API call
// This is for reference only - actual implementation should use the service layer
func ExampleDelhiveryAPICall() {
	url := "https://track.delhivery.com/api/dc/fetch/serviceability/pincode?product_type=Heavy&pincode=400086"

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("Authorization", "Token 5c2243c9d55f103ed21341076ccbf38700dabf5d")
	req.Header.Add("Accept", "application/json")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	fmt.Println(string(body))
}
