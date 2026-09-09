package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"example/httpservice/internal/presenter"
	"example/httpservice/internal/repository"
	"example/httpservice/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://shop:shoppass@localhost:5432/shop"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("cannot configure database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("cannot reach database: %v", err)
	}

	userRepo := repository.NewUserRepository(pool)
	userUsecase := usecase.NewUserUsecase(userRepo)
	userPresenter := presenter.NewUserPresenter(userUsecase)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	r.GET("/users", userPresenter.List)
	r.GET("/users/:id", userPresenter.GetByID)
	r.POST("/users", userPresenter.Create)
	r.PATCH("/users/:id", userPresenter.Patch)
	r.DELETE("/users/:id", userPresenter.Delete)

	log.Println("listening on :8080")
	r.Run(":8080")
}
