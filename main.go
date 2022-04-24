package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	DB     *mongo.Database
}

var mg MongoInstance

const (
	dbName   = "fiber-hrms"
	mongoURI = "mongodb://localhost:27017/" + dbName
)

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name,omitempty"`
	Salary float64 `json:"salary,omitempty"`
	Age    float64 `json:"age,omitempty"`
}

func Connect() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		return err
	}

	db := client.Database(dbName)
	mg = MongoInstance{Client: client, DB: db}

	return nil
}

func main() {
	if err := Connect(); err != nil {
		log.Fatalln(err)
	}

	app := fiber.New()

	app.Get("/employee", func(c *fiber.Ctx) error {
		query := bson.D{}
		cursor, err := mg.DB.Collection("employees").Find(c.Context(), query)
		if err != nil {
			return c.Status(500).JSON(err)
		}

		employees := make([]Employee, 0)
		if err := cursor.All(c.Context(), &employees); err != nil {
			return c.Status(500).JSON(err)
		}

		return c.JSON(employees)
	})
	app.Post("/employee", func(c *fiber.Ctx) error {
		collection := mg.DB.Collection("employees")

		employee := new(Employee)
		if err := c.BodyParser(employee); err != nil {
			return c.Status(422).JSON(err)
		}

		employee.ID = ""
		insertOneResult, err := collection.InsertOne(c.Context(), employee)
		if err != nil {
			return c.Status(500).JSON(err)
		}

		// filter := bson.D{{Key: "_id", Value: insertOneResult.InsertedID}}
		// createdRecord := collection.FindOne(c.Context(), filter)

		// createdEmployee := &Employee{}
		// createdRecord.Decode(&createdEmployee)

		// return c.Status(201).JSON(createdEmployee)

		employeeID, _ := insertOneResult.InsertedID.(primitive.ObjectID).MarshalText()
		employee.ID = string(employeeID)

		return c.Status(201).JSON(employee)

	})

	app.Put("/employee/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")

		id, err := primitive.ObjectIDFromHex(idParam)
		if err != nil {
			return c.Status(400).JSON(err)
		}

		employee := new(Employee)
		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).JSON(err)
		}

		query := bson.D{{Key: "_id", Value: id}}
		update := bson.D{
			{
				Key: "$set",
				Value: bson.D{
					{Key: "name", Value: employee.Name},
					{Key: "age", Value: employee.Age},
					{Key: "salary", Value: employee.Salary},
				},
			},
		}

		if err := mg.DB.Collection("employees").FindOneAndUpdate(c.Context(), query, update).Err(); err != nil {
			if err == mongo.ErrNoDocuments {
				return c.SendStatus(400)
			}
			return c.Status(500).JSON(err)
		}
		employee.ID = idParam

		return c.Status(200).JSON(employee)
	})

	app.Delete("/employee/:id", func(c *fiber.Ctx) error {
		id, err := primitive.ObjectIDFromHex(c.Params("id"))
		if err != nil {
			return c.Status(400).JSON(err)
		}

		query := bson.D{{Key: "_id", Value: id}}
		deleteResult, err := mg.DB.Collection("employees").DeleteOne(c.Context(), query)

		if err != nil {
			return c.Status(500).JSON(err)
		}

		if deleteResult.DeletedCount < 1 {
			return c.SendStatus(404)
		}

		return c.SendStatus(204)
	})

	log.Fatalln(app.Listen(":3000"))
}
