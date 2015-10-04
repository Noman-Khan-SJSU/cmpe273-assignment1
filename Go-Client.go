// Author: Noman Khan, SJSU (CMPE 273, Fall 2015)

package main

import (

	"github.com/bitly/go-simplejson"
	"os"
	"strconv"
	"strings"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type BuyingStockRequest struct {
	StockSymbolAndPercentage string
	Budget                   float32
}

type BuyingStockResponse struct {
	TradeID         int
	Stocks           []string
	UninvestedAmount float32
}

type CheckingPortfolio struct {
	Stocks           []string
	UninvestedAmount float32
	CurrentMarketValue float32
}

func main() {

	
	if len(os.Args) > 4 || len(os.Args) < 2 {
		fmt.Println("Wrong number of arguments!")
		usage()
		return

	} else if len(os.Args) == 2 { 

		_, err := strconv.ParseInt(os.Args[1], 10, 64)
		if err != nil {
			fmt.Println("Illegal argument!")
			usage()
			return
		}

		// chkResp := new(CheckingPortfolio)

		data, err := json.Marshal(map[string]interface{}{
			"method": "StockRecords.Check",
			"id":     1,
			"params": []map[string]interface{}{map[string]interface{}{"TradeId": os.Args[1]}},
		})

		if err != nil {
			log.Fatalf("Marshal : %v", err)
		}

		resp, err := http.Post("http://127.0.0.1:1333/rpc", "application/json", strings.NewReader(string(data)))

		if err != nil {
			log.Fatalf("Post: %v", err)
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			log.Fatalf("ReadAll: %v", err)
		}

		newjson, err := simplejson.NewJson(body)

		checkError(err)

		fmt.Print("stocks: ")
		stocks := newjson.Get("result").Get("Stocks")
		fmt.Println(stocks)

		fmt.Print("uninvested amount: ")
		uninvestedAmount, _ := newjson.Get("result").Get("UninvestedAmount").Float64()
		fmt.Print("$")
		fmt.Println(uninvestedAmount)

		fmt.Print("current market value: ")
		CurrentMarketValue, _ := newjson.Get("result").Get("CurrentMarketValue").Float64()
		fmt.Print("$")
		fmt.Println(CurrentMarketValue)

	} else if len(os.Args) == 3 {
		budget, err := strconv.ParseFloat(os.Args[2], 64)
		if err != nil {
			fmt.Println("Wrong budget argument.")
			usage()
			return
		}

		data, err := json.Marshal(map[string]interface{}{
			"method": "StockRecords.Buy",
			"id":     2,
			"params": []map[string]interface{}{map[string]interface{}{"StockSymbolAndPercentage": os.Args[1], "Budget": float32(budget)}},
		})

		if err != nil {
			log.Fatalf("Marshal : %v", err)
		}

		resp, err := http.Post("http://127.0.0.1:1333/rpc", "application/json", strings.NewReader(string(data)))

		if err != nil {
			log.Fatalf("Post: %v", err)
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			log.Fatalf("ReadAll: %v", err)
		}

		newjson, err := simplejson.NewJson(body)

		checkError(err)

		fmt.Print("TradeID: ")
		TradeID, _ := newjson.Get("result").Get("TradeID").Int()
		fmt.Println(TradeID)

		fmt.Print("stocks: ")
		stocks := newjson.Get("result").Get("Stocks")
		fmt.Println(*stocks)

		fmt.Print("uninvested amount: ")
		uninvestedAmount, _ := newjson.Get("result").Get("UninvestedAmount").Float64()
		fmt.Print("$")
		fmt.Println(uninvestedAmount)

	} else {
		fmt.Println("Unknown error.")
		usage()
		return
	}

}

//Error Handling 
func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		log.Fatal("error: ", err)
		os.Exit(2)
	}

}

//Printing the Usage
func usage() {

	fmt.Println("Usage: ", os.Args[0], "tradeId")
	fmt.Println("or")
	fmt.Println(os.Args[0], "“GOOG:45%,YHOO:55%” 75000")
}
