package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"github.com/mike-kimani/fechronizo/internal/database"

	_ "github.com/lib/pq"
)

/* holds a connection to a database*/
type apiConfig struct {
	DB *database.Queries
}

func main() {

	godotenv.Load(".env")

	portString := os.Getenv("PORT")
	if portString == "" {
		log.Fatal("Port not found in this environment")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL not found in this environment")
	}

	conn, err := sql.Open("postgres", dbURL)

	if err != nil {
		log.Fatal("Cannot connect to database")
	}

	router := chi.NewRouter()
	/* middleware configuration to allow connection to our server through a browser*/
	router.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://*", "http://*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		// AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	db := database.New(conn)

	apiCfg := apiConfig{
		db,
	}
	go startScraping(db, 10, 1*time.Minute)

	v1Router := chi.NewRouter()
	/* handler only fires on get requests*/
	v1Router.Get("/healthz", handlerReadiness)
	v1Router.Get("/err", handlerErr)
	v1Router.Post("/users", apiCfg.handlerCreateUser)
	v1Router.Get("/users", apiCfg.middlewareAuth(apiCfg.handlerGetUser))
	v1Router.Post("/feeds", apiCfg.middlewareAuth(apiCfg.handlerCreateFeed))
	v1Router.Get("/feeds", apiCfg.handlerGetFeed)
	v1Router.Post("/followfeeds", apiCfg.middlewareAuth(apiCfg.handlerCreateFeedFollow))
	v1Router.Get("/followfeeds", apiCfg.middlewareAuth(apiCfg.handlerGetFeedFollows))
	v1Router.Delete("/followfeedstest", apiCfg.middlewareAuth(apiCfg.handlerDeleteFeedFollows))
	v1Router.Delete("/followfeeds/{feedFollowID}", apiCfg.middlewareAuth(apiCfg.handlerDeleteFeedFollows2))
	v1Router.Get("/userposts", apiCfg.middlewareAuth(apiCfg.handlerGetPostsForUser))

	router.Mount("/v1", v1Router)

	srv := &http.Server{
		Handler: router,
		Addr:    ":" + portString,
	}

	log.Printf("Server starting on port %v", portString)
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
