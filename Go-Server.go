// Author: Noman Khan, SJSU (CMPE 273, Fall 2015)

package main

import (

	"github.com/bakins/net-http-recover"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
	"github.com/justinas/alice"
	"github.com/bitly/go-simplejson"
	"io/ioutil"
	"log"
	"math"
	"strconv"
	"strings"
	"errors"
	"fmt"
	"net/http"
	"os"

)

type StockRecords struct {
	stockPortfolio map[int](*Portfolio)
}

type Portfolio struct {
	stocks           map[string](*Share)
	uninvestedAmount float32
}

type Share struct {
	boughtPrice float32
	shareNum    int
}

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

type CheckingRequest struct {
	TradeId string
}


var st StockRecords


var tradeId int


func (st *StockRecords) Buy(httpRq *http.Request, rq *BuyingStockRequest, rsp *BuyingStockResponse) error {


	tradeId++
	rsp.TradeID = tradeId

	//Setup account if it doesn't already exists
	if st.stockPortfolio == nil {

		st.stockPortfolio = make(map[int](*Portfolio))

		st.stockPortfolio[tradeId] = new(Portfolio)
		st.stockPortfolio[tradeId].stocks = make(map[string]*Share)

	}

	//Parsing the arguments for buying the stocks
	symbolAndPercentages := strings.Split(rq.StockSymbolAndPercentage, ",")
	newbudget := float32(rq.Budget)
	var spent float32

	for _, stk := range symbolAndPercentages {


		splited := strings.Split(stk, ":")
		stkQuote := splited[0]
		percentage := splited[1]
		strPercentage := strings.TrimSuffix(percentage, "%")
		floatPercentage64, _ := strconv.ParseFloat(strPercentage, 32)
		floatPercentage := float32(floatPercentage64 / 100.00)
		currentPrice := checkQuote(stkQuote)

		shares := int(math.Floor(float64(newbudget * floatPercentage / currentPrice)))
		sharesFloat := float32(shares)
		spent += sharesFloat * currentPrice

		// Setting up portfolio for new trade Ids
		if _, ok := st.stockPortfolio[tradeId]; !ok {

			newPortfolio := new(Portfolio)
			newPortfolio.stocks = make(map[string]*Share)
			st.stockPortfolio[tradeId] = newPortfolio
		}
		if _, ok := st.stockPortfolio[tradeId].stocks[stkQuote]; !ok {

			newShare := new(Share)
			newShare.boughtPrice = currentPrice
			newShare.shareNum = shares
			st.stockPortfolio[tradeId].stocks[stkQuote] = newShare
		} else {

			total := float32(sharesFloat*currentPrice) + float32(st.stockPortfolio[tradeId].stocks[stkQuote].shareNum)*st.stockPortfolio[tradeId].stocks[stkQuote].boughtPrice
			st.stockPortfolio[tradeId].stocks[stkQuote].boughtPrice = total / float32(shares+st.stockPortfolio[tradeId].stocks[stkQuote].shareNum)
			st.stockPortfolio[tradeId].stocks[stkQuote].shareNum += shares
		}

		stockBought := stkQuote + ":" + strconv.Itoa(shares) + ":$" + strconv.FormatFloat(float64(currentPrice), 'f', 2, 32)

		rsp.Stocks = append(rsp.Stocks, stockBought)
	}

	//Calculation of the un-invested amount
	leftOver := newbudget - spent
	rsp.UninvestedAmount = leftOver
	st.stockPortfolio[tradeId].uninvestedAmount += leftOver

	return nil
}


func (st *StockRecords) Check(httpRq *http.Request, checkRq *CheckingRequest, checkResp *CheckingPortfolio) error {

	if st.stockPortfolio == nil {
		return errors.New("No account set up yet.")
	}


	TradeID64, err := strconv.ParseInt(checkRq.TradeId, 10, 64)

	if err != nil {
		return errors.New("Illegal Trade ID. ")
	}
	TradeID := int(TradeID64)

	if pocket, ok := st.stockPortfolio[TradeID]; ok {

		var currentMarketVal float32
		for stockquote, sh := range pocket.stocks {

			currentPrice := checkQuote(stockquote)


			var str string
			if sh.boughtPrice < currentPrice {
				str = "+$" + strconv.FormatFloat(float64(currentPrice), 'f', 2, 32)
			} else if sh.boughtPrice > currentPrice {
				str = "-$" + strconv.FormatFloat(float64(currentPrice), 'f', 2, 32)
			} else {
				str = "$" + strconv.FormatFloat(float64(currentPrice), 'f', 2, 32)
			}


			entry := stockquote + ":" + strconv.Itoa(sh.shareNum) + ":" + str

			checkResp.Stocks = append(checkResp.Stocks, entry)

			currentMarketVal += float32(sh.shareNum) * currentPrice
		}


		checkResp.UninvestedAmount = pocket.uninvestedAmount

		//Calculation of the current market value
		checkResp.CurrentMarketValue = currentMarketVal
	} else {
		return errors.New("No such trade ID. ")
	}

	return nil
}

func main() {

	//Creating stock records
	var st = (new(StockRecords))

	//TradeID Initialization
	tradeId = 0


	router := mux.NewRouter()
	server := rpc.NewServer()
	server.RegisterCodec(json.NewCodec(), "application/json")
	server.RegisterService(st, "")

	chain := alice.New(
		func(h http.Handler) http.Handler {
			return handlers.CombinedLoggingHandler(os.Stdout, h)
		},
		handlers.CompressHandler,
		func(h http.Handler) http.Handler {
			return recovery.Handler(os.Stderr, h, true)
		})

	router.Handle("/rpc", chain.Then(server))
	log.Fatal(http.ListenAndServe(":1333", server))



}

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func checkQuote(stockName string) float32 {

	baseUrlLeft := "https://query.yahooapis.com/v1/public/yql?q=select%20LastTradePriceOnly%20from%20yahoo.finance%0A.quotes%20where%20symbol%20%3D%20%22"
	baseUrlRight := "%22%0A%09%09&format=json&env=http%3A%2F%2Fdatatables.org%2Falltables.env"


	resp, err := http.Get(baseUrlLeft + stockName + baseUrlRight)

	if err != nil {
		log.Fatal(err)
	}


	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		log.Fatal(err)
	}


	if resp.StatusCode != 200 {
		log.Fatal("Query failure, possibly no network connection or illegal stock quote ")
	}


	newjson, err := simplejson.NewJson(body)
	if err != nil {
		fmt.Println(err)
	}

	price, _ := newjson.Get("query").Get("results").Get("quote").Get("LastTradePriceOnly").String()
	floatPrice, err := strconv.ParseFloat(price, 32)

	return float32(floatPrice)
}
