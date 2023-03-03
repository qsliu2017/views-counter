package main

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	//go:embed store.sql
	sql string
	pg  *pgx.Conn
)

func main() {
	dburl := os.Getenv("DATABASE_URL")
	if dburl == "" {
		dburl = "postgres://username:password@localhost:5432/views_count"
	}

	conn, err := pgx.Connect(context.TODO(), dburl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	pg = conn
	defer conn.Close(context.TODO())

	if err := pg.QueryRow(context.TODO(), "SELECT 1 FROM count").Scan(nil); err != nil {
		_, err := pg.Exec(context.TODO(), sql)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to create table count: %v\n", err)
			os.Exit(1)
		}
	}

	e := echo.New()

	e.Use(middleware.Logger())

	e.GET("/", getBadge)

	e.Start(":8081")
}

func getBadge(c echo.Context) error {
	ctx := c.Request().Context()

	tx, err := pg.Begin(ctx)
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, fmt.Sprintf("INSERT INTO count (payload) VALUES ('%+v')", c.Request())); err != nil {
		c.Logger().Error(err)
		return err
	}

	row := tx.QueryRow(ctx, "SELECT COUNT(*) FROM count")
	var count int
	if err := row.Scan(&count); err != nil {
		c.Logger().Error(err)
		return err
	}

	tx.Commit(ctx)

	r, err := http.Get(fmt.Sprintf("https://badgen.net/badge/views-counter/%d/green?icon=github", count))
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
