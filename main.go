package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

const userAgent = "GitHub-Redirector/0.1"

type githubAsset struct {
	DownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type cacheEntry struct {
	time time.Time
	url  string
	err  error
}

type config map[string]map[string]string

var urlCache = make(map[string]cacheEntry)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func fetchDownloadURL(client *fasthttp.Client, requestURL string) (downloadURL string, err error) {
	req := fasthttp.AcquireRequest()
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	req.SetRequestURI(requestURL)

	resp := fasthttp.AcquireResponse()
	err = client.Do(req, resp)
	if err != nil {
		return
	}

	var rel githubRelease
	err = json.Unmarshal(resp.Body(), &rel)
	if err != nil {
		return
	}

	if len(rel.Assets) < 1 {
		err = fmt.Errorf("Latest release '%s' has no assets", rel.TagName)
		return
	}

	downloadURL = rel.Assets[0].DownloadURL
	return
}

func getDownloadURL(client *fasthttp.Client, repo string) (assetURL string, err error) {
	// Get cache entry and use if age <= 5 min
	if entry, ok := urlCache[repo]; ok && time.Since(entry.time) <= 5*time.Minute {
		// Return cached value
		assetURL = entry.url
		err = entry.err
	} else {
		// Construct URL and fetch from GitHub
		releaseURL := "https://api.github.com/repos/" + repo + "/releases/latest"
		assetURL, err = fetchDownloadURL(client, releaseURL)

		// Commit result to cache
		urlCache[repo] = cacheEntry{
			time: time.Now(),
			url:  assetURL,
			err:  err,
		}
	}

	return
}

func releaseHandler(ctx *fasthttp.RequestCtx, client *fasthttp.Client, repo string) {
	targetURL, err := getDownloadURL(client, repo)
	check(err)

	ctx.Redirect(targetURL, fasthttp.StatusFound)
}

func getReqHandler(client *fasthttp.Client, fileMap map[string]string) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		defer func() {
			if err := recover(); err != nil {
				errStr := fmt.Sprintf("%+v", err)
				fmt.Printf("Error in request handler: %s\n", errStr)
				ctx.Error(errStr+"\n", fasthttp.StatusInternalServerError)
			}
		}()

		ctx.Response.Header.Set("Server", userAgent)

		fileKey := string(ctx.Path())[1:]
		if repo, ok := fileMap[fileKey]; ok {
			releaseHandler(ctx, client, repo)
		} else {
			ctx.Error("File not found\n", fasthttp.StatusNotFound)
		}
	}
}

func main() {
	var listenAddr string
	var configPath string
	flag.StringVar(&listenAddr, "addr", ":8947", "Address for the HTTP server to listen on (or unix:/path/to/socket.sock)")
	flag.StringVar(&configPath, "config", "config.json", "JSON configuration file to read")
	flag.Parse()

	// Read config
	configData, err := ioutil.ReadFile(configPath)
	check(err)

	var config config
	err = json.Unmarshal(configData, &config)
	check(err)

	// Create shared HTTP client
	client := &fasthttp.Client{
		Name:         userAgent,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start server
	fmt.Printf("Starting server on %s\n", listenAddr)
	reqHandler := getReqHandler(client, config["files"])

	if strings.HasPrefix(listenAddr, "unix:") {
		err = fasthttp.ListenAndServeUNIX(listenAddr[5:], 0777, reqHandler)
	} else {
		err = fasthttp.ListenAndServe(listenAddr, reqHandler)
	}

	check(err)
}
