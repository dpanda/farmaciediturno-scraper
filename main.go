package main

import (
	"net/http"
	"net/url"
	"github.com/PuerkitoBio/goquery"
	"regexp"
	"strings"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"golang.org/x/net/context"
	"encoding/json"
	"log"
	"os"
)

var errorLogger = log.New(os.Stderr, "ERROR ", log.Llongfile)

type Pharmacy struct {
	Name    string   `json:"name"`
	Address string   `json:"address"`
	Phones  []string `json:"phones"`
}

func main() {
	lambda.Start(HandleRequest)
	/*
	pharmas := scrape("piazza brembana")

	for i := range pharmas {
		pharma := pharmas[i]
		fmt.Println(pharma.Name + "\n" + pharma.Address + "\n" + pharma.Phones[0] + "\n-----")
	}
	*/
}

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	pharmas, err := scrape(request.QueryStringParameters["address"])
	if err != nil {
		return serverError(err)
	}

	jsonPharmas, err := json.Marshal(pharmas)
	if err != nil {
		return serverError(err)
	}
	//return pharmas, nil //events.APIGatewayProxyResponse{Body: pharmas, Headers: headers, StatusCode: 200}, nil
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(jsonPharmas),
	}, nil
}

func scrape(address string) ([]Pharmacy, error) {

	resp, err := http.Get("https://www.farmaciediturno.org/ricercaditurno.asp?indirizzo=" + url.QueryEscape(address))
	if err != nil {
		return nil, err
	}
	htmlDoc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	phoneregexp, _ := regexp.Compile("[0-9]{8,15}")

	pharmas := []Pharmacy{}

	htmlDoc.Find(".sf.mnu.c").NextAll().Each(func(i int, s *goquery.Selection) {

		btag := s.Find("td.bbo b")

		name := btag.Find("a").Text()

		atag := btag.Next()
		atag.Find("b").BeforeHtml("&nbsp;")

		addrs := atag.Text()
		phones := phoneregexp.FindAllString(addrs, 3)

		addrs = phoneregexp.ReplaceAllString(addrs, "")
		addrs = strings.Replace(addrs, "Tel.", "", 1)
		addrs = strings.Replace(addrs, " , ", "", 3)

		pharma := Pharmacy{name, addrs, phones}

		pharmas = append(pharmas, pharma)

	})

	return pharmas, nil
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	errorLogger.Println(err.Error())

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       http.StatusText(http.StatusInternalServerError),
	}, nil
}

func clientError(status int) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       http.StatusText(status),
	}, nil
}
