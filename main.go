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

var (
	db              *sql.DB
	authError       string
	createUserError string
	isRoot          bool = false
)

func main() {
	http.HandleFunc("/", login)
	http.HandleFunc("/executeQuery", executeQueryHandler())
	http.HandleFunc("/createUser", createUser)
	log.Println("Server is running on port 8088...")
	log.Fatal(http.ListenAndServe(":8088", nil))
}

// Login function
func login(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/login.html")
	if err != nil {
		http.Error(w, "Unable to load template", http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		// Database Connection
		authString := fmt.Sprintf("%s:%s@tcp(localhost:3306)/information_schema", username, password)
		db, err = sql.Open("mysql", authString)
		if err != nil {
			authError = "Failed to connect to database"
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		err = db.Ping()
		if err != nil {
			authError = "Invalid username or password"
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Giving access to create user who has Create_user_priv
		row := db.QueryRow("SELECT Create_user_priv FROM mysql.user WHERE User = ?", username)
		var createUserPriv string
		if err := row.Scan(&createUserPriv); err != nil {
			authError = "Failed to check user privileges"
		} else {
			isRoot = (createUserPriv == "Y")
		}

		authError = ""
		http.Redirect(w, r, "/executeQuery", http.StatusSeeOther)
		return
	}

	tmpl.Execute(w, struct {
		Error string
	}{
		Error: authError,
	})
}

// Query handler
func executeQueryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			http.Error(w, "Database connection not established", http.StatusInternalServerError)
			return
		}

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
			Root      bool
		}{
			Query:         data.Query,
			QueryResponse: response,
			Submitted:     data.Submitted,
			Root:          isRoot,
		})
		if err != nil {
			http.Error(w, "Error rendering template", http.StatusInternalServerError)
		}
	}
}

// __________________________________   Create user  function   _________________________________________

func createUser(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/register.html")
	if err != nil {
		http.Error(w, "Unable to load template", http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		query := fmt.Sprintf("CREATE USER '%s'@'localhost' IDENTIFIED BY '%s';", username, password)

		// Execute the query
		_, err = db.Exec(query)
		if err != nil {
			createUserError = "Failed to create user"
		} else {
			createUserError = "User created successfully"
		}

		if err != nil {
			createUserError = "Failed to connect to database"
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/executeQuery", http.StatusSeeOther)
		return
	}

	tmpl.Execute(w, struct {
		Error string
	}{
		Error: createUserError,
	})
}
