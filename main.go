package main

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	//go:embed store.sql
	sql string
	//go:embed badge.svg
	badge string

	databaseUrl string
	address     string
	pg          *pgx.Conn
	count       uint64
	reqChan     chan *http.Request
)

func init() {
	databaseUrl = os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		databaseUrl = "postgres://username:password@localhost:5432/views_count"
	}

	address = os.Getenv("ADDRESS")
	if address == "" {
		address = ":80"
	}

	reqChan = make(chan *http.Request, 10)
}

func main() {
	ctx, cancle := context.WithCancel(context.Background())
	defer cancle()

	conn, err := pgx.Connect(ctx, databaseUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	pg = conn
	defer conn.Close(ctx)

	if err := pg.QueryRow(ctx, "SELECT COUNT(*) FROM count").Scan(&count); err != nil {
		_, err := pg.Exec(ctx, sql)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to create table count: %v\n", err)
			os.Exit(1)
		}
	}

	e := echo.New()

	e.Use(middleware.Logger())

	go requestRecorder(ctx, e.Logger, reqChan)

	e.GET("/", getBadge)

	e.Start(address)
}

func getBadge(c echo.Context) error {
	reqChan <- c.Request()

	c.Response().Header().Set("content-type", "image/svg+xml;charset=utf-8")
	c.Response().Header().Set("cache-control", "public, max-age=120")
	return c.String(http.StatusOK, badgen(atomic.AddUint64(&count, 1)))
}

func requestRecorder(ctx context.Context, logger echo.Logger, reqChan <-chan *http.Request) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-reqChan:
			_, err := pg.Exec(ctx, "INSERT INTO count (payload) VALUES ($1)", fmt.Sprintf("%+v", req))
			if err != nil {
				logger.Error(err)
			}
		}
	}
}

func badgen(count uint64) string {
	return fmt.Sprintf(badge, count, count, count, count)
}
