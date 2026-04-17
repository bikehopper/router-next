package debug

import (
	"fmt"
	"net/http"
)

func StartServer(port string) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, you've requested: %s\n", r.URL.Path)
	})

	fmt.Println("Server running on port", port)
	http.ListenAndServe(":"+port, nil)
}
