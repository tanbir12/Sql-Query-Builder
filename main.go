package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

type QueryRequest struct {
	Query     string
	Submitted bool
}

type QueryResponse struct {
	Columns []string
	Results [][]interface{}
	Error   string
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("mysql", "root:Tanbir@12345@tcp(localhost:3306)/testdb")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	http.HandleFunc("/", executeQuery)

	log.Println("Server is running on port 8088...")
	log.Fatal(http.ListenAndServe(":8088", nil))
}

func executeQuery(w http.ResponseWriter, r *http.Request) {
	data := QueryRequest{}
	response := QueryResponse{}

	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}

		data.Submitted = true
		data.Query = r.FormValue("queries")
	}

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Unable to load template", http.StatusInternalServerError)
		return
	}

	if data.Query != "" {

		// If the query have prefix select but no limit
		// if !strings.Contains(strings.ToLower(data.Query), "limit") && strings.HasPrefix(strings.ToLower(data.Query), "select") {
		// 	query = query[:len(query)-1] + " LIMIT 1 ;"
		// }

		rows, err := db.Query(data.Query)
		if err != nil {
			response.Error = fmt.Sprintf("Query error: %v", err)
		} else {
			defer rows.Close()

			response.Columns, err = rows.Columns()
			if err != nil {
				response.Error = fmt.Sprintf("Error retrieving columns: %v", err)
			} else {
				for rows.Next() {
					record := make([]interface{}, len(response.Columns))
					recordPtrs := make([]interface{}, len(response.Columns))
					for i := range record {
						recordPtrs[i] = &record[i]
					}

					if err := rows.Scan(recordPtrs...); err != nil {
						response.Error = fmt.Sprintf("Error scanning row: %v", err)
						return
					}

					for i, rawValue := range record {
						if b, ok := rawValue.([]byte); ok {
							record[i] = string(b)
						}
					}

					response.Results = append(response.Results, record)
				}
			}
		}
	}

	err = tmpl.Execute(w, struct {
		Query string
		QueryResponse
		Submitted bool
	}{
		Query:         data.Query,
		QueryResponse: response,
		Submitted:     data.Submitted,
	})
	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}
