package main

import (
	"net/http"
	"net/url"
	"github.com/PuerkitoBio/goquery"
	"regexp"
	"strings"
	"github.com/aws/aws-lambda-go/events"
	"golang.org/x/net/context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
)

var errorLogger = log.New(os.Stderr, "ERROR ", log.Llongfile)

type Pharmacy struct {
	Id      string
	Name    string   `json:"name"`
	Address string   `json:"address"`
	Lat     float64  `json:"lat"`
	Lon     float64  `json:"lon"`
	Phones  []string `json:"phones"`
}

func (p Pharmacy) String() string {
	return fmt.Sprintf("[\n%d\n%d\n%d\n%d,%d\n]",
		p.Name, p.Address, p.Phones[0], p.Lat, p.Lon)
}

func main() {
	lambda.Start(HandleRequest)
	/*
	pharmas, _ := getPharmas("piazza brembana")
	for i := range pharmas {
		pharma := pharmas[i]
		fmt.Println(pharma)
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
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(jsonPharmas),
	}, nil
}

func getPharmas(address string) ([]Pharmacy, error) {
	pharmas, err := scrape(address)
	if err != nil {
		return nil, err
	}

	done := make(chan bool)
	for i := range pharmas {
		pharma := &pharmas[i]
		go func() {
			fillLatLon(pharma)
			done <- true
		}()
	}
	for i := 0; i < len(pharmas); i++ {
		<-done
	}
	return pharmas, nil
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

	idregexp, _ := regexp.Compile("[0-9]+")
	phoneregexp, _ := regexp.Compile("[0-9]{8,15}")

	pharmas := []Pharmacy{}

	htmlDoc.Find(".sf.mnu.c").NextAll().Each(func(i int, s *goquery.Selection) {

		btag := s.Find("td.bbo b")

		name := btag.Find("a").Text()

		atag := btag.Next()
		href, _ := atag.Attr("href")
		id := idregexp.FindString(href)

		atag.Find("b").BeforeHtml("&nbsp;")

		addrs := atag.Text()
		phones := phoneregexp.FindAllString(addrs, 3)

		addrs = phoneregexp.ReplaceAllString(addrs, "")
		addrs = strings.Replace(addrs, "Tel.", "", 1)
		addrs = strings.Replace(addrs, " , ", "", 3)

		pharma := Pharmacy{id, name, addrs, 0, 0, phones}

		pharmas = append(pharmas, pharma)

	})

	return pharmas, nil
}

func fillLatLon(pharmacy *Pharmacy) {

	resp, err := http.Get("https://www.farmaciediturno.org/farmacia.asp?idf=" + pharmacy.Id)
	if err != nil {
		return
	}
	htmlDoc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return
	}

	href, exists := htmlDoc.Find("table table td a.mnu").Attr("href")
	if exists {
		hrefUrl, _ := url.Parse(href)
		latLon := strings.Split(hrefUrl.Query().Get("saddr"), ",")
		lat, _ := strconv.ParseFloat(latLon[0], 64)
		lon, _ := strconv.ParseFloat(latLon[1], 64)
		pharmacy.Lat = lat
		pharmacy.Lon = lon
	}
	return

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
