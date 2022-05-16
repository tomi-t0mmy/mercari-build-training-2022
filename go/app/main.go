package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"mercari-build-training-2022/app/item_store"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	_ "github.com/mattn/go-sqlite3"
)

const (
	ImgDir = "image"
)

type Response struct {
	Message string `json:"message"`
}
type Item struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Image    string `json:"image"`
}

type Items struct {
	Items []Item `json:"items"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

func getItems(c echo.Context) error {
	// // step3-3
	// // get from json
	// raw, err := ioutil.ReadFile("./app/items.json")
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	return err
	// }
	// items := Items{}
	// json.Unmarshal(raw, &items)
	// res := items

	rows, err := item_store.GetItems()
	defer rows.Close()
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	var items Items
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.Name, &item.Category, &item.Image); err != nil {
			fmt.Println(err.Error())
			return err
		}
		items.Items = append(items.Items, item)
	}
	return c.JSON(http.StatusOK, items)
}

func getItemById(c echo.Context) error {
	strId := c.Param("id")
	id, _ := strconv.Atoi(strId)
	row := item_store.GetItemById(id)
	var item Item
	if err := row.Scan(&item.Name, &item.Category, &item.Image); err != nil {
		if err == sql.ErrNoRows {
			fmt.Println(err.Error())
			message := fmt.Sprintf("No row")
			res := Response{Message: message}
			return c.JSON(http.StatusOK, res)
		} else {
			fmt.Println(err.Error())
			return err
		}
	}

	return c.JSON(http.StatusOK, item)
}

func addItem(c echo.Context) error {
	// Get form data
	name := c.FormValue("name")
	category := c.FormValue("category")
	c.Logger().Infof("Receive item: name:%s, category:%s", name, category)

	file, err := c.FormFile("image")
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("error")
		return err
	}
	src, err := file.Open()
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	defer src.Close()

	fileName := strings.Split(file.Filename, ".")[0]
	sha256 := sha256.Sum256([]byte(fileName))
	hashedFileName := hex.EncodeToString(sha256[:]) + ".jpg"

	saveFile, err := os.Create("image/" + hashedFileName)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	defer saveFile.Close()

	if _, err = io.Copy(saveFile, src); err != nil {
		fmt.Println(err.Error())
		return err
	}

	// // step3-2
	// // save to json
	// raw, err := ioutil.ReadFile("./app/items.json")
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	return err
	// }
	// items := Items{}
	// json.Unmarshal(raw, &items)
	// item := Item{Name: name, Category: category}
	// items.Items = append(items.Items, item)
	// b_items, err := json.Marshal(items)
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	return err
	// }
	// if err = ioutil.WriteFile("./app/items.json", b_items, os.ModePerm); err != nil {
	// 	fmt.Println(err.Error())
	// 	return err
	// }

	if err := item_store.InsertItem(name, category, hashedFileName); err != nil {
		fmt.Println(err.Error())
		return err
	}
	message := fmt.Sprintf("item received: name:%s, category:%s", name, category)
	res := Response{Message: message}

	return c.JSON(http.StatusOK, res)
}

func getImg(c echo.Context) error {
	// Create image path
	imgPath := path.Join(ImgDir, c.Param("imageFilename"))

	if !strings.HasSuffix(imgPath, ".jpg") {
		res := Response{Message: "Image path does not end with .jpg"}
		return c.JSON(http.StatusBadRequest, res)
	}
	if _, err := os.Stat(imgPath); err != nil {
		c.Logger().Debugf("Image not found: %s", imgPath)
		imgPath = path.Join(ImgDir, "default.jpg")
	}
	return c.File(imgPath)
}

func searchNameByKwd(c echo.Context) error {
	keyword := c.FormValue("keyword")
	rows, err := item_store.SerchItems(keyword)
	defer rows.Close()
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	var items Items
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.Name, &item.Category); err != nil {
			fmt.Println(err.Error())
			return err
		}
		items.Items = append(items.Items, item)
	}
	return c.JSON(http.StatusOK, items)
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Logger.SetLevel(log.INFO)

	front_url := os.Getenv("FRONT_URL")
	if front_url == "" {
		front_url = "http://localhost:3000"
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{front_url},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

	// Routes
	e.GET("/", root)
	e.GET("/items", getItems)
	e.POST("/items", addItem)
	e.GET("/image/:itemImg", getImg)
	e.GET("/search", searchNameByKwd)
	e.GET("/items/:id", getItemById)
	e.GET("/image/:imageFilename", getImg)

	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
