package sqlite_test

import (
	"fmt"

	"github.com/AndreiCocan/golang-aip/ordering"
	"github.com/AndreiCocan/golang-aip/ordering/sqlite"
)

func ExampleTranspile() {
	schema := ordering.NewSchema("display_name", "create_time", "author.name")

	checked, err := ordering.Compile("create_time desc, author.name, display_name", schema)
	if err != nil {
		fmt.Println(err)

		return
	}

	frag, err := sqlite.Transpile(checked,
		sqlite.Column("author.name", "author_name"),
	)
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println(frag)
	// Output:
	// "create_time" DESC, "author_name", "display_name"
}
