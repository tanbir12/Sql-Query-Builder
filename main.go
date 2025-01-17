package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

// Structure to handle query input
type QueryRequest struct {
	Query     string
	Submitted bool
}

type QueryResponse struct {
	Results []map[string]interface{} `json:"results,omitempty"`
	Error   string                   `json:"error,omitempty"`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("mysql", "root:Tanbir@12345@tcp(localhost:3306)/testdb")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	http.HandleFunc("/", executeQuery)

	log.Println("Server is running on port 8088...")
	log.Fatal(http.ListenAndServe(":8088", nil))
}

func executeQuery(w http.ResponseWriter, r *http.Request) {

	data := QueryRequest{}
	results := []map[string]interface{}{}

	if r.Method == http.MethodPost {
		// Parse form data
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}

		// Populate form data
		data.Submitted = true
		data.Query = r.FormValue("queries")
	}

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Unable to load template", http.StatusInternalServerError)
		return
	}

	var columns []string

	if data.Query != "" {
		var query = data.Query
		// If the query have prefix select but no limit
		// if !strings.Contains(strings.ToLower(data.Query), "limit") && strings.HasPrefix(strings.ToLower(data.Query), "select") {
		// 	query = query[:len(query)-1] + " LIMIT 1 ;"
		// }

		rows, err := db.Query(query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		columns, err = rows.Columns() // Get column names
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for rows.Next() {
			record := make([]interface{}, len(columns))
			recordPtrs := make([]interface{}, len(columns))
			for i := range record {
				recordPtrs[i] = &record[i]
			}

			if err := rows.Scan(recordPtrs...); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			rowMap := make(map[string]interface{})
			for i, colName := range columns {
				rawValue := record[i]
				if b, ok := rawValue.([]byte); ok {
					rowMap[colName] = string(b)
				} else {
					rowMap[colName] = rawValue
				}
			}

			results = append(results, rowMap)

		}
	}

	// Render the template with the query results
	err = tmpl.Execute(w, struct {
		Query     string
		Columns   []string
		Results   []map[string]interface{}
		Submitted bool
	}{
		Query:     data.Query,
		Columns:   columns,
		Results:   results,
		Submitted: data.Submitted,
	})
	if err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}
