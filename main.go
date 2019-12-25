package main

import (
	"fmt"
	"os"
	"io"
	"io/ioutil"
	"errors"
	"net/http"
	"encoding/json"
	"strings"
	"flag"

	"encoding/base64"
	"crypto/md5"
	
	"github.com/avast/apkparser"
	"github.com/PuerkitoBio/goquery"
)

const WPP_APK_PATH = "whatsapp.apk"
const DEX_FILE = "classes.dex"

var DEFAULT_WPP_PAGE_URL = "https://www.cdn.whatsapp.net/android/"
var DEFAULT_RESULT_JSON = "./result.json"

type result struct {
	Url string `json:"url"`
	Version string `json:"version"`
	Hash string `json:"hash"`
}

func main() {
	var wppPageUrl string
	flag.StringVar(&wppPageUrl, "url", DEFAULT_WPP_PAGE_URL, "Put whatsapp website that holds apk download here")

	var resultFileName string
	flag.StringVar(&resultFileName, "o", DEFAULT_RESULT_JSON, "Put the output file name (json) here")
	flag.Parse()

	doc, err := GetDocument(wppPageUrl)
	if err != nil {
		panic(err)
	}

	url, err := GetApkDownloadUrl(doc)
	if err != nil {
		panic(err)
	}

	version, err := GetApkVersion(doc)
	if err != nil {
		panic(err)
	}

	hash, err := GetApkHash(url)
	if err != nil {
		panic(err)
	}

	info := result{url, version, hash}

	file, _ := json.MarshalIndent(info, "", "\t")
	_ = ioutil.WriteFile(resultFileName, file, 0644)

	fmt.Println(info)
}

func GetDocument(url string) (*goquery.Document, error) {
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func GetApkDownloadUrl(doc *goquery.Document) (string, error) {
	var apk_url string

	doc.Find(".button").Each(func(index int, item *goquery.Selection) {
		href, ok := item.Attr("href")
		if ok {
			apk_url = href
		}
	})

	if apk_url == "" {
		return "", errors.New("Can't find Apk Url")
	}

	return apk_url, nil
}

func GetApkVersion(doc *goquery.Document) (string, error) {
	var apk_version string

	doc.Find(".hint").Each(func(index int, item *goquery.Selection) {
		content := item.Contents().Text()
		if content != "" {
			slice := strings.Split(content, " ")
			apk_version = slice[len(slice) - 1]
		}
	})

	if apk_version == "" {
		return "", errors.New("Can't find Apk Version")
	}

	return apk_version, nil
}

func GetApkHash(url string) (string, error) {
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	file, err := os.Create(WPP_APK_PATH)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, res.Body)
	if err != nil {
		return "", err
	}

	ziper, err := apkparser.OpenZip(WPP_APK_PATH)
	if err != nil {
		return "", err
	}
	defer ziper.Close()

	var dex_name string
	var dex_data []byte

	for _, file := range ziper.File {
		err := file.Open()
		if err != nil {
			return "", err
		}
		defer file.Close()

		if file.Name == DEX_FILE {
			data, err := ioutil.ReadAll(file)
			if err != nil {
				return "", err
			}

			dex_name = file.Name
			dex_data = data
		}

		if err != nil {
			return "", err
		}
	}

	if dex_name == "" {
		return "", errors.New("Can't find " + DEX_FILE)
	}

	algorithm := md5.New()
	algorithm.Write(dex_data)
	sum := algorithm.Sum(nil)

	hash := base64.StdEncoding.EncodeToString(sum)
	return hash, nil
}