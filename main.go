package main

import (
	"go-spider/tujidao"
	"log"
	"os"
)

func init() {
	file, err := os.OpenFile("logs.txt",os.O_APPEND|os.O_CREATE|os.O_WRONLY,0666)
	if err != nil {
		log.Fatalln(err)
	}
	log.SetOutput(file)
}

func main()  {
	tujidao.TujidaoSpider()
}