package main

import (
    "log"
	"net"
	"net/http"
	"time"

	"gosearch/pkg/api"
	"gosearch/pkg/crawler"
	"gosearch/pkg/crawler/spider"
	"gosearch/pkg/engine"
	"gosearch/pkg/index"
	"gosearch/pkg/index/hash"
	"gosearch/pkg/storage"
	"gosearch/pkg/storage/memstore"

	"github.com/gorilla/mux" // маршрутизатор HTTP-запросов
)


// Сервер Интернет-поисковика GoSearch.
type gosearch struct {
	api     *api.Service
	engine  *engine.Service
	scanner crawler.Interface
	index   index.Interface
	storage storage.Interface

	router *mux.Router

	sites []string
	depth int
	addr  string
}

func main() {
    // var nFlag = flag.String("s", "Seacrh string", "Enter search value")
    // flag.Parse()
	server := new()
	// server.init(*nFlag)
	server.init()
	server.run()
}

// new создаёт объект и службы сервера и возвращает указатель на него.
func new() *gosearch {
	gs := gosearch{}
	gs.router = mux.NewRouter()
	gs.scanner = spider.New()
	gs.index = hash.New()
	gs.storage = memstore.New()
	gs.engine = engine.New(gs.index, gs.storage)
	gs.api = api.New(gs.router, gs.engine)
	gs.sites = []string{"https://go.dev", "https://golang.org/"}
	gs.depth = 2
	gs.addr = ":80"
	return &gs
}

// init производит сканирование сайтов и индексирование данных.
func (gs *gosearch) init() {
	log.Println("Сканирование сайтов")
	chDocs, chErr := gs.scanner.BatchScan(gs.sites, 2, 10)
	go func() {
		for err := range chErr {
			log.Println("ошибка при добавлении сканировании документов:", err)
		}
	}()
	go func() {
		id := 0
		for doc := range chDocs {
			doc.ID = id
			id++
			gs.index.Add([]crawler.Document{doc})
			err := gs.storage.StoreDocs([]crawler.Document{doc})
			if err != nil {
				log.Println("ошибка при сохранении документа в БД:", err)
				continue
			}
		}
		log.Println("Сканирование сайтов завершено")

        // log.Println("По запросу", searchString, "найдены следующие ссылки:")
        // searchResults := gs.engine.Search(searchString)
        // log.Println(searchResults)
	}()

}

// run запускает веб-сервер.
func (gs *gosearch) run() {
	log.Println("Запуск http-сервера на интерфейсе:", gs.addr)
	srv := &http.Server{
		ReadTimeout:  40 * time.Second,
		WriteTimeout: 40 * time.Second,
		Handler:      gs.router,
		Addr:         gs.addr,
	}
	listener, err := net.Listen("tcp4", srv.Addr)
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(srv.Serve(listener))
}

