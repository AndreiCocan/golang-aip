package sqlite_test

import (
	"fmt"

	"github.com/AndreiCocan/golang-aip/filtering"
	"github.com/AndreiCocan/golang-aip/filtering/sqlite"
)

func ExampleTranspile() {
	schema := filtering.NewSchema(
		filtering.String("display_name"),
		filtering.Bool("published"),
		filtering.Message("author", filtering.String("name")),
	)

	checked, err := filtering.Compile(
		`display_name = "war*" AND published = true AND author.name = "Tolstoy"`,
		schema,
	)
	if err != nil {
		fmt.Println(err)

		return
	}

	frag, args, err := sqlite.Transpile(checked,
		sqlite.Column("author.name", "author_name"),
	)
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println(frag)
	fmt.Println(args)
	// Output:
	// ("display_name" GLOB ? AND "published" = ? AND "author_name" = ?)
	// [war* 1 Tolstoy]
}

func ExampleSearchColumns() {
	schema := filtering.NewSchema(
		filtering.String("display_name"),
	)

	checked, err := filtering.Compile("Tolstoy peace", schema)
	if err != nil {
		fmt.Println(err)

		return
	}

	frag, args, err := sqlite.Transpile(checked,
		sqlite.SearchColumns("display_name", "author_name"),
	)
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println(frag)
	fmt.Println(args)
	// Output:
	// (("display_name" LIKE ? ESCAPE '\' OR "author_name" LIKE ? ESCAPE '\') AND ("display_name" LIKE ? ESCAPE '\' OR "author_name" LIKE ? ESCAPE '\'))
	// [%Tolstoy% %Tolstoy% %peace% %peace%]
}
