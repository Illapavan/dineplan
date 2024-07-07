package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Age   int    `json:"age"`
}

type UserRepository struct {
	users []User
	mu    sync.RWMutex
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		users: []User{
			{ID: 1, Name: "Pavan Illa", Phone: "9381122977", Age: 24},
			{ID: 2, Name: "John Doe", Phone: "9381122988", Age: 28},
			{ID: 3, Name: "Peter Griffin", Phone: "8978987899", Age: 44},
		},
	}
}

func (ur *UserRepository) GetAll() []User {
	ur.mu.RLock()
	defer ur.mu.RUnlock()
	return ur.users
}

func (ur *UserRepository) GetByID(id int) (User, bool) {
	ur.mu.RLock()
	defer ur.mu.RUnlock()
	for _, user := range ur.users {
		if user.ID == id {
			return user, true
		}
	}
	return User{}, false
}

func (ur *UserRepository) Add(user User) User {
	ur.mu.Lock()
	defer ur.mu.Unlock()
	user.ID = len(ur.users)+1
	ur.users = append(ur.users, user)
	return user
}

func main() {
	app := NewServer()
	userRepo := NewUserRepository()

	shutdownChan := make(chan struct{})

	app.Get("/get-all-users", func(req *Request, res *Response) {
		users := userRepo.GetAll()
		res.Status("200").Json(users)
	})

	app.Get("/get-user/:userId", func(req *Request, res *Response) {
		userIdStr := req.PathParam("userId")
		userId, err := strconv.Atoi(userIdStr)
		if err != nil {
			res.Status("400").Json(map[string]string{"error": "Invalid UserId"})
			return
		}
		user, found := userRepo.GetByID(userId)
		if !found {
			res.Status("404").Json(map[string]string{"error": "User not found"})
		} else {
			res.Status("200").Json(user)
		}
	})

	app.Post("/add-student", func(req *Request, res *Response) {
		user := Body[User](req)

		if user == nil {
			res.Status("400").Json(map[string]string{"error": "Invalid request body"})
			return
		}

		if user.Name == "" {
			res.Status("400").Json(map[string]string{"error": "Name is required"})
			return
		}

		if user.Phone == "" {
			res.Status("400").Json(map[string]string{"error": "Phone is required"})
			return
		}

		if user.Age <= 0 {
			res.Status("400").Json(map[string]string{"error": "Invalid age"})
			return
		}

		newUser := userRepo.Add(*user)
		res.Status("201").Json(map[string]interface{}{
			"message": "Student added successfully",
			"user":    newUser,
		})
	})

	go func() {
		fmt.Println("Server is starting on port : 8000")
		if err := app.Listen(8000); err != nil {
			fmt.Printf("Server error: %s\n", err)
			close(shutdownChan)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	select {
	case <-sigChan:
		fmt.Println("Received OS signal, shutting down")
	case <-shutdownChan:
		fmt.Println("Received signal shutdown, shutting down")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %s\n", err)
	}

	fmt.Println("Server gracefully stopped")
}
