package main

import (
	"context"
	"encoding/json"
	"fmt"
	_ "log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	dbUser = "admin"
	dbPass = "secret"
	dbName = "shop_db"
)

var client *mongo.Client

type ItemRequestBody struct {
	Code        string   `json:"code"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Price       int      `json:"price"`
	Stock       int      `json:"stock"`
	Photos      []string `json:"photos"`
}

type Item struct {
	Code        string     `bson:"code" json:"code"`
	Title       string     `bson:"title" json:"title"`
	Description string     `bson:"description" json:"description"`
	Price       int        `bson:"price" json:"price"`
	Stock       int        `bson:"stock" json:"stock"`
	Photos      []string   `bson:"photos" json:"photos"`
	Status      string     `bson:"status" json:"status"`
	CreatedAt   *time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt   *time.Time `bson:"updated_at" json:"updated_at"`
}

type ItemResponse struct {
	Status int    `json:"status"`
	ID     string `json:"id"`
	Data   Item   `json:"data"`
}

func main() {
	var err error
	ctx := context.Background()

	client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://"+dbUser+":"+dbPass+"@localhost:27017"))
	if err != nil {
		panic("error connecting to DB: " + err.Error())
	}

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/items", getItemsHandler).Methods("GET")
	router.HandleFunc("/api/v1/items", createItemHandler).Methods("POST")

	err = http.ListenAndServe(":8080", router)
	if err != nil {
		panic("error running http server: " + err.Error())
	}
	fmt.Println("running")
}

func getItemsHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	result, err := getRecords(req.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(struct {
			Status int
			Data   interface{}
		}{
			Status: http.StatusInternalServerError,
			Data:   err,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		Status int
		Data   interface{}
	}{
		Status: http.StatusOK,
		Data:   result,
	})
}

func createItemHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	data := new(ItemRequestBody)
	err := json.NewDecoder(req.Body).Decode(data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(struct {
			Status int
			Data   interface{}
		}{
			Status: http.StatusBadRequest,
			Data:   err,
		})
		return
	}

	var status string
	if status = "active"; data.Stock == 0 {
		status = "inactive"
	}

	timeNow := time.Now()

	itemToSave := Item{
		Code:        data.Code,
		Title:       data.Title,
		Description: data.Description,
		Price:       data.Price,
		Stock:       data.Stock,
		Photos:      data.Photos,
		Status:      status,
		CreatedAt:   &timeNow,
		UpdatedAt:   &timeNow,
	}

	collection := client.Database(dbName).Collection("items")

	result, err := collection.InsertOne(req.Context(), itemToSave)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(struct {
			Status int
			Data   interface{}
		}{
			Status: http.StatusInternalServerError,
			Data:   err,
		})
		return
	}

	id := fmt.Sprintf("%v", result.InsertedID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ItemResponse{
		Status: http.StatusOK,
		ID:     id,
		Data:   itemToSave,
	})
}

func getRecords(ctx context.Context) ([]Item, error) {
	collection := client.Database(dbName).Collection("items")

	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []Item

	for cursor.Next(ctx) {
		var item Item

		if err = cursor.Decode(&item); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, nil
}
