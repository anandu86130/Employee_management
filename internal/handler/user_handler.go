package handler

import (
	"context"
	"employee-management/internal/model"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type UserHandler struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewUserHandler(e *echo.Echo, db *gorm.DB) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
		DB:   0,
	})

	// Check Redis connection
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
		panic("Failed to connect to redis")
	}

	handler := &UserHandler{db: db, rdb: rdb}

	e.POST("/employees", handler.CreateEmployee)
	e.GET("/employees/:id", handler.GetEmployee)
	e.PUT("/employees/:id", handler.UpdateEmployee)
	e.DELETE("/employees/:id", handler.DeleteEmployee)
	e.GET("/employees", handler.ListEmployees)
}

func (h *UserHandler) CreateEmployee(c echo.Context) error {
	var input struct {
		Name      string `json:"name"`
		Position  string `json:"position"`
		Salary    uint   `json:"salary"`
		HiredDate string `json:"hired_date"` // Use string for date input
	}

	// Bind the request body to the input struct
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	// Parse the hired_date manually
	hiredDate, err := time.Parse("2006-01-02", input.HiredDate) // Ensure format matches input
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid date format"})
	}

	// Map input to Employee model
	employee := model.Employee{
		Name:       input.Name,
		Position:   input.Position,
		Salary:     input.Salary,
		Hired_date: hiredDate,
	}

	// Save to database
	if err := h.db.Create(&employee).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create employee"})
	}

	// Create response with selected fields
	response := map[string]interface{}{
		"id":         employee.ID,
		"name":       employee.Name,
		"position":   employee.Position,
		"salary":     employee.Salary,
		"hired_date": employee.Hired_date,
		"created_at": employee.CreatedAt,
	}

	// Serialize the response to JSON
	cachedData, err := json.Marshal(response)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to serialize employee data"})
	}

	// Cache the employee data in Redis
	cacheKey := "employee:" + strconv.Itoa(int(employee.ID))
	_, err = h.rdb.Set(context.Background(), cacheKey, cachedData, 24*time.Hour).Result()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to cache employee"})
	}

	return c.JSON(http.StatusCreated, response)
}

func (h *UserHandler) GetEmployee(c echo.Context) error {
	employeeID := c.Param("id")
	if employeeID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "EmployeeID is required"})
	}

	id, err := strconv.Atoi(employeeID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid Employee ID format"})
	}

	// Check if employee data exists in Redis cache
	cacheKey := "employee:" + strconv.Itoa(id)
	cachedEmployee, err := h.rdb.Get(context.Background(), cacheKey).Result()
	if err == nil {
		// If data is found in Redis, unmarshal the cached response to a struct
		var cachedData map[string]interface{}
		if err := json.Unmarshal([]byte(cachedEmployee), &cachedData); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to unmarshal cached data"})
		}
		// Return the cached data
		return c.JSON(http.StatusOK, cachedData)
	} else if err != redis.Nil {
		// If there is another error with Redis, return internal server error
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve employee from cache"})
	}

	// If not found in cache, retrieve the employee from the database
	var employee model.Employee
	if err := h.db.First(&employee, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Employee not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve employee"})
	}

	// Create response with selected fields
	response := map[string]interface{}{
		"id":         employee.ID,
		"name":       employee.Name,
		"position":   employee.Position,
		"salary":     employee.Salary,
		"hired_date": employee.Hired_date,
		"created_at": employee.CreatedAt,
	}

	// Return the employee data without re-caching it again
	return c.JSON(http.StatusOK, response)
}

func (h *UserHandler) UpdateEmployee(c echo.Context) error {
	employeeID := c.Param("id")
	if employeeID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "EmployeeID is required"})
	}

	id, err := strconv.Atoi(employeeID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid Employee ID format"})
	}

	// Check if the employee exists
	var employee model.Employee
	if err := h.db.First(&employee, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Employee not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve employee"})
	}

	// Bind the request body to a struct
	var input struct {
		Name      string `json:"name"`
		Position  string `json:"position"`
		Salary    uint   `json:"salary"`
		HiredDate string `json:"hired_date"`
	}

	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	// Parse the hired_date manually if provided
	if input.HiredDate != "" {
		hiredDate, err := time.Parse("2006-01-02", input.HiredDate)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid date format"})
		}
		employee.Hired_date = hiredDate
	}

	// Update employee fields if provided
	if input.Name != "" {
		employee.Name = input.Name
	}
	if input.Position != "" {
		employee.Position = input.Position
	}
	if input.Salary != 0 {
		employee.Salary = input.Salary
	}

	// Save the updated employee in the database
	if err := h.db.Save(&employee).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update employee"})
	}

	// Update the Redis cache
	cacheKey := "employee:" + strconv.Itoa(int(employee.ID))
	updatedData := map[string]interface{}{
		"id":         employee.ID,
		"name":       employee.Name,
		"position":   employee.Position,
		"salary":     employee.Salary,
		"hired_date": employee.Hired_date,
		"updated_at": time.Now().UTC(),
	}

	cachedData, err := json.Marshal(updatedData)
	if err == nil {
		_, cacheErr := h.rdb.Set(context.Background(), cacheKey, cachedData, 24*time.Hour).Result()
		if cacheErr != nil {
			log.Printf("Warning: Failed to update cache for employee %d: %v", employee.ID, cacheErr)
		}
	} else {
		log.Printf("Warning: Failed to serialize updated employee data: %v", err)
	}

	return c.JSON(http.StatusOK, updatedData)
}

func (h *UserHandler) DeleteEmployee(c echo.Context) error {
	// Extract the employee ID from the URL
	employeeID := c.Param("id")
	if employeeID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "EmployeeID is required"})
	}

	id, err := strconv.Atoi(employeeID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid Employee ID format"})
	}

	// Check if the employee exists in the database
	var employee model.Employee
	if err := h.db.First(&employee, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Employee not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve employee"})
	}

	// Delete the employee from the database
	if err := h.db.Delete(&employee).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete employee"})
	}

	// Remove the employee data from Redis cache
	cacheKey := "employee:" + strconv.Itoa(id)
	_, cacheErr := h.rdb.Del(context.Background(), cacheKey).Result()
	if cacheErr != nil && cacheErr != redis.Nil {
		log.Printf("Warning: Failed to delete cache for employee %d: %v", id, cacheErr)
	}

	// Return 204 No Content response
	return c.NoContent(http.StatusNoContent)
}

func (h *UserHandler) ListEmployees(c echo.Context) error {
	// Retrieve all employees from the database
	var employees []model.Employee
	if err := h.db.Find(&employees).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve employees"})
	}

	// Prepare the response with selected fields
	var employeeList []map[string]interface{}
	for _, employee := range employees {
		employeeList = append(employeeList, map[string]interface{}{
			"id":         employee.ID,
			"name":       employee.Name,
			"position":   employee.Position,
			"salary":     employee.Salary,
			"hired_date": employee.Hired_date,
			"created_at": employee.CreatedAt,
		})
	}

	// Return the employee list as a JSON response
	return c.JSON(http.StatusOK, employeeList)
}
