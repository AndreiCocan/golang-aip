package resourceid_test

import (
	"fmt"

	"github.com/AndreiCocan/golang-aip/resourceid"
)

func ExampleValidateUserSettable() {
	fmt.Println(resourceid.ValidateUserSettable("les-miserables"))
	fmt.Println(resourceid.ValidateUserSettable("Les-Miserables"))
	// Output:
	// <nil>
	// invalid resource ID: must begin with a lowercase letter
}

func ExampleNewSystemGenerated() {
	id := resourceid.NewSystemGenerated()
	fmt.Println(len(id))
	// Output:
	// 36
}
