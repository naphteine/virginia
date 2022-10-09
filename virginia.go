package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	//"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

const (
	tmplParts   = "templates/parts.html"
	tmplIndex   = "templates/index.html"
	tmplAuthors = "templates/authors.html"
)

type BookData struct {
	ID          int
	Name        string
	AuthorName  string
	Year        int
	EntryDate   string
}

type AuthorData struct {
	ID          int
	Name        string
	EntryDate   string
}

type BookAuthorsData struct {
	ID			int
	AuthorID	int
	BookID		int
}

type IndexData struct {
	Books       []BookData
    Authors     []AuthorData
}

var tmpl = make(map[string]*template.Template)

// Database
var db, _ = sql.Open("sqlite3", "./virginia.db")

func getDate() string {
	current := time.Now()
	return current.Format("2006-01-02 15:04:05 -0700")
}

func addAuthorToDB(name string) {
	date := getDate()
	var err error
	stmt, err := db.Prepare("INSERT INTO authors (author_name,author_entry_date) VALUES ($1,$2)")

	if err != nil {
		return
	}

	res, err := stmt.Exec(name, date)

	if err != nil {
		panic(err)
	}

	affect, err := res.RowsAffected()
	
	if err != nil {
		panic(err)
	}

	fmt.Println(affect)
}

func main() {
	// Prepare templates
	tmpl[tmplIndex] = template.Must(template.ParseFiles(tmplIndex, tmplParts))
	tmpl[tmplAuthors] = template.Must(template.ParseFiles(tmplAuthors, tmplParts))

	// Router
	r := mux.NewRouter()
	
	r.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		// Load author data
		row, err := db.Query("SELECT author_id, author_name, author_entry_date FROM authors")
		if err != nil {
			panic(err)
		}
		defer row.Close()

		var authors []AuthorData

		for row.Next() {
			var id int
			var name sql.NullString
			var entry_date sql.NullString

			err = row.Scan(&id, &name, &entry_date)
			if err != nil {
				panic(err)
			}

			authors = append(authors, AuthorData{ID: id, Name: name.String, EntryDate: entry_date.String})
		}

		// Load bookAuthors data
		row, err = db.Query("SELECT id, author_id, book_id FROM bookAuthors")
		if err != nil {
			panic(err)
		}
		defer row.Close()

		var bookAuthors []BookAuthorsData

		for row.Next() {
			var id int
			var authorID int
			var bookID int

			err = row.Scan(&id, &authorID, &bookID)
			if err != nil {
				panic(err)
			}

			bookAuthors = append(bookAuthors, BookAuthorsData{ID: id, AuthorID: authorID, BookID: bookID})
		}
		
		// Load book data
		row, err = db.Query("SELECT book_id, book_name, book_year, book_entry_date FROM books")
		if err != nil {
			panic(err)
		}
		defer row.Close()

		var books []BookData

		for row.Next() {
			var id int
			var name sql.NullString
			var year sql.NullInt32
			var entry_date sql.NullString

			err = row.Scan(&id, &name, &year, &entry_date)
			if err != nil {
				panic(err)
			}

			var authorName string

			// bookAuthor 
			for _, author := range bookAuthors {
				if author.AuthorID == 0 || author.BookID == 0 {
					continue
				}

				if author.BookID == id {
					if len(authorName) > 0 {
						authorName += "; "
					}

					authorName += authors[author.AuthorID-1].Name
				}
			}

			books = append(books, BookData{ID: id, Name: name.String, AuthorName: authorName, Year: int(year.Int32), EntryDate: entry_date.String})
		}

		// Prepare index data
		var indexData IndexData

		indexData.Authors = authors
		indexData.Books = books

		// Execute template with prepared data
		err = tmpl[tmplIndex].Execute(response, indexData)

		if err != nil {
			return
		}
	})

	r.HandleFunc("/authors", func(response http.ResponseWriter, request *http.Request) {
		// Load author data
		row, err := db.Query("SELECT author_id, author_name, author_entry_date FROM authors")
		if err != nil {
			panic(err)
		}
		defer row.Close()

		var authors []AuthorData

		for row.Next() {
			var id int
			var name sql.NullString
			var entry_date sql.NullString

			err = row.Scan(&id, &name, &entry_date)
			if err != nil {
				panic(err)
			}

			authors = append(authors, AuthorData{ID: id, Name: name.String, EntryDate: entry_date.String})
		}

		// Execute template with prepared data
		err = tmpl[tmplAuthors].Execute(response, authors)

		if err != nil {
			return
		}
	})
	
	r.HandleFunc("/post/book", func(response http.ResponseWriter, request *http.Request) {
		// Check request method
		if request.Method != "POST" {
			http.Redirect(response, request, "/error", 302)
			return
		}
		
		// Get fields
		name := request.FormValue("name")
		author := request.FormValue("author")
		year := request.FormValue("year")
		date := getDate()
		
		// Check if name empty
		if name == "" || author == "" {
			http.Redirect(response, request, "/error", 302)
			return
		}

		// Get author id from name
		author_search := db.QueryRow("SELECT author_id FROM authors WHERE author_name=$1", author)
		var author_id int
		err := author_search.Scan(&author_id)

		// If err, then assume it doesn't exist and add it
		if err != nil {
		   fmt.Println("ERR - Author ID not found")

		   addAuthorToDB(author)
		}

		// Try to get id again
		author_search = db.QueryRow("SELECT author_id FROM authors WHERE author_name=$1", author)
		err = author_search.Scan(&author_id)

		if err != nil {
		   fmt.Println("ERR - Failed to acquire Author ID second time")
		   return
		}

		// Add book to database
		stmt, err := db.Prepare("INSERT INTO books (book_name,book_author,book_year,book_entry_date) VALUES ($1,$2,$3,$4)")

		if err != nil {
			http.Redirect(response, request, "/error", 302)
			return
		}

		res, err := stmt.Exec(name, author_id, year, date)

		if err != nil {
			panic(err)
		}

		affect, err := res.RowsAffected()
		if err != nil {
			panic(err)
		}

		fmt.Println(affect)

		// Redirect user
		http.Redirect(response, request, "/", 302)
	})
	
	r.HandleFunc("/post/author", func(response http.ResponseWriter, request *http.Request) {
		// Check request method
		if request.Method != "POST" {
			http.Redirect(response, request, "/error", 302)
			return
		}
		
		// Get fields
		name := request.FormValue("name")
		
		// Check if name empty
		if name == "" {
			http.Redirect(response, request, "/error", 302)
			return
		}

		addAuthorToDB(name)

		// Redirect user
		http.Redirect(response, request, "/", 302)
	})

	// File server
	r.PathPrefix("/res/").Handler(http.StripPrefix("/res/", http.FileServer(http.Dir("static"))))

	// Server
	http.ListenAndServe(":8600", r)

	// Shutting down
	db.Close()
}