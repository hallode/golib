package json_test

import (
	"fmt"

	"github.com/hallode/golib/v2/json"
)

// Call json.Init("sonic") once at startup to switch the backend to
// bytedance/sonic; the API below is unchanged.
func ExampleMarshal() {
	type user struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	b, _ := json.Marshal(user{Name: "Ada", Age: 36})
	fmt.Println(string(b))

	var u user
	_ = json.Unmarshal(b, &u)
	fmt.Println(u.Name, u.Age)
	// Output:
	// {"name":"Ada","age":36}
	// Ada 36
}
