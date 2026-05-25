package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"

	mysql "github.com/go-sql-driver/mysql"
)

type JSON map[string]any

//go:embed sql/find.sql
var find string

func main() {
	godotenv.Load()

	e := echo.New()

	e.GET("/company", func(c echo.Context) error {
		db := c.Get("db").(*sql.DB)

		name := c.QueryParam("name")
		topStr := c.QueryParam("top")

		top, err := strconv.Atoi(topStr)
		if err != nil {
			top = 1
		}

		if name == "" {
			return c.JSON(http.StatusBadRequest, JSON{
				"message": "cannot resolve empty company name.",
			})
		}

		matches := GetMatchingCompanies(db, Normalize(name), top, c.Request().Context())

		return c.JSON(http.StatusOK, JSON{
			"values": matches,
		})
	}, DabaseConnectionMiddleware())

	log.Fatal(e.Start(":6173"))
}

func Map[T any, U any](input []T, f func(T) U) []U {
	out := make([]U, 0, len(input))

	for _, val := range input {
		out = append(out, f(val))
	}

	return out
}

func DabaseConnectionMiddleware() echo.MiddlewareFunc {
	config := mysql.Config{
		Addr:   os.Getenv("MYSQL_HOST"),
		DBName: os.Getenv("MYSQL_SCHEMA"),
		User:   os.Getenv("MYSQL_USER"),
		Passwd: os.Getenv("MYSQL_PASSWD"),
	}

	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("db", db)

			return next(c)
		}
	}
}

func GetMatchingCompanies(db *sql.DB, normalizedName string, top int, ctx context.Context) []Company {
	var query strings.Builder

	query.WriteString(`
	select id, name
	from company
	where 
	`)

	query.WriteString(
		strings.Join(
			Map(
				strings.Split(normalizedName, " "),
				func(input string) string {
					return "name like ?"
				},
			),
			" or ",
		),
	)

	query.WriteString("\nlimit 100;")

	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	rows, err := tx.Query(
		query.String(),
		Map(
			strings.Split(normalizedName, " "),
			func(input string) any {
				return "%" + input + "%"
			},
		)...,
	)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	var companies []Company

	for rows.Next() {
		var newWord Company

		err = rows.Scan(
			&newWord.ID,
			&newWord.Name,
		)
		if err != nil {
			fmt.Println(err.Error())
			return nil
		}

		companies = append(companies, newWord)
	}

	return Map(
		MatchCompanies(normalizedName, companies, top),
		func(input Candidate) Company {
			return input.Company
		},
	)
}
