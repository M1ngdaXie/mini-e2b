package main

import (
	"log"
	"net/http"

	"github.com/M1ngdaXie/mini-e2b/internal/proxy"
)

func main() {
	// 1. 假的 smap
	fakeVMs := map[string]string{
		"abc": "127.0.0.1",
	}

	// 2. lookup：匿名函数，捕获上面的 fakeVMs
	lookup := func(id string) (string, bool) {
		id, ok := fakeVMs[id]
		if !ok {
			return "", false
		}
		return id, true
	}

	// 3. 建 proxy
	p, err := proxy.NewProxy(lookup) // 如果你还保留了 error 返回值，就照旧处理 err
	if err != nil {
		log.Println("error starting proxy")
		return
	}
	// 4. 建 mux，把 proxy 挂上去
	mux := http.NewServeMux()
	mux.Handle("/vms/{id}/proxy/{port}/{path...}", p)
	// 5. 启动，交给服务器的是 mux，不是 p
	log.Fatal(http.ListenAndServe(":9000", mux))
}
