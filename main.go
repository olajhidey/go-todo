package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func setupSQlite() *sql.DB {
	database, err := sql.Open("sqlite3", "./todo.db")
	if err != nil {
		panic(err)
	}

	println("Database created!")

	return database
}

func createTable(database *sql.DB) {
	createTableSQL := `CREATE TABLE IF NOT EXISTS todos (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"title" TEXT,
		"description" TEXT,
		"isCompleted" INTEGER,
		"completedDate" TEXT
	)`

	statement, err := database.Prepare(createTableSQL)
	if err != nil {
		panic(err)
	}
	statement.Exec()
	fmt.Println("Users table created")
}

type Todo struct {
	Id            int    `json:"id,omitempty"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	IsCompleted   bool   `json:"isCompleted"`
	CompletedDate string `json:"completedDate"`
}

func createTodo(database *sql.DB, todo Todo) Todo {

	insertTodoSQL := `INSERT INTO todos (title, description, isCompleted, completedDate ) VALUES (?, ?, ?, ?)`
	statement, err := database.Prepare(insertTodoSQL)

	if err != nil {
		panic(err)
	}
	result, err := statement.Exec(todo.Title, todo.Description, todo.IsCompleted, todo.CompletedDate)

	if err != nil {
		panic(err)
	}
	fmt.Println("Inserted todo:", todo)

	lastInsertedId, err := result.LastInsertId()

	println("Id: ", lastInsertedId)

	if err != nil {
		panic(err)
	}

	return Todo{
		Id:            int(lastInsertedId),
		Title:         todo.Title,
		Description:   todo.Description,
		CompletedDate: todo.CompletedDate,
		IsCompleted:   todo.IsCompleted,
	}

}

func getAllTodos(database *sql.DB) []Todo {

	var results []Todo

	row, err := database.Query("SELECT * FROM todos ORDER BY id")
	if err != nil {
		panic(err)
	}

	defer row.Close()

	for row.Next() {
		var id int
		var title string
		var description string
		var isCompleted bool
		var completedDate string

		row.Scan(&id, &title, &description, &isCompleted, &completedDate)
		results = append(results, Todo{
			Id:            id,
			Title:         title,
			Description:   description,
			CompletedDate: completedDate,
			IsCompleted:   isCompleted,
		})
	}

	return results
}

// Get a particular todo information
func getTodo(database *sql.DB, id int) (*Todo, error) {

	var todo Todo

	getTodoQuery := "SELECT id, title, description, isCompleted, completedDate FROM todos WHERE id = ?"

	row := database.QueryRow(getTodoQuery, id)

	err := row.Scan(&todo.Id, &todo.Title, &todo.Description, &todo.IsCompleted, &todo.CompletedDate)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}

		return nil, fmt.Errorf("GetUserById: %v", err)
	}

	return &todo, nil

}

// Update a user todo
func updateTodo(database *sql.DB, id string, todo Todo) (int, error) {

	updateQuery := "UPDATE todos SET title = ?, description = ?, completedDate = ?, isCompleted = ? WHERE id = ?"

	result, err := database.Exec(updateQuery, todo.Title, todo.Description, todo.CompletedDate, todo.IsCompleted, id)

	if err != nil {
		return 500, err
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		return 500, err
	}

	if rowsAffected == 0 {
		return 404, nil
	}

	return 200, nil

}

func deleteTodo(database *sql.DB, id string) (int, error) {

	deleteQuery := " DELETE FROM todos WHERE id = ?"
	result, err := database.Exec(deleteQuery, id)

	if err != nil {
		return 500, err
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		return 500, err
	}

	if rowsAffected == 0 {
		return 404, nil
	}

	return 200, nil

}

func main() {

	router := gin.Default()

	database := setupSQlite()

	// Create table operation
	createTable(database)

	defer database.Close()

	router.Static("/static", "./www")

	router.GET("/", func(ctx *gin.Context) {
		ctx.File("./www/index.html")
	})

	api := router.Group("/api")
	{

		api.GET("/todos", func(ctx *gin.Context) {
			todos := getAllTodos(database)

			ctx.JSON(http.StatusOK, todos)
		})

		api.POST("/todo/create", func(ctx *gin.Context) {

			var todo Todo

			if err := ctx.BindJSON(&todo); err != nil {
				log.Println(err)
				ctx.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			insertedTodo := createTodo(database, todo)

			ctx.JSON(http.StatusCreated, gin.H{
				"id":            insertedTodo.Id,
				"title":         insertedTodo.Title,
				"description":   insertedTodo.Description,
				"completedDate": insertedTodo.CompletedDate,
				"isCompleted":   insertedTodo.IsCompleted,
			})

		})

		api.GET("/todo/view/:id", func(ctx *gin.Context) {

			id := ctx.Params.ByName("id")

			userId, _ := strconv.Atoi(id)
			todo, err := getTodo(database, userId)

			if err != nil {

				if err == sql.ErrNoRows {
					ctx.JSON(http.StatusNotFound, gin.H{
						"message": "Todo not found",
					})
					return
				}

				ctx.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			ctx.JSON(http.StatusOK, todo)

		})

		api.PUT("/todo/update/:id", func(ctx *gin.Context) {

			var todo Todo

			id := ctx.Param("id")

			if err := ctx.ShouldBindBodyWithJSON(&todo); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid body request",
				})
				return
			}

			code, err := updateTodo(database, id, todo)

			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"message": err.Error(),
				})
				return
			}

			if code == 404 {
				ctx.JSON(http.StatusNotFound, gin.H{
					"message": "Todo ID not found",
				})
				return
			}

			ctx.JSON(http.StatusOK, gin.H{
				"message": "Todo updated successfully!",
			})

		})

		api.DELETE("/todo/delete/:id", func(ctx *gin.Context) {
			id := ctx.Param("id")
			code, err := deleteTodo(database, id)

			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"message": err.Error(),
				})
				return
			}

			if code == 404 {
				ctx.JSON(http.StatusNotFound, gin.H{
					"message": "Todo ID not found",
				})
				return
			}

			ctx.JSON(http.StatusOK, gin.H{
				"message": "Todo deleted successfully",
			})
		})
	}

	httpPort := "8080"

	srv := &http.Server{
		Addr:              ":" + httpPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Printf("Failed to start server: %v", err)
	}

}
