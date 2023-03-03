package main

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	//go:embed store.sql
	sql     string
	pg      *pgx.Conn
	count   uint64
	reqChan chan *http.Request
)

func init() {
	reqChan = make(chan *http.Request, 10)
}

func main() {
	ctx, cancle := context.WithCancel(context.Background())
	defer cancle()

	dburl := os.Getenv("DATABASE_URL")
	if dburl == "" {
		dburl = "postgres://username:password@localhost:5432/views_count"
	}

	conn, err := pgx.Connect(ctx, dburl)
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

	e.Start(":8081")
}

func getBadge(c echo.Context) error {
	reqChan <- c.Request()

	r, err := http.Get(fmt.Sprintf("https://badgen.net/badge/views-counter/%d/green?icon=github", atomic.AddUint64(&count, 1)))
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	c.Response().Header().Set("content-type", r.Header.Get("content-type"))
	b, err := io.ReadAll(r.Body)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.String(http.StatusOK, string(b))
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
