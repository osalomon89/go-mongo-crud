package main

import (
	"context"
	"encoding/json"
	"fmt"
	_ "log"
	"net/http"

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

type ItemResponse struct {
	ID string `json:"id"`
	ItemRequestBody
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

	id, err := createRecord(req.Context(), data)
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
		Data   ItemResponse
	}{
		Status: http.StatusOK,
		Data: ItemResponse{
			ID:              id,
			ItemRequestBody: *data,
		},
	})
}

func getRecords(ctx context.Context) ([]ItemRequestBody, error) {
	collection := client.Database(dbName).Collection("items")

	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []ItemRequestBody

	for cursor.Next(ctx) {
		var item ItemRequestBody

		if err = cursor.Decode(&item); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, nil
}

func createRecord(ctx context.Context, data *ItemRequestBody) (string, error) {
	collection := client.Database(dbName).Collection("items")

	req, err := collection.InsertOne(ctx, data)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", req.InsertedID), nil
}
