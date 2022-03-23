package main

import (
    "fmt"
    "encoding/json"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type DummyStruct struct {
    User     string  `bson:"user" json:"user"`
    Foo      FooType `bson:"foo" json:"foo"`
}

type FooType struct {
    BarA int `bson:"barA" json:"barA"`
    BarB int `bson:"bar_b" json:"bar_b"`
}

func fetchExpensiveItems(origin string, minPrice float64) ([]Item, error) {
	session, err := mgo.Dial("mongodb://localhost:27017")
	if err != nil {
		return nil, err
	}
	session.SetMode(mgo.Monotonic, true)
	c := session.DB("subdoctest").C("catalog")
	defer session.Close()

	//create the aggregator pipeline that will fetch just the needed data from MongoDB, and nothing more
	pipe := c.Pipe([]bson.M{
		{"$match": bson.M{
			"brands": bson.M{
				"$elemMatch": bson.M{
					"items.origin": bson.M{"$eq": origin},
					"items.price":  bson.M{"$gte": minPrice},
				},
			},
		}},
		{"$project": bson.M{"_id": 0, "brands": 1}},
		{"$addFields": bson.M{
			"brands": bson.M{
				"$filter": bson.M{
					"input": bson.M{
						"$map": bson.M{
							"input": "$brands",
							"as":    "b",
							"in": bson.M{
								"items": bson.M{
									"$filter": bson.M{
										"input": "$$b.items",
										"as":    "i",
										"cond": bson.M{
											"$and": []interface{}{
												bson.M{"$eq": []interface{}{"$$i.origin", origin}},
												bson.M{"$gte": []interface{}{"$$i.price", minPrice}},
											},
										},
									},
								},
							},
						},
					},
					"as":   "b",
					"cond": bson.M{"$gt": []interface{}{bson.M{"$size": "$$b.items"}, 0}},
				},
			},
		},
		}})

	//execute the aggregation query
	var resp []bson.M
	err = pipe.All(&resp)
	if err != nil {
		return nil, err
	}

	//traverse the bson Map returned by the aggregation and extract the items
	var itemsFound []Item
	for _, catalogMap := range resp {
		brands := catalogMap["brands"].([]interface{})
		for _, b := range brands {
			brandsMap := b.(bson.M)
			items := brandsMap["items"].([]interface{})
			for _, b := range items {
				itemsMap := b.(bson.M)
				data, _ := json.Marshal(itemsMap)
				var item Item
				if err := json.Unmarshal(data, &item); err != nil {
					return nil, err
				}
				itemsFound = append(itemsFound, item)
			}
		}
	}

	return itemsFound, err
}

type Item struct {
	Name   string  `bson:"name" json:"name"`
	Origin string  `bson:"origin" json:"origin"`
	Price  float64 `bson:"price" json:"price"`
}

func main() {
    test:=DummyStruct{}
    test.User="test"
    test.Foo.BarA=123
    test.Foo.BarB=321
    b,err:=json.Marshal(test)
    if err!=nil{
        fmt.Println("error marshaling test struct",err)
        return
    }
    fmt.Println("test data\n",string(b))


	items, err := fetchExpensiveItems("Italy", 200)
	if err != nil {
		fmt.Printf("Failed with error: %v", err)
	}

	fmt.Printf("Items matching criteria: %+v", items)

}